# XGC2 Agent Runtime

Shared runtime that presents local third-party agent CLIs through one XGC conversation contract. The Go module normalizes provider protocols, launches a reviewed executable, and stores session events, questions, approvals, cancellation, explicit resume, and bounded HTTP/SSE replay. It does not implement a model loop, project store, robot controller, or workspace policy.

Context and workspace values are identifiers. They do not carry filesystem paths. Provider credentials stay with those CLIs. This repository does not read a home directory, shell profile, or secret store, and it does not embed a machine path.

The Go module is `github.com/XGC-Team/xgc2-agent-runtime`. The TypeScript package
[`web`](web/package.json), `@xgc2/agent-runtime`, owns the matching state/reducer,
HTTP client, React stream hook and `AgentConversation` presenter. A product renders these shared facts.
The presenter uses a pinned T3 Code chat slice.
Its source, license, theme contract and capability boundaries are documented in
[`web/UPSTREAM.md`](web/UPSTREAM.md).

## Product composition

```go
profiles, err := nativeagent.LoadConfig(reviewedProfilePath)
// Handle err. Empty path provides no profiles and launches nothing.
prepare := func(ctx context.Context, scope nativeagent.Create, sessionID string, fresh bool) (string, error) {
    // Verify scope.Context, resolve scope.Workspace through product authority,
    // and return its reviewed private working directory. On resume validate
    // the same retained workspace, never substitute a new mutable source.
    return productPrepare(ctx, scope, sessionID, fresh)
}
broker, err := nativeagent.NewBroker(privateJournalRoot, profiles, prepare, nil)
// Handle err and close broker at product shutdown.
// Optional shared local provider settings (one file for every local product).
err = nativeagent.ConfigureBroker(broker, nativeagent.BrokerOptions{SettingsFile: reviewedProfilePath})
// Handle err. Omit ConfigureBroker for legacy immutable profiles.
err = nativeagent.RegisterRoutes(mux, broker, "/api/agent-runtime")
```

`Create` has `ProfileID`, `Context ContextRef{Kind, ID}`,
`Workspace WorkspaceRef{ID, Revision}`, and `AgentAccessConfirmed`.
Context and workspace are comparable value references; they never contain
private paths. The product's `Prepare` callback verifies resource ownership,
reviewed workspace revision and fresh/resume policy. `Factory` is injectable
for deterministic protocol tests. The default factory starts the selected
reviewed native CLI; native login remains owned by that CLI.

A product may use a `research-project` context with an exact clean Git commit, or an
`experiment` context with a server-configured workspace. Those policies do not
enter this package. Idempotency is scoped to context kind and
ID; a repeated key with changed workspace/profile/consent is a conflict.

```ts
import { createAgentClient } from '@xgc2/agent-runtime/client'
import { AgentConversation, useAgentStream } from '@xgc2/agent-runtime/react'
import '@xgc2/agent-runtime/styles.css'

const client = createAgentClient({ basePath: '/api/agent-runtime' })
// Inside a product component:
const { state, connection, error } = useAgentStream(session, reload, client)
// Render shared native facts and forward only explicit operator decisions:
<AgentConversation state={state} locale="en"
  onSend={(text) => sendWithProductIdempotency(session.id, text)}
  onInterrupt={() => client.cancelNativeTurn(session.id)}
  onAnswer={(requestId, answer) => client.answerNativeRequest(session.id, requestId, answer)} />
```

The client exposes provider settings/refresh and profiles/sessions/create/prompt/answer/cancel/reconnect/close.
Inject `fetch` for product transport tests. Both HTTP and SSE accept only a
same-origin absolute API root. The HTTP server independently enforces loopback
peer, loopback Host, same-origin browser requests and the custom mutation
header `X-XGC-Agent-Client: 1`. A product-controlled proxy on the same machine may forward the original matching
Host/Origin and mutation header over a real loopback connection, after checking
its own resource/session authority. Forwarded headers never establish trust;
remote authenticated ingress is not provided.

The single protocol schema is `xgc.agent-runtime/v1`. Deltas append, snapshots
replace, patches preserve text, and item identity includes its turn. Submitted
answers remain pending until a native resolution event. Transport reconnect
replays by cursor and never resubmits a prompt. EOF does not imply success.
Approvals are answered only through explicit product operator callbacks.

This module preserves the reviewed native drivers for Codex app-server, ACP
clients and bounded Claude stream-json. Codex uses native `thread/start`,
`thread/resume`, `item/agentMessage/delta`, approval requests and terminal turn
events. It does not inspect vendor authentication or transcript files.

## Shared provider settings and current-turn options

`DefaultSettingsPath()` resolves the shared local `xgc/agent-runtime.json` under
`os.UserConfigDir()`. A missing file is allowed. `ConfigureBroker` initially
lists the five native clients disabled; installing a CLI never enables it.
Settings contain enabled state, a reviewed binary path and default model,
effort and permission selections. Credentials remain with the native clients;
login status is reported as unknown until a public native status interface
confirms it. The settings API accepts no tokens, environment overrides or
arbitrary command arguments.

