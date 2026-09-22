package agentruntime

import (
	"context"
	"encoding/json"
	"reflect"
	"strings"
	"testing"
)

func testLocalMCP() LocalMCPServer {
	return LocalMCPServer{Name: "xgc2_session_123", URL: "http://127.0.0.1:4567/mcp", BearerToken: strings.Repeat("T", 43)}
}

func TestHostContextKeepsUserTextAndPermissionsSeparate(t *testing.T) {
	server := testLocalMCP()
	server.Context = &LocalMCPContext{URI: "xgc2://experiments/fixture", Text: `{"application":"XGC2","experiment":{"id":"fixture"}}`}
	ctx := withLocalMCP(context.Background(), server)
	for _, method := range []string{"thread/start", "thread/resume"} {
		t.Run(method, func(t *testing.T) {
			params := map[string]any{"cwd": "/reviewed", "approvalPolicy": "never"}
			localMCPThreadContext(ctx, params)
			if params["developerInstructions"] != server.Context.Text || params["approvalPolicy"] != "never" || params["cwd"] != "/reviewed" {
				t.Fatal("context changed native scope or permission")
			}
		})
	}
	content := localMCPACPContent(ctx, "original user message")
	if len(content) != 2 || text(obj(content[0]), "text") != server.Context.Text || text(obj(content[1]), "text") != "original user message" {
		t.Fatal("context replaced the user message")
	}
	params := map[string]any{}
	localMCPThreadContext(context.Background(), params)
	if len(params) != 0 || len(localMCPACPContent(context.Background(), "original")) != 1 {
		t.Fatal("unbound conversation gained host context")
	}
	for _, invalid := range []string{`null`, `[]`, `"instructions"`, strings.Repeat("x", 4097)} {
		server.Context.Text = invalid
		if server.validate() == nil {
			t.Fatal("accepted non-object or unbounded context")
		}
	}
}
func TestLocalMCPValidationAndSecretSerialization(t *testing.T) {
	good := testLocalMCP()
	if err := good.validate(); err != nil {
		t.Fatal(err)
	}
	encoded, _ := json.Marshal(good)
	if strings.Contains(string(encoded), good.BearerToken) {
		t.Fatal("bearer serialized")
	}
	for _, endpoint := range []string{"https://example.com/mcp", "http://localhost:4567/mcp", "http://127.0.0.1:4567/settings", "http://127.0.0.1:4567/mcp?token=x", "http://u:p@127.0.0.1:4567/mcp"} {
		value := good
		value.URL = endpoint
		if value.validate() == nil {
			t.Errorf("accepted %s", endpoint)
		}
	}
	bad := good
	bad.Name = "a.b"
	if bad.validate() == nil {
		t.Fatal("TOML path injection")
	}
	bad = good
	bad.BearerToken = "x\nAuthorization: other"
	if bad.validate() == nil {
		t.Fatal("header injection")
	}
}
func TestHostProjectInstructionsAreAnExplicitScopedLaunchChoice(t *testing.T) {
	server := testLocalMCP()
	original := []string{"app-server"}
	args, _, err := localMCPLaunch(withLocalMCP(context.Background(), server), "codex", original)
	if err != nil || strings.Contains(strings.Join(args, " "), "project_doc_max_bytes") {
		t.Fatal("ordinary repository session changed")
	}
	server.SkipProjectInstructions = true
	args, _, err = localMCPLaunch(withLocalMCP(context.Background(), server), "codex", original)
	if err != nil || !strings.Contains(strings.Join(args, " "), "project_doc_max_bytes=0") {
		t.Fatal("host boundary missing")
	}
	if !reflect.DeepEqual(original, []string{"app-server"}) {
		t.Fatal("caller arguments mutated")
	}
}
func TestCodexLocalMCPLaunchContainsOnlyCredentialReference(t *testing.T) {
	server := testLocalMCP()
	ctx := withLocalMCP(context.Background(), server)
	original := []string{"-c", `model_provider="openai"`, "app-server"}
	args, env, err := localMCPLaunch(ctx, "codex", original)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(strings.Join(args, " "), server.BearerToken) {
		t.Fatal("bearer leaked to argv")
	}
	if !strings.Contains(args[1], "bearer_token_env_var") || !strings.Contains(args[1], server.URL) || !strings.Contains(args[1], "required=true") || len(env) != 1 || env[0] != mcpBearerEnvironment+"="+server.BearerToken {
		t.Fatal("incomplete scoped launch")
	}
	if !reflect.DeepEqual(original, []string{"-c", `model_provider="openai"`, "app-server"}) {
		t.Fatal("mutated caller arguments")
	}
	// Launch overrides are identical for start/resume; credentials are minted by
	// the host, not recovered from a provider's transcript or configuration file.
	other := server
	other.BearerToken = strings.Repeat("U", 43)
	_, resumed, err := localMCPLaunch(withLocalMCP(context.Background(), other), "codex", original)
	if err != nil || resumed[0] == env[0] {
		t.Fatal("resume retained stale bearer")
	}
}
func TestACPRequiresAdvertisedTransport(t *testing.T) {
	server := testLocalMCP()
	ctx := withLocalMCP(context.Background(), server)
	for _, init := range []map[string]any{{}, {"agentCapabilities": map[string]any{"mcpCapabilities": map[string]any{"http": false}}}} {
		if _, err := localMCPACPServers(ctx, init); err == nil {
			t.Fatal("unsupported MCP silently omitted")
		}
	}
	init := map[string]any{"agentCapabilities": map[string]any{"mcpCapabilities": map[string]any{"http": true}}}
	servers, err := localMCPACPServers(ctx, init)
	if err != nil || len(servers) != 1 {
		t.Fatal(err)
	}
	encoded, _ := json.Marshal(servers)
	if !strings.Contains(string(encoded), "Bearer "+server.BearerToken) {
		t.Fatal("private authenticated session setup missing")
	}
	for _, provider := range []string{"cursor", "grok", "opencode"} {
		_, env, err := localMCPLaunch(ctx, provider, []string{"acp"})
		if err != nil || len(env) != 0 {
			t.Fatal("ACP launch needlessly exported bearer environment")
		}
	}
}
func TestNoMCPPreservesExistingNativeLaunch(t *testing.T) {
	original := []string{"acp"}
	args, env, err := localMCPLaunch(context.Background(), "cursor", original)
	if err != nil || len(env) != 0 || !reflect.DeepEqual(args, original) {
		t.Fatal("changed unconfigured native launch")
	}
	servers, err := localMCPACPServers(context.Background(), nil)
	if err != nil || len(servers) != 0 {
		t.Fatal("changed unconfigured ACP setup")
	}
}
