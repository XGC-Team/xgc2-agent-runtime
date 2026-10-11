package agentruntime

import (
	"bufio"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"testing"
)

type claudeMCPObservation struct {
	Args        []string `json:"args"`
	Name        string   `json:"name"`
	Type        string   `json:"type"`
	URL         string   `json:"url"`
	TokenDigest string   `json:"tokenDigest"`
	Registered  bool     `json:"registered"`
	// Leaks name the places the credential must never be found: the arguments and the environment.
	InArguments   bool   `json:"inArguments"`
	InEnvironment bool   `json:"inEnvironment"`
	Answer        string `json:"answer"`
}

// A real subprocess fixture plays Claude's stream-json control channel: it takes
// the host-owned MCP server over mcp_set_servers, as Claude does, and then
// exercises the native permission control channel. A server named
// xgc2_unreachable fails to connect; one named xgc2_rejected makes the CLI
// refuse the registration. It never calls a model or opens a network connection.
func fixtureClaudeMCP() {
	args := os.Args[1:]
	if slices.Contains(args, "--mcp-config") || !slices.Contains(args, "--strict-mcp-config") {
		return
	}
	native := "claude-mcp-fixture"
	for _, arg := range args {
		if strings.HasPrefix(arg, "--resume=") {
			native = strings.TrimPrefix(arg, "--resume=")
		}
	}
	report := claudeMCPObservation{Args: args}
	send := func(value any) { _ = json.NewEncoder(os.Stdout).Encode(value) }
	respond := func(id string, subtype string, body map[string]any) {
		send(map[string]any{"type": "control_response", "response": map[string]any{"subtype": subtype, "request_id": id, "response": body}})
	}
	input := map[string]any{"experimentId": "fixture-experiment", "limit": float64(1)}
	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		var message map[string]any
		if json.Unmarshal(scanner.Bytes(), &message) != nil {
			return
		}
		switch text(message, "type") {
		case "control_request":
			request := obj(message["request"])
			id := text(message, "request_id")
			switch text(request, "subtype") {
			case "initialize":
				respond(id, "success", map[string]any{})
			case "mcp_set_servers":
				servers := obj(request["servers"])
				if len(servers) != 1 {
					return
				}
				for name, value := range servers {
					server := obj(value)
					authorization := text(obj(server["headers"]), "Authorization")
					bearer := strings.TrimPrefix(authorization, "Bearer ")
					if bearer == authorization || !mcpBearer.MatchString(bearer) {
						return
					}
					digest := sha256.Sum256([]byte(bearer))
					report.Name, report.Type, report.URL, report.TokenDigest = name, text(server, "type"), text(server, "url"), hex.EncodeToString(digest[:])
					report.InArguments = strings.Contains(strings.Join(args, " "), bearer)
					report.InEnvironment = strings.Contains(strings.Join(os.Environ(), "\n"), bearer)
				}
				report.Registered = true
				switch report.Name {
				case "xgc2_rejected":
					respond(id, "error", nil)
				case "xgc2_unreachable":
					respond(id, "success", map[string]any{"added": []string{report.Name}, "removed": []string{}, "errors": map[string]any{report.Name: "ECONNREFUSED"}})
				default:
					respond(id, "success", map[string]any{"added": []string{report.Name}, "removed": []string{}, "errors": map[string]any{}})
				}
			}
		case "user":
			if !report.Registered {
				return // the prompt must wait for the registration
			}
			send(map[string]any{"type": "system", "subtype": "init", "session_id": native})
			send(map[string]any{"type": "control_request", "request_id": "mcp-permission", "request": map[string]any{"subtype": "can_use_tool", "tool_name": "mcp__" + report.Name + "__xgc2_context", "tool_use_id": "mcp-tool", "input": input}})
		case "control_response":
			response := obj(obj(message["response"])["response"])
			report.Answer = text(response, "behavior")
			if report.Answer == "allow" && !reflect.DeepEqual(obj(response["updatedInput"]), input) {
				return
			}
			encoded, _ := json.Marshal(report)
			send(map[string]any{"type": "result", "subtype": "success", "is_error": false, "session_id": native, "result": string(encoded)})
			return
		}
	}
}

