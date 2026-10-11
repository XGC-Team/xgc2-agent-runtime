# Host-owned, session-scoped MCP binding

This is an optional product composition seam, not another agent runtime or a change to native permissions. Existing factories, Create payloads, journals and unconfigured native launches remain compatible.

`SessionScopedDriver.BindSession(ctx, create, localSessionID)` is called after the product Prepare callback validates the retained workspace and resource scope, before Driver.Open. It runs on creation and explicit resume. A product wrapper must reject rebinding, release grants in Close, and revoke a failed/opening grant. Broker's existing failure, session-close and host-shutdown paths close the wrapper.

`NewDriverWithLocalMCP` injects one transient loopback HTTP MCP server. Its name, URL and bounded bearer are reviewed host inputs, not model or browser parameters. The bearer is excluded from JSON encoding and is not added to the shared profile settings or broker journal.

Codex receives a process-level `-c mcp_servers.<name>=...` override, including `required=true` and a bearer environment-variable reference. The secret itself is only added to the child's filtered environment, never argv. Applying the binding before app-server startup covers both thread/start and thread/resume. ACP clients receive an Authorization header in private session/new or session/load parameters only when their initialize response advertises `agentCapabilities.mcpCapabilities.http`. Missing support fails instead of silently creating an ungrounded conversation. Per-turn options retain the context if permission changes reopen ACP.

Claude receives the server over its stream-json control channel. The launch only adds `--strict-mcp-config`, which excludes ambient MCP configuration; once Claude has initialized, the driver registers the server with an `mcp_set_servers` control request and sends the prompt after the answer. The bearer therefore never appears in the argument list, in the environment that every command run by the agent inherits, or in a file; each launch, including `--resume`, registers the current binding. A Claude that rejects the registration fails the turn before its prompt is sent. A server that cannot be reached leaves the turn without its tools, and the journal says so without repeating the native error text, which can echo the credential. The control request was exercised against Claude Code 2.1.296 (registration, `Authorization` header on the server's requests and the answer for an unreachable server).

Supported adapters: Codex, Claude, and compatible Cursor/Grok/OpenCode ACP versions. Installed-provider interoperability and native reconnect remain acceptance gates; protocol fixtures are not live-provider acceptance.

A product must own resource checks, token expiry/revocation, response bounds and effects admission. This library does not mint grants. This binding does NOT sandbox the native process, remove its other tools, restrict filesystem/network access, or erase credentials already observed by the provider. Native permission prompts remain meaningful and independent.

Verification: `go test -race -count=1 ./...` and `go vet ./...` from this module using the version in go.mod. `mcp_config_test.go` exercises credential placement, URL validation, required ACP capability negotiation and unchanged unconfigured launch. `session_scope_test.go` exercises durable scope on restart/resume, no prompt replay and cleanup on rejection.

Protocol references: https://developers.openai.com/codex/mcp/ and https://agentclientprotocol.com/protocol/v1/session-setup . The consuming product owns its MCP protocol-version support and compatibility matrix.
