package agentruntime

import (
	"context"
	"errors"
)

// NewDriverWithLocalMCP adds exactly one host-reviewed transient MCP endpoint.
// It does not change the user's native permissions or claim OS isolation.
func NewDriverWithLocalMCP(p Profile, sink Sink, ask Ask, server LocalMCPServer) (Driver, error) {
	if err := server.validate(); err != nil {
		return nil, err
	}
	switch p.Provider {
	case "codex", "claude", "cursor", "grok", "opencode":
	default:
		return nil, errors.New("provider does not support host-owned local MCP")
	}
	driver, err := NewDriver(p, sink, ask)
	if err != nil {
		return nil, err
	}
	return &localMCPDriver{Driver: driver, server: server}, nil
}

type localMCPDriver struct {
	Driver
	server LocalMCPServer
}

func (d *localMCPDriver) Open(ctx context.Context, cwd, native string) error {
	return d.Driver.Open(withLocalMCP(ctx, d.server), cwd, native)
}
func (d *localMCPDriver) Prompt(ctx context.Context, turn, prompt string) error {
	return d.Driver.Prompt(withLocalMCP(ctx, d.server), turn, prompt)
}
func (d *localMCPDriver) PromptWithOptions(ctx context.Context, turn, prompt string, options AgentOptions) error {
	// Propagate through per-turn permission changes, which may reopen ACP.
	if configurable, ok := d.Driver.(optionDriver); ok {
		return configurable.PromptWithOptions(withLocalMCP(ctx, d.server), turn, prompt, options)
	}
	return errors.New("driver does not support selected options")
}
