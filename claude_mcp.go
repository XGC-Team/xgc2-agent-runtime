package agentruntime

import (
	"context"
	"errors"
	"strings"
	"time"
)

func checkClaudeLocalMCP(ctx context.Context, profile Profile) error {
	server, bound := localMCP(ctx)
	if !bound {
		return nil
	}
	if err := server.validate(); err != nil {
		return err
	}
	help, err := cliOutput(ctx, profile, "--help")
	if err != nil {
		return errors.New("cannot verify Claude support for host-owned MCP configuration")
	}
	for _, flag := range []string{"--strict-mcp-config", "--input-format"} {
		if !strings.Contains(string(help), flag) {
			return errors.New("installed Claude CLI does not support host-owned MCP configuration: missing " + flag)
		}
	}
	if server.Context != nil && !strings.Contains(string(help), "--append-system-prompt") {
		return errors.New("installed Claude CLI does not support host-owned context resources")
	}
	return nil
}

// claudeMCPRegistrationTimeout bounds the wait for Claude's answer to the MCP
// registration. Claude answers once the server is connected or its own start-up
// timeout (30 s) has passed, and the control channel has no deadline of its own.
const claudeMCPRegistrationTimeout = 90 * time.Second

// claudeLocalMCPLaunch prepares a native process for the host-owned MCP server.
// A configuration file with an environment reference keeps the credential out of
// the argument list, but the reference is expanded from an environment that
// every command the agent runs inherits. The server therefore goes over the
// stdin control channel once the process has started (claudeMCPRegistration);
// the launch itself only excludes ambient MCP configuration, so neither the
// arguments, the environment nor a file of the process holds the credential. The
// registration is repeated on every launch, including --resume, so revoked
// credentials are never recovered from the provider's transcript or from a
// previous runtime.
func claudeLocalMCPLaunch(ctx context.Context, args []string) ([]string, map[string]any, error) {
	server, bound := localMCP(ctx)
	if !bound {
		return args, nil, nil
	}
	if err := server.validate(); err != nil {
		return nil, nil, err
	}
	result := append(append([]string{}, args...), "--strict-mcp-config")
	if server.Context != nil {
		result = append(result, "--append-system-prompt", server.Context.Text)
	}
	servers := map[string]any{server.Name: map[string]any{
		"type": "http", "url": server.URL,
		"headers": map[string]string{"Authorization": "Bearer " + server.BearerToken},
	}}
	return result, servers, nil
}

// claudeMCPRegistration is the control request that gives a started process its
// MCP servers.
func claudeMCPRegistration(servers map[string]any) map[string]any {
	return map[string]any{"type": "control_request", "request_id": claudeMCPRequest, "request": map[string]any{"subtype": "mcp_set_servers", "servers": servers}}
}

const claudeMCPRequest = "xgc-mcp"
