package nativeagent

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
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
		return errors.New("cannot verify native Claude support for host-owned MCP configuration")
	}
	for _, flag := range []string{"--mcp-config", "--strict-mcp-config", "--input-format"} {
		if !strings.Contains(string(help), flag) {
			return errors.New("installed Claude CLI does not support host-owned MCP configuration: missing " + flag)
		}
	}
	if server.Context != nil && !strings.Contains(string(help), "--append-system-prompt") {
		return errors.New("installed Claude CLI does not support host-owned context resources")
	}
	return nil
}

// Claude accepts --mcp-config files and expands environment variables in HTTP
// headers. Each native process gets a private, temporary file containing only a
// credential reference. Neither argv nor the config contains the bearer. Strict
// loading excludes ambient MCP configuration; no user settings are modified.
// Create this on every launch, including --resume, so revoked credentials are
// never recovered from the provider's transcript or a previous runtime file.
func claudeLocalMCPLaunch(ctx context.Context, args []string) ([]string, []string, func(), error) {
	noop := func() {}
	server, bound := localMCP(ctx)
	if !bound {
		return args, nil, noop, nil
	}
	if err := server.validate(); err != nil {
		return nil, nil, noop, err
	}
	directory, err := os.MkdirTemp("", "xgc-native-claude-mcp-")
	if err != nil {
		return nil, nil, noop, errors.New("cannot create private Claude MCP configuration")
	}
	cleanup := func() { _ = os.RemoveAll(directory) }
	config := map[string]any{"mcpServers": map[string]any{server.Name: map[string]any{
		"type": "http", "url": server.URL,
		"headers": map[string]string{"Authorization": "Bearer ${" + mcpBearerEnvironment + "}"},
	}}}
	data, err := json.Marshal(config)
	if err != nil {
		cleanup()
		return nil, nil, noop, errors.New("cannot encode private Claude MCP configuration")
	}
	path := filepath.Join(directory, "mcp.json")
	if err = os.WriteFile(path, data, 0600); err != nil {
		cleanup()
		return nil, nil, noop, errors.New("cannot write private Claude MCP configuration")
	}
	result := append(append([]string{}, args...), "--strict-mcp-config", "--mcp-config", path)
	if server.Context != nil {
		result = append(result, "--append-system-prompt", server.Context.Text)
	}
	return result, []string{mcpBearerEnvironment + "=" + server.BearerToken}, cleanup, nil
}
