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
	Args        []string       `json:"args"`
	Config      map[string]any `json:"config"`
	Path        string         `json:"path"`
	TokenDigest string         `json:"tokenDigest"`
	FileMode    uint32         `json:"fileMode"`
	DirMode     uint32         `json:"dirMode"`
	Answer      string         `json:"answer"`
}

// A real subprocess fixture consumes the same private file and delegated env
// as Claude, then exercises its native permission control channel. It never
// calls a model or opens a network connection.
func fixtureClaudeMCP() {
	args := os.Args[1:]
	index := slices.Index(args, "--mcp-config")
	if index < 0 || index+1 >= len(args) || !slices.Contains(args, "--strict-mcp-config") {
		return
	}
	path := args[index+1]
	data, err := os.ReadFile(path)
	if err != nil {
		return
	}
	var config map[string]any
	if json.Unmarshal(data, &config) != nil {
		return
	}
	servers := obj(config["mcpServers"])
	if len(servers) != 1 {
		return
	}
	name := ""
	for key, value := range servers {
		name = key
		server := obj(value)
		if text(server, "type") != "http" || text(obj(server["headers"]), "Authorization") != "Bearer ${"+mcpBearerEnvironment+"}" {
			return
		}
	}
	bearer := os.Getenv(mcpBearerEnvironment)
	if !mcpBearer.MatchString(bearer) || strings.Contains(string(data), bearer) || strings.Contains(strings.Join(args, " "), bearer) {
		return
	}
	info, err := os.Stat(path)
	if err != nil {
		return
	}
	directory, err := os.Stat(filepath.Dir(path))
	if err != nil {
		return
	}
	digest := sha256.Sum256([]byte(bearer))
	report := claudeMCPObservation{Args: args, Config: config, Path: path, TokenDigest: hex.EncodeToString(digest[:]), FileMode: uint32(info.Mode().Perm()), DirMode: uint32(directory.Mode().Perm())}
	native := "claude-mcp-fixture"
	for _, arg := range args {
		if strings.HasPrefix(arg, "--resume=") {
			native = strings.TrimPrefix(arg, "--resume=")
		}
	}
	send := func(value any) { _ = json.NewEncoder(os.Stdout).Encode(value) }
	input := map[string]any{"experimentId": "fixture-experiment", "limit": float64(1)}
	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		var message map[string]any
		if json.Unmarshal(scanner.Bytes(), &message) != nil {
			return
		}
		switch text(message, "type") {
		case "control_request":
			send(map[string]any{"type": "control_response", "response": map[string]any{"subtype": "success", "request_id": "xgc-initialize", "response": map[string]any{}}})
		case "user":
			send(map[string]any{"type": "system", "subtype": "init", "session_id": native})
			send(map[string]any{"type": "control_request", "request_id": "mcp-permission", "request": map[string]any{"subtype": "can_use_tool", "tool_name": "mcp__" + name + "__xgc2_context", "tool_use_id": "mcp-tool", "input": input}})
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
			previousPath := ""
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
				if err = driver.Open(context.Background(), t.TempDir(), native); err != nil {
					t.Fatal(err)
				}
				if scenario.options == (AgentOptions{}) {
					err = driver.Prompt(context.Background(), "turn", "inspect the experiment")
				} else {
					err = driver.(optionDriver).PromptWithOptions(context.Background(), "turn", "inspect the experiment", scenario.options)
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
				if report.Answer != scenario.answer || report.FileMode != 0600 || report.DirMode != 0700 {
					t.Fatalf("invalid private native launch: %+v", report)
				}
				if _, err := os.Stat(filepath.Dir(report.Path)); !os.IsNotExist(err) || report.Path == previousPath {
					t.Fatal("private runtime configuration retained or reused")
				}
				previousPath = report.Path
				digest := sha256.Sum256([]byte(server.BearerToken))
				if report.TokenDigest != hex.EncodeToString(digest[:]) || text(obj(obj(report.Config["mcpServers"])[server.Name]), "url") != server.URL {
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

func TestClaudeLocalMCPUnsupportedCLIAndFailedStartCleanup(t *testing.T) {
	for _, help := range []string{"--input-format", "--input-format --mcp-config", "--mcp-config --strict-mcp-config"} {
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
		if err = driver.Open(context.Background(), t.TempDir(), ""); err == nil || !strings.Contains(err.Error(), "does not support host-owned MCP") {
			t.Fatalf("unsupported CLI did not fail explicitly: %v", err)
		}
	}

	temporary := t.TempDir()
	t.Setenv("TMPDIR", temporary)
	driver := &claudeDriver{profile: Profile{Provider: "claude", Executable: filepath.Join(t.TempDir(), "missing")}}
	err := driver.PromptWithOptions(withLocalMCP(context.Background(), testLocalMCP()), "turn", "unused", AgentOptions{Permission: "approval-required"})
	if err == nil {
		t.Fatal("missing native process unexpectedly started")
	}
	entries, err := os.ReadDir(temporary)
	if err != nil || len(entries) != 0 {
		t.Fatal("failed launch retained private MCP configuration")
	}
}

func TestClaudeUnboundMCPLaunchPreservesArgs(t *testing.T) {
	args := []string{"--print", "--permission-mode=plan"}
	actual, environment, cleanup, err := claudeLocalMCPLaunch(context.Background(), args)
	cleanup()
	if err != nil || len(environment) != 0 || !reflect.DeepEqual(args, actual) {
		t.Fatalf("changed unbound native launch: %v", err)
	}
}
