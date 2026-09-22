package agentruntime

import (
	"context"
	"encoding/json"
	"errors"
	"net"
	"net/url"
	"regexp"
	"strconv"
)

// LocalMCPServer is an ephemeral, host-owned runtime binding. It is NOT a
// provider setting, model input, public API payload or persisted session field.
// JSON encoding deliberately cannot export the delegated bearer.
type LocalMCPServer struct {
	Name        string           `json:"name"`
	URL         string           `json:"url"`
	BearerToken string           `json:"-"`
	Context     *LocalMCPContext `json:"context,omitempty"`
	// Host application sessions can keep repository documents available through
	// filesystem tools without automatically importing their startup instructions.
	SkipProjectInstructions bool `json:"-"`
}

// LocalMCPContext is a bounded host-owned JSON resource identifying the surface
// attached to this conversation. It carries data, not a task or tool sequence.
type LocalMCPContext struct {
	URI  string `json:"uri"`
	Text string `json:"text"`
}

var mcpName = regexp.MustCompile(`^[A-Za-z][A-Za-z0-9_]{0,47}$`)
var mcpBearer = regexp.MustCompile(`^[A-Za-z0-9_-]{32,128}$`)

const mcpBearerEnvironment = "XGC_NATIVE_MCP_BEARER"

type localMCPContextKey struct{}

func (s LocalMCPServer) validate() error {
	if s.Context != nil {
		uri, err := url.Parse(s.Context.URI)
		var object map[string]json.RawMessage
		if err != nil || uri.Scheme == "" || uri.User != nil || len(s.Context.Text) > 4096 || json.Unmarshal([]byte(s.Context.Text), &object) != nil || object == nil {
			return errors.New("invalid host-owned MCP context resource")
		}
	}
	u, err := url.Parse(s.URL)
	if err != nil || !mcpName.MatchString(s.Name) || !mcpBearer.MatchString(s.BearerToken) || u.Scheme != "http" || !net.ParseIP(u.Hostname()).IsLoopback() || u.User != nil || u.Path != "/mcp" || u.RawPath != "" || u.RawQuery != "" || u.Fragment != "" {
		return errors.New("invalid host-owned local MCP binding")
	}
	port, err := strconv.Atoi(u.Port())
	if err != nil || port < 1 || port > 65535 || strconv.Itoa(port) != u.Port() {
		return errors.New("local MCP requires an explicit canonical port")
	}
	return nil
}

func localMCPThreadContext(ctx context.Context, params map[string]any) {
	if server, ok := localMCP(ctx); ok && server.Context != nil {
		params["developerInstructions"] = server.Context.Text
	}
}

func localMCPACPContent(ctx context.Context, prompt string) []any {
	content := []any{}
	if server, ok := localMCP(ctx); ok && server.Context != nil {
		// Text content is required by ACP; embedded resources are optional.
		content = append(content, map[string]any{"type": "text", "text": server.Context.Text})
	}
	return append(content, map[string]any{"type": "text", "text": prompt})
}
func withLocalMCP(ctx context.Context, server LocalMCPServer) context.Context {
	return context.WithValue(ctx, localMCPContextKey{}, server)
}
func localMCP(ctx context.Context) (LocalMCPServer, bool) {
	server, ok := ctx.Value(localMCPContextKey{}).(LocalMCPServer)
	return server, ok
}

// Apply configuration at process launch, not only thread/start: the same
// ephemeral binding must also apply to native thread/resume. Only an environment
// variable NAME appears on argv; only the restricted bearer is added to env.
func localMCPLaunch(ctx context.Context, provider string, args []string) ([]string, []string, error) {
	server, ok := localMCP(ctx)
	if !ok {
		return args, nil, nil
	}
	if err := server.validate(); err != nil {
		return nil, nil, err
	}
	switch provider {
	case "codex":
		config := "mcp_servers." + server.Name + "={url=" + strconv.Quote(server.URL) + ",bearer_token_env_var=" + strconv.Quote(mcpBearerEnvironment) + ",enabled=true,required=true,startup_timeout_sec=15,tool_timeout_sec=20}"
		result := append([]string{"-c", config}, args...)
		if server.SkipProjectInstructions {
			result = append([]string{"-c", "project_doc_max_bytes=0"}, result...)
		}
		return result, []string{mcpBearerEnvironment + "=" + server.BearerToken}, nil
	case "cursor", "grok", "opencode":
		return args, nil, nil // ACP receives a private session-setup header.
	default:
		return nil, nil, errors.New("this provider does not support host-owned MCP bindings")
	}
}
func localMCPACPServers(ctx context.Context, initialization map[string]any) ([]any, error) {
	server, ok := localMCP(ctx)
	if !ok {
		return []any{}, nil
	}
	if err := server.validate(); err != nil {
		return nil, err
	}
	capabilities, _ := initialization["agentCapabilities"].(map[string]any)
	transports, _ := capabilities["mcpCapabilities"].(map[string]any)
	if supported, _ := transports["http"].(bool); !supported {
		return nil, errors.New("ACP provider does not advertise HTTP MCP support")
	}
	return []any{map[string]any{"type": "http", "name": server.Name, "url": server.URL, "headers": []any{map[string]string{"name": "Authorization", "value": "Bearer " + server.BearerToken}}}}, nil
}