func TestClaudeLocalMCPFreshAndResumeNativeContracts(t *testing.T) {
	for _, scenario := range []struct {
		name       string
		options    AgentOptions
		answer     string
		permission string
	}{
		{"legacy-narrow", AgentOptions{}, "allow", ""},
		{"manual", AgentOptions{Permission: "approval-required"}, "deny", ""},
		{"accept-edits", AgentOptions{Permission: "auto-accept-edits"}, "allow", "--permission-mode=acceptEdits"},
		{"full-access", AgentOptions{Permission: "full-access"}, "allow", "--permission-mode=bypassPermissions"},
		{"plan", AgentOptions{Permission: "plan"}, "deny", "--permission-mode=plan"},
	} {
		t.Run(scenario.name, func(t *testing.T) {
			for _, resumed := range []bool{false, true} {
				server := testLocalMCP()
				server.Context = &LocalMCPContext{URI: "xgc2://experiments/fixture", Text: `{"application":"XGC2","experiment":{"id":"fixture"}}`}
				native := ""
				if resumed {
					native = "claude-mcp-fixture"
					server.BearerToken = strings.Repeat("R", 43)
				}
				var events []Event
				var requests []Request
				driver, err := NewDriverWithLocalMCP(testProfile(t, "claude"), func(event Event) error { events = append(events, event); return nil }, func(ctx context.Context, request Request) (Answer, error) {
					requests = append(requests, request)
					return Answer{OptionID: scenario.answer}, nil
				}, server)
				if err != nil {
					t.Fatal(err)
				}
				if err = driver.Open(testNativeContext(t), t.TempDir(), native); err != nil {
					t.Fatal(err)
				}
				if scenario.options == (AgentOptions{}) {
					err = driver.Prompt(testNativeContext(t), "turn", "inspect the experiment")
				} else {
					err = driver.(optionDriver).PromptWithOptions(testNativeContext(t), "turn", "inspect the experiment", scenario.options)
				}
				_ = driver.Close()
				if err != nil {
					t.Fatal(err)
				}
				if len(requests) != 1 || requests[0].Details.ToolName != "mcp__"+server.Name+"__xgc2_context" || !strings.Contains(requests[0].Details.Preview, "fixture-experiment") {
					t.Fatalf("MCP permission lacks its arguments: %+v", requests)
				}
				var report claudeMCPObservation
				for _, event := range events {
					if event.Role == "assistant" {
						if err := json.Unmarshal([]byte(event.Text), &report); err != nil {
							t.Fatal(err)
						}
					}
				}
				if report.Answer != scenario.answer || !report.Registered {
					t.Fatalf("invalid native launch: %+v", report)
				}
				// The credential travels over the control channel only: neither the
				// process list nor the environment every agent command inherits holds it.
				if report.InArguments || report.InEnvironment || slices.Contains(report.Args, "--mcp-config") || !slices.Contains(report.Args, "--strict-mcp-config") {
					t.Fatalf("credential or configuration reached the process launch: %+v", report)
				}
				digest := sha256.Sum256([]byte(server.BearerToken))
				if report.TokenDigest != hex.EncodeToString(digest[:]) || report.Name != server.Name || report.Type != "http" || report.URL != server.URL {
					t.Fatal("native launch did not receive the current binding")
				}
				encoded, _ := json.Marshal(events)
				if strings.Contains(string(encoded), server.BearerToken) {
					t.Fatal("credential leaked to event history")
				}
				if slices.Contains(report.Args, "--resume=claude-mcp-fixture") != resumed {
					t.Fatal("native resume identity changed")
				}
				contextIndex := slices.Index(report.Args, "--append-system-prompt")
				if contextIndex < 0 || report.Args[contextIndex+1] != server.Context.Text {
					t.Fatal("native context resource missing on create/resume")
				}
				if scenario.permission != "" && !slices.Contains(report.Args, scenario.permission) {
					t.Fatal("MCP injection changed selected native permissions")
				}
				if slices.Contains(report.Args, "--allow-dangerously-skip-permissions") != (scenario.options.Permission == "full-access") {
					t.Fatal("MCP injection changed bypass permission authority")
				}
				toolsIndex := slices.Index(report.Args, "--tools")
				if scenario.options == (AgentOptions{}) {
					if toolsIndex < 0 || report.Args[toolsIndex+1] != "Read,Glob,Grep" || !slices.Contains(report.Args, "--allowedTools") {
						t.Fatal("legacy MCP launch expanded built-in tools")
					}
				} else if toolsIndex >= 0 || slices.Contains(report.Args, "--allowedTools") {
					t.Fatal("MCP injection rewrote configured native tool permissions")
				}
			}
		})
	}
}

