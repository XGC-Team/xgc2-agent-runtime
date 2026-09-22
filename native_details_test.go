package nativeagent

import (
	"context"
	"encoding/json"
	"os"
	"strings"
	"testing"
	"time"
)

func TestCodexStructuredToolDetailsAndNativeIdentity(t *testing.T) {
	events := []Event{}
	d := &rpcDriver{profile: Profile{Provider: "codex"}, turn: "local-turn", nativeTurn: "native-turn", nativeSession: "native-thread", sink: func(e Event) error { events = append(events, e); return nil }}
	d.onNotification("item/completed", map[string]any{"threadId": "native-thread", "turnId": "native-turn", "item": map[string]any{
		"id": "command-one", "type": "commandExecution", "command": "rg experiment src", "cwd": "/reviewed/workspace", "exitCode": float64(2), "durationMs": float64(17), "status": "completed", "aggregatedOutput": "no matches",
		"commandActions": []any{map[string]any{"type": "search", "command": "rg experiment src", "query": "experiment", "path": "src"}}, "environment": map[string]any{"AUTH_TOKEN": "must-not-cross"},
	}})
	if len(events) != 1 {
		t.Fatal(events)
	}
	event := events[0]
	if event.TurnID != "local-turn" || event.NativeTurnID != "native-turn" || event.NativeThreadID != "native-thread" || event.SourceMethod != "item/completed" {
		t.Fatalf("identity: %+v", event)
	}
	if event.Details["cwd"] != "/reviewed/workspace" || event.Details["exitCode"] != float64(2) || event.Details["durationMs"] != float64(17) || event.Text != "no matches" {
		t.Fatalf("details: %+v", event)
	}
	encoded, _ := json.Marshal(event)
	if strings.Contains(string(encoded), "must-not-cross") || strings.Contains(string(encoded), "environment") {
		t.Fatal("unlisted native field escaped")
	}
}
func TestCodexStructuredFileChangeKeepsMoveAndKind(t *testing.T) {
	details := codexItemDetails(map[string]any{"type": "fileChange", "changes": []any{map[string]any{"path": "old.ts", "kind": map[string]any{"type": "update", "move_path": "new.ts"}, "diff": "-old\n+new"}}})
	change := obj(arr(details["changes"])[0])
	if text(change, "path") != "old.ts" || text(obj(change["kind"]), "movePath") != "new.ts" || text(obj(change["kind"]), "type") != "update" || text(change, "diff") != "-old\n+new" {
		t.Fatal(details)
	}
}
func TestCodexMCPMetadataIsBoundedAndDoesNotCopyAuthOrBinaryFrames(t *testing.T) {
	details := codexItemDetails(map[string]any{"type": "mcpToolCall", "server": "docs", "tool": "read", "arguments": map[string]any{"path": "guide.md", "authorization": "secret-argument", "nested": map[string]any{"api_key": "secret-key", "query": "read this"}}, "result": map[string]any{
		"_meta": map[string]any{"auth": "secret-meta"}, "content": []any{map[string]any{"type": "text", "text": "body"}, map[string]any{"type": "image", "data": "binary-body"}}, "structuredContent": map[string]any{"title": "Guide", "headers": map[string]any{"Cookie": "secret-cookie"}},
	}})
	encoded, _ := json.Marshal(details)
	for _, forbidden := range []string{"secret-argument", "secret-key", "secret-meta", "secret-cookie", "binary-body", "authorization", "_meta"} {
		if strings.Contains(string(encoded), forbidden) {
			t.Fatalf("unreviewed field %s: %s", forbidden, encoded)
		}
	}
	if text(obj(details["arguments"]), "path") != "guide.md" || details["truncated"] != true {
		t.Fatal(details)
	}
	large := codexItemDetails(map[string]any{"type": "mcpToolCall", "server": "fixture", "tool": "read", "arguments": map[string]any{"body": strings.Repeat("a", 100000)}})
	if len(text(obj(large["arguments"]), "body")) != 16<<10 || large["truncated"] != true {
		t.Fatal("tool metadata was not bounded")
	}
	if codexItemDetails(map[string]any{"type": "reasoning", "content": []any{"private reasoning"}}) != nil {
		t.Fatal("raw reasoning was retained")
	}
}
func TestCodexApprovalKeepsSourceAndQuestionSemanticsWithoutWideningAuthority(t *testing.T) {
	var seen Request
	d := &rpcDriver{profile: Profile{Provider: "codex"}, turn: "local-turn", nativeSession: "native-thread", nativeTurn: "native-turn", turnContext: context.Background(), ask: func(_ context.Context, r Request) (Answer, error) { seen = r; return Answer{OptionID: "accept"}, nil }}
	params := map[string]any{"threadId": "native-thread", "turnId": "native-turn", "itemId": "tool-one", "command": "read source", "cwd": "/reviewed/workspace", "reason": "Inspect current experiment", "availableDecisions": []any{"accept", "decline", "acceptForSession"}}
	if _, err := d.onRequest(context.Background(), "item/commandExecution/requestApproval", params); err != nil {
		t.Fatal(err)
	}
	if seen.NativeItemID != "tool-one" || seen.NativeTurnID != "native-turn" || seen.NativeThreadID != "native-thread" || seen.SourceMethod != "item/commandExecution/requestApproval" || seen.Details.Cwd != "/reviewed/workspace" {
		t.Fatalf("request: %+v", seen)
	}
	if len(seen.Options) != 2 || seen.ValidateAnswer(Answer{OptionID: "acceptForSession"}) == nil {
		t.Fatal("permission was widened")
	}
	d.ask = func(_ context.Context, r Request) (Answer, error) {
		seen = r
		return Answer{Answers: map[string][]string{"scope": {"One robot"}}}, nil
	}
	question := map[string]any{"id": "scope", "header": "Debug scope", "question": "Which robot?", "isOther": true, "options": []any{map[string]any{"label": "One robot", "description": "Keep the debug scope narrow"}}}
	params["questions"] = []any{question}
	if _, err := d.onRequest(context.Background(), "item/tool/requestUserInput", params); err != nil {
		t.Fatal(err)
	}
	if seen.Questions[0].Header != "Debug scope" || seen.Questions[0].Options[0].Description != "Keep the debug scope narrow" || !seen.Questions[0].FreeText {
		t.Fatal(seen)
	}
	question["isSecret"] = true
	if _, err := d.onRequest(context.Background(), "item/tool/requestUserInput", params); err == nil {
		t.Fatal("secret-input boundary changed")
	}
}
func TestNativeEventTimeIsPersistedReceiptTime(t *testing.T) {
	file, err := os.CreateTemp(t.TempDir(), "event-*.jsonl")
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	session := newSession(Session{ID: "s_time", Provider: "codex"}, file)
	start := time.Now().UTC()
	if err := session.appendLocked(Event{Kind: "notice", Text: "fixture", CreatedAt: "2000-01-01T00:00:00Z"}); err != nil {
		t.Fatal(err)
	}
	receipt, err := time.Parse(time.RFC3339Nano, session.events[0].CreatedAt)
	if err != nil || receipt.Before(start) || receipt.After(time.Now().UTC()) {
		t.Fatalf("receipt=%v err=%v", receipt, err)
	}
}
