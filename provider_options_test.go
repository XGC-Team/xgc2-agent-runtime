package agentruntime

import (
	"context"
	"encoding/json"
	"io"
	"strings"
	"testing"
	"time"
)

func TestClaudeFixedCatalogVersionAndNativeEffortMapping(t *testing.T) {
	old := claudeCatalog("2.1.100")
	current := claudeCatalog("2.1.261")
	has := func(models []Model, id string) bool {
		for _, m := range models {
			if m.ID == id {
				return true
			}
		}
		return false
	}
	if has(old, "claude-fable-5-1") || !has(current, "claude-fable-5-1") {
		t.Fatal("fixed upstream minVersion was ignored")
	}
	for _, m := range current {
		for _, e := range m.Efforts {
			if e.ID == "ultracode" || e.ID == "ultrathink" {
				t.Fatal("prompt-injected orchestration mode escaped scope")
			}
		}
	}
	args, err := claudeOptionArgs(AgentOptions{Model: "claude-opus-4-7", Effort: "xhigh", Permission: "auto-accept-edits"})
	if err != nil {
		t.Fatal(err)
	}
	joined := strings.Join(args, " ")
	if !strings.Contains(joined, "--model=claude-opus-4-7") || !strings.Contains(joined, "--effort=max") || !strings.Contains(joined, "--permission-mode=acceptEdits") || strings.Contains(joined, "fallback") {
		t.Fatalf("native CLI args: %v", args)
	}
}
func TestClaudeRealSubprocessControlChannelAllowAndDeny(t *testing.T) {
	for _, decision := range []string{"allow", "deny"} {
		t.Run(decision, func(t *testing.T) {
			events := []Event{}
			asked := false
			driver, err := NewDriver(testProfile(t, "claude"), func(e Event) error { events = append(events, e); return nil }, func(ctx context.Context, r Request) (Answer, error) {
				asked = true
				if r.Kind != "permission" || r.AgentItemID != "tool-1" || r.Details.Command != "fixture-only" {
					t.Errorf("lost native request: %+v", r)
				}
				return Answer{OptionID: decision}, nil
			})
			if err != nil {
				t.Fatal(err)
			}
			defer driver.Close()
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			if err = driver.Open(ctx, t.TempDir(), ""); err != nil {
				t.Fatal(err)
			}
			err = driver.(optionDriver).PromptWithOptions(ctx, "turn", "fixture", AgentOptions{Model: "claude-sonnet-5", Effort: "high", Permission: "auto-accept-edits"})
			if err != nil {
				t.Fatal(err)
			}
			expected := "denied"
			if decision == "allow" {
				expected = "allowed"
			}
			textValue := ""
			completed := false
			for _, e := range events {
				if e.Kind == "item.delta" {
					textValue += e.Text
				}
				completed = completed || e.Kind == "turn.end" && e.Status == "completed"
			}
			if !asked || textValue != expected || !completed {
				t.Fatalf("asked=%v text=%s complete=%v", asked, textValue, completed)
			}
		})
	}
}
func TestACPAdvertisedModelsEffortsAndModesRemainExact(t *testing.T) {
	var setup map[string]any
	if err := json.Unmarshal([]byte(`{"models":{"currentModelId":"native/model","availableModels":[{"modelId":"native/model","name":"Native model"}]},"configOptions":[{"id":"thinking_level","name":"Reasoning effort","category":"thought_level","type":"select","currentValue":"balanced","options":[{"value":"balanced","name":"Balanced"},{"value":"deep","name":"Deep"}]}],"modes":{"currentModeId":"ask-native","availableModes":[{"id":"ask-native","name":"Ask","description":"Provider policy"}]}}`), &setup); err != nil {
		t.Fatal(err)
	}
	setting := emptySetting(Profile{ID: "cursor", Provider: "cursor", Disabled: true})
	parseACPInventory(&setting, setup)
	if len(setting.Models) != 1 || len(setting.Models[0].Efforts) != 2 || setting.Defaults != (AgentOptions{Model: "native/model", Effort: "balanced", Permission: "ask-native"}) {
		t.Fatalf("native descriptors flattened: %+v", setting)
	}
	if validateOptions(AgentOptions{Model: "native/model", Effort: "deep", Permission: "ask-native"}, setting) != nil {
		t.Fatal("real native options rejected")
	}
	if validateOptions(AgentOptions{Model: "native/model", Effort: "xhigh"}, setting) == nil || validateOptions(AgentOptions{Permission: "full-access"}, setting) == nil {
		t.Fatal("Codex choices leaked into ACP")
	}
}