func TestClaudeLocalMCPUnsupportedCLIAndFailedStart(t *testing.T) {
	for _, help := range []string{"--input-format", "--input-format --mcp-config", "--strict-mcp-config"} {
		path := filepath.Join(t.TempDir(), "old-claude")
		data := []byte("#!/bin/sh\nprintf '%s\\n' '" + help + "'\n")
		if err := os.WriteFile(path, data, 0700); err != nil {
			t.Fatal(err)
		}
		digest := sha256.Sum256(data)
		profile := Profile{ID: "claude", Provider: "claude", Executable: path, SHA256: hex.EncodeToString(digest[:]), ReviewedVersion: "old-fixture", BillingReviewed: true}
		driver, err := NewDriverWithLocalMCP(profile, func(Event) error { return nil }, nil, testLocalMCP())
		if err != nil {
			t.Fatal(err)
		}
		if err = driver.Open(testNativeContext(t), t.TempDir(), ""); err == nil || !strings.Contains(err.Error(), "does not support host-owned MCP") {
			t.Fatalf("unsupported CLI did not fail explicitly: %v", err)
		}
	}

	driver := &claudeDriver{profile: Profile{Provider: "claude", Executable: filepath.Join(t.TempDir(), "missing")}}
	err := driver.PromptWithOptions(withLocalMCP(context.Background(), testLocalMCP()), "turn", "unused", AgentOptions{Permission: "approval-required"})
	if err == nil {
		t.Fatal("missing native process unexpectedly started")
	}
}

func claudeMCPDriver(t *testing.T, name string, events *[]Event) optionDriver {
	t.Helper()
	server := testLocalMCP()
	server.Name = name
	driver, err := NewDriverWithLocalMCP(testProfile(t, "claude"), func(event Event) error { *events = append(*events, event); return nil }, func(context.Context, Request) (Answer, error) {
		return Answer{OptionID: "allow"}, nil
	}, server)
	if err != nil {
		t.Fatal(err)
	}
	if err = driver.Open(testNativeContext(t), t.TempDir(), ""); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = driver.Close() })
	return driver.(optionDriver)
}

func TestClaudeLocalMCPServerThatCannotConnectLeavesTheTurnWithoutItsTools(t *testing.T) {
	var events []Event
	driver := claudeMCPDriver(t, "xgc2_unreachable", &events)
	if err := driver.PromptWithOptions(testNativeContext(t), "turn", "inspect the experiment", AgentOptions{Permission: "approval-required"}); err != nil {
		t.Fatal(err)
	}
	var notice *Event
	for i := range events {
		if events[i].Kind == "notice" && strings.Contains(events[i].Text, "MCP server could not be reached") {
			notice = &events[i]
		}
	}
	if notice == nil {
		t.Fatalf("an unreachable MCP server went unnoticed: %+v", events)
	}
	encoded, _ := json.Marshal(events)
	if strings.Contains(string(encoded), "ECONNREFUSED") || strings.Contains(string(encoded), strings.Repeat("T", 43)) {
		t.Fatal("the native error text or the credential reached the event history")
	}
}

func TestClaudeLocalMCPRejectedRegistrationFailsTheTurnBeforeThePrompt(t *testing.T) {
	var events []Event
	driver := claudeMCPDriver(t, "xgc2_rejected", &events)
	err := driver.PromptWithOptions(testNativeContext(t), "turn", "inspect the experiment", AgentOptions{Permission: "approval-required"})
	if err == nil || !strings.Contains(err.Error(), "rejected the registration") {
		t.Fatalf("a refused registration did not fail the turn: %v", err)
	}
	for _, event := range events {
		if event.Role == "assistant" || event.Kind == "turn.end" {
			t.Fatalf("the prompt was sent although the host's MCP server was refused: %+v", event)
		}
	}
}

func TestClaudeUnboundMCPLaunchPreservesArgs(t *testing.T) {
	args := []string{"--print", "--permission-mode=plan"}
	actual, servers, err := claudeLocalMCPLaunch(context.Background(), args)
	if err != nil || servers != nil || !reflect.DeepEqual(args, actual) {
		t.Fatalf("changed unbound native launch: %v", err)
	}
}