`GET /settings` rereads the shared file and cached metadata without spawning a
CLI. `POST /settings/refresh {id}` probes only that provider. `POST /settings`
accepts `{revision, provider:{id,provider,enabled,binaryPath,defaults}}`; an OS
file lock, disk re-read and atomic private-file replacement enforce revision
CAS across independent product processes. A stale revision returns 409.
Other brokers observe committed settings on their next read or creation.
Probe results are process-local caches, not a second settings store.

`Create.options` and prompt `{text,options}` accept `{model,effort,permission}`.
Selections must match the reported native capability descriptors. Each new
session records a private resolved profile/default snapshot; later shared
settings changes do not silently change existing sessions. Per-turn overrides
are included in idempotency checks, so replaying a key with changed selections
conflicts. Existing calls without options retain their prior signature and
behavior. Native defaults are used only when actually reported; no higher
model, reasoning level or permission fallback is invented.

Prompt receipts with selections use the typed details contract
`{type:"userMessage",providerOptions:{model?,effort?,permission?}}` on the submitted
`user` snapshot. The same option values restore the durable idempotency check.
The Web decoder normalizes only the previously persisted, untagged
`{providerOptions:{...}}` shape on that exact receipt; other unknown detail types,
roles or fields remain errors. Existing journals are replayed without rewriting
or resubmitting their prompts. The shared sanitized nine-event replay fixture
tests the Go receipt and Web decoder together; native/tool details and prompt
receipt metadata must evolve under the same protocol contract.

Codex reads `model/list`, `config/read` and `account/read`; model, effort and
explicit sandbox/approval choices reach `thread/start` and `turn/start`.
Grok reads native ACP model metadata and carries reasoning through its native
model extension, with the fixed T3 permission launch flags. Cursor reads native
ACP session/config modes and its model-picker extension; per-model reasoning
and native modes remain distinct. OpenCode uses native ACP model/mode/effort
config options and only advertised CLI variants. Claude uses the fixed T3
compatibility catalog filtered by installed CLI version, with native CLI
model/effort/permission flags and the stdin/stdout permission/question control
channel. Its catalog is not a claim that every listed model is entitled by the
current account. The legacy unconfigured Claude path remains Read/Glob/Grep.
Mappings and their precise upstream source hashes are recorded in
[`upstream/t3/PROVENANCE.json`](upstream/t3/PROVENANCE.json); the original license
is retained alongside the extracted manifest.

Metadata inspection has one 60-second budget, including cold selection lookup;
there is no retry loop. Native connect has 65 seconds. Consumers must allow
metadata/settings/create/prompt responses through those bounds while retaining
shorter ordinary read/cancel timeouts. Invalid or already replayed commands do
not trigger fresh discovery. A native timeout is not evidence of missing login.

Passive queue/catalog/usage/hook notifications and unknown passive updates do
not become chat messages. Normalized messages, tools, plans, explicit errors
and operator requests remain visible. Unknown requests that need an answer are
rejected explicitly; no unknown request is implicitly approved. This module
adds no project/session-management UI, Git operations, workspace manager or
second agent loop.

## Verification and adoption

```bash
go test -race ./...
go vet ./...
npm --prefix web ci --ignore-scripts
npm --prefix web test
```

Go tests use fake subprocess protocol fixtures; they make no vendor requests.
The frontend uses Node's native test runner for reducer and transport tests,
and Vitest for the shared chat and decision controls.
Consumer typechecking checks the React hook against its installed React peer.
Research integration remains in `platforms/research-os` and uses a local Go
replace plus a `file:` dependency. Run `npm --prefix web ci --ignore-scripts` and
`npm --prefix web run build` before installing a consumer. Generated `dist`
is a package artifact and is not a second source authority. Install local file
dependencies with
`install-links=true` to create a consumer package snapshot with normal peer
resolution. Refresh that snapshot with a clean install when editing the shared source; a
same-version install can retain the previous copied package.

The shared owner must be included in consumer build inputs and CI checkouts.
No published module, hosted API, live subscription verification or operating
system sandbox is implied by these extraction tests.

### Optional durable prompt queue

`POST /sessions/{id}/queue` accepts `enqueue`, `edit`, `remove`, `reorder`,
`pause`, and `resume`. Enqueue uses its stable Idempotency-Key; other mutations
require the current `expectedRevision`. The journal's `prompt.queue` event
projects the ordered pending messages and pause state. A queued message captures
its model, effort, and permission selection at admission. Normal completion
advances one message; failed, interrupted, or unknown completion pauses the
remaining queue. Host restart retains pending input but requires explicit resume.
A durably submitted user item is never replayed as another turn after a crash.

Queueing is separate from provider steering and from answering native approval
or question requests. Hosts opt into `AgentConversation.queueEnabled` and use
`AgentPromptQueue` for presentation callbacks. Ordinary direct prompt clients
retain their existing ready-only contract.