func TestPassiveProviderNotificationsDoNotBecomeOperatorMessages(t *testing.T) {
	// Preserve the 10 passive method occurrences captured in the authorized
	// Grok HTTP/SSE smoke, mixed with real messages/tool updates/error/approval.
	methods := []string{"_x.ai/session_notification", "_x.ai/sessions/changed", "_x.ai/queue/changed", "_x.ai/queue/changed", "_x.ai/models/update", "_x.ai/session_notification", "_x.ai/queue/changed", "_x.ai/session_notification", "_x.ai/session/prompt_complete", "_x.ai/sessions/changed"}
	events := []Event{}
	asked := 0
	d := &rpcDriver{profile: Profile{Provider: "grok"}, turn: "t", nativeSession: "n", turnContext: context.Background(), sink: func(e Event) error { events = append(events, e); return nil }, ask: func(_ context.Context, r Request) (Answer, error) {
		asked++
		return Answer{OptionID: r.Options[0].ID}, nil
	}}
	for _, method := range methods {
		d.onNotification(method, map[string]any{"sessionId": "n"})
	}
	for _, kind := range []string{"usage_update", "config_option_update", "future_passive_update"} {
		d.onNotification("session/update", map[string]any{"sessionId": "n", "update": map[string]any{"sessionUpdate": kind}})
	}
	if len(events) != 0 {
		t.Fatalf("passive telemetry reached chat: %+v", events)
	}
	d.onNotification("session/update", map[string]any{"sessionId": "n", "update": map[string]any{"sessionUpdate": "agent_message_chunk", "content": map[string]any{"type": "text", "text": "reply"}}})
	d.onNotification("session/update", map[string]any{"sessionId": "n", "update": map[string]any{"sessionUpdate": "tool_call", "toolCallId": "tool", "title": "Inspect", "status": "running"}})
	d.onNotification("error", map[string]any{"sessionId": "n"})
	response, err := d.onRequest(context.Background(), "session/request_permission", map[string]any{"sessionId": "n", "toolCall": map[string]any{"title": "Inspect"}, "options": []any{map[string]any{"optionId": "allow", "name": "Allow once", "kind": "allow_once"}}})
	if err != nil || asked != 1 || text(obj(obj(response)["outcome"]), "optionId") != "allow" {
		t.Fatalf("approval lost: %+v %v", response, err)
	}
	if len(events) != 3 || events[0].Role != "assistant" || events[1].Role != "tool" || events[2].Status != "error" {
		t.Fatalf("operator facts lost: %+v", events)
	}
	if _, err = d.onRequest(context.Background(), "unknown/respond-needed", map[string]any{}); err == nil || len(events) != 4 || events[3].Status != "blocked" || strings.Contains(events[3].Text, "unknown/respond-needed") {
		t.Fatal("unknown request was swallowed or exposed protocol plumbing")
	}
	codex := &rpcDriver{profile: Profile{Provider: "codex"}, turn: "t", nativeSession: "n", sink: func(e Event) error { t.Errorf("passive Codex telemetry: %+v", e); return nil }}
	for _, method := range []string{"hook/started", "hook/completed", "account/rateLimits/updated", "thread/tokenUsage/updated", "future/changed"} {
		codex.onNotification(method, map[string]any{"threadId": "n"})
	}
	decoder := claudeDecoder{turn: "t", blocks: map[int]claudeBlock{}, sink: func(e Event) error { t.Errorf("passive Claude telemetry: %+v", e); return nil }}
	if err = decoder.consume(map[string]any{"type": "rate_limit_event"}); err != nil {
		t.Fatal(err)
	}
}

func TestOpenCodeConfigOnlyModesAndRealVariantSubset(t *testing.T) {
	var setup map[string]any
	if err := json.Unmarshal([]byte(`{"configOptions":[{"id":"model","type":"select","category":"model","currentValue":"vendor/model","options":[{"value":"vendor/model","name":"Model"}]},{"id":"mode","type":"select","category":"mode","currentValue":"plan","options":[{"value":"build","name":"Build"},{"value":"plan","name":"Plan"}]},{"id":"effort","type":"select","category":"thought_level","currentValue":"high","options":[{"value":"low","name":"Low"},{"value":"high","name":"High"}]}]}`), &setup); err != nil {
		t.Fatal(err)
	}
	row := emptySetting(Profile{Provider: "opencode"})
	parseACPInventory(&row, setup)
	if len(row.Permissions) != 2 || row.Defaults != (AgentOptions{Model: "vendor/model", Effort: "high", Permission: "plan"}) {
		t.Fatalf("config-only capability lost: %+v", row)
	}
	variants := openCodeVariants("vendor/model\n{\"variants\":{\"high\":{\"headers\":\"private\"},\"low\":{}},\"options\":{\"apiKey\":\"never-return\"}}\nvendor/plain\n{\"variants\":{}}\n")
	encoded, _ := json.Marshal(variants)
	if len(variants["vendor/model"]) != 2 || len(variants["vendor/plain"]) != 0 || strings.Contains(string(encoded), "private") || strings.Contains(string(encoded), "never-return") {
		t.Fatalf("variant whitelist: %s", encoded)
	}
}

func TestClaudeQuestionControlPreservesNativeQuestionKeysAndAnswers(t *testing.T) {
	reader, writer := io.Pipe()
	defer reader.Close()
	defer writer.Close()
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	control := &claudeControl{ctx: ctx, input: writer, native: "native-thread", pending: map[string]context.CancelFunc{}, ask: func(_ context.Context, r Request) (Answer, error) {
		if r.Kind != "question" || r.AgentThreadID != "native-thread" || len(r.Questions) != 1 || r.Questions[0].ID != "Which checks?" || !r.Questions[0].Multiple || r.Questions[0].Options[0].Description != "First check" {
			t.Errorf("native question flattened: %+v", r)
		}
		return Answer{Answers: map[string][]string{"Which checks?": {"A", "B"}}}, nil
	}}
	defer control.close()
	control.request(map[string]any{"request_id": "r", "request": map[string]any{"subtype": "can_use_tool", "tool_name": "AskUserQuestion", "tool_use_id": "tool", "input": map[string]any{"questions": []any{map[string]any{"question": "Which checks?", "header": "Checks", "multiSelect": true, "options": []any{map[string]any{"label": "A", "description": "First check"}, map[string]any{"label": "B"}}}}}}})
	var response map[string]any
	if err := json.NewDecoder(reader).Decode(&response); err != nil {
		t.Fatal(err)
	}
	decision := obj(obj(response["response"])["response"])
	if text(decision, "behavior") != "allow" || text(obj(obj(decision["updatedInput"])["answers"]), "Which checks?") != "A, B" {
		t.Fatalf("native answer mapping: %+v", decision)
	}
}
