# XGC2 Agent Runtime

Shared runtime that presents local third-party agent CLIs through one XGC conversation contract. The Go module normalizes provider protocols, launches a reviewed executable, and stores session events, questions, approvals, cancellation, explicit resume, and journal replay through XRPC HTTP/event streams. It does not implement a model loop, project store, robot controller, or workspace policy.

Context and workspace values are identifiers. They do not carry filesystem paths. Provider credentials stay with those CLIs. This repository does not read a home directory, shell profile, or secret store, and it does not embed a machine path.

The Go module is `github.com/XGC-Team/xgc2-agent-runtime`. The TypeScript package
[`web`](web/package.json), `@xgc2/agent-runtime`, owns the matching state/reducer,
HTTP client, React stream hook and `AgentConversation` presenter. A product renders these shared facts.
The presenter uses a pinned T3 Code chat slice.
Its source, license, theme contract and capability boundaries are documented in
[`web/UPSTREAM.md`](web/UPSTREAM.md).

## Product composition

```go
// The process owner creates an authenticated storage/client and owns its
// XRPC transport lifetime. Bindings name the explicit initialized database.
conversations, err := nativeagent.NewStore(conversationBinding)
settings, err := nativeagent.NewStore(sharedSettingsBinding)
// Handle errors. Bindings use agent-runtime / agent-runtime.v1 and granted scopes.
prepare := func(ctx context.Context, scope nativeagent.Create, sessionID string, fresh bool) (string, error) {
    return productPrepare(ctx, scope, sessionID, fresh)
}
broker, err := nativeagent.Open(nativeagent.OpenOptions{
    Storage: conversations, Settings: settings, NativeFiles: hostNativeFiles, Prepare: prepare,
})
// Handle err. Close broker before closing its storage transport and file roots.
var service nativeagent.Service = broker
session, err := service.Create(ctx, "create:operator-action", create)
// Handle err. Call the same service directly or mount it on the host edge.
const prefix = "/api/agent-runtime"
mux.Handle(prefix+"/", http.StripPrefix(prefix,
    nativeagent.Handler(service, nativeagent.HandlerOptions{})))
edge, err := httpx.ServeEdge(listener, authenticatedHostHandler(mux), httpx.HostOptions{
    MaxInFlight: 256, MaxConnections: 256,
})
// Handle err. The host owns listener, authentication and edge shutdown.
```

`nativeagent` is `github.com/XGC-Team/xgc2-agent-runtime`; `httpx` is
`github.com/XGC-Team/xgc2-xrpc/go/httpx`. `Open(OpenOptions)` returns a `*Broker`
implementing the function-first [`Service`](service.go). Every service call
takes the caller's context. The API includes provider settings, conversation
creation/listing/metadata, prompt and queue commands, Stop, reconnect/close,
pending inputs/answers, the host decision evaluator, attention and event
subscriptions. `Subscribe(ctx, id, afterSeq, emit)` replays the journal and then
emits live events until the context ends; ending it leaves the worker running.

[`Handler`](xrpc_service.go) mounts that service as XRPC `http.v1`, named
`xgc2.agent-runtime.v1.AgentRuntime` with API version `1`. Authentication,
authorization, CORS and CSRF belong to the host. The former loopback/Host checks
and `X-XGC-Agent-Client` mutation guard are removed. Size edge `MaxInFlight` and
`MaxConnections` for open streams plus ordinary calls; the values above are
examples. Leave edge `MaxCallTime` and `MaxResponseBytes` unset for event streams.
The mounted handler bounds ordinary calls at 70 seconds and request bodies at
256 KiB by default (`HandlerOptions.MaxCallTime` / `MaxBodyBytes`).

`hostNativeFiles` comes from `NewNativeFiles` with explicitly opened host grants;
see [native file capabilities](docs/native-files.md). The storage owner initializes
the namespace from `storage-manifest.json` (`DeploymentManifest`) and supplies
the exact database/schema/scope identity and instance-bound ServiceRef. The
consumer never initializes a database or imports old settings/journals.

Broker startup reads saved conversations without writing recovery events. It
projects sessions without a live worker as disconnected and pending queues as
paused. Unknown prior outcomes remain unknown; explicit resume and queue commands must
persist before dispatch. Retirement of eligible development history is bounded
and event-driven.

`Create` has `ProfileID`, `Context ContextRef{Kind, ID}`,
`Workspace WorkspaceRef{ID, Revision}`, and `AgentAccessConfirmed`.
Context and workspace are comparable value references; they never contain
private paths. The product's `Prepare` callback verifies resource ownership,
reviewed workspace revision and fresh/resume policy. `Factory` is injectable
for deterministic protocol tests. The default factory starts the selected
reviewed native CLI; native login remains owned by that CLI.

The host supplies context/workspace policy. Create idempotency is scoped to
context kind and ID; a repeated key with changed workspace/profile/consent is a
conflict. The keys of `Create`, `Prompt` and queue enqueue travel as XRPC
`X-Request-ID`, replacing the private `Idempotency-Key` header. Reuse the same
request ID and content when retrying. A mutation returning `deadline_exceeded`
has an unknown outcome, so retry it with that same ID. Success bodies contain
the result directly; failures use XRPC error codes and
`{"error":{"code","message","details"?}}`. The optional
`X-Xrpc-Timeout-Ms` header requests a budget within the handler's cap.

```ts
import { createAgentClient } from '@xgc2/agent-runtime/client'
import { AgentConversation, useAgentStream } from '@xgc2/agent-runtime/react'
import '@xgc2/agent-runtime/styles.css'

const transport = { basePath: '/api/agent-runtime', fetch: hostFetch }
const client = createAgentClient(transport)
// Inside a product component:
const { state, connection, error } = useAgentStream(session, reload, transport)
// Render shared native facts and forward only explicit operator decisions:
<AgentConversation state={state} locale="en"
  onSend={(text) => sendWithProductIdempotency(session.id, text)}
  onInterrupt={() => client.cancelNativeTurn(session.id)}
  onAnswer={(requestId, answer) => client.answerNativeRequest(session.id, requestId, answer)} />
```

The client exposes provider settings/refresh and profiles/sessions/create/prompt/answer/cancel/reconnect/close.
It uses `@xgc2/xrpc-client.call()` and throws `XrpcError` with code, status,
request ID and disposition. `isStaleRequest(error)` identifies an input conflict
whose `details.reason` is `stale`. `basePath` accepts an origin-relative path or
an HTTP(S) URL. Optional `headers` and `fetch` carry host credentials and support
transport tests; `deadlines` overrides the 15-second ordinary and 70-second
inspection/connect budgets. `useAgentStream` uses `@xgc2/xrpc-client.events()`
with the same host transport options. It resumes from the last applied sequence
via `Last-Event-ID`. The first connection and a `reset` obtain the bounded
`/current` seed and start at its exact cursor. Older records are fetched only
through an explicit history page request. Event streams have no unary call
deadline or periodic recycle. Invalid events stop the stream with a diagnosis.

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

`ConfigureBroker` binds one explicit shared storage scope. A missing providers
record lists the five native clients disabled; installing a CLI never enables it.
Settings contain enabled state, a reviewed binary path and default model,
effort and permission selections. Credentials remain with the native clients;
login status is reported as unknown until a public native status interface
confirms it. The settings API accepts no tokens, environment overrides or
arbitrary command arguments.

`GET /settings` reads the shared storage record and cached metadata without spawning a
CLI. `POST /settings/refresh {id}` probes only that provider. `POST /settings`
accepts `{revision, provider:{id,provider,enabled,binaryPath,defaults}}`; a guarded
snapshot and atomic storage batch enforce revision CAS across independent
product processes. A stale revision returns 409.
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
The Web decoder requires the typed receipt and rejects unknown detail types,
roles and fields. Saved new-model events replay without resubmitting prompts. The shared sanitized nine-event replay fixture
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
go vet ./... && go test -p 2 ./...
cd web
npm ci && npm run build && npm run lint && npx vitest run
node --test test/*.test.mjs
```

Go tests use fake subprocess protocol fixtures; they make no vendor requests.
The frontend uses Node's native test runner for reducer and transport tests,
and Vitest for the shared chat and decision controls.
Consumer typechecking checks the React hook against its installed React peer.
The current development inputs use a local Go replace for XRPC and a `file:`
dependency on `@xgc2/xrpc-client`; integration must pin the released inputs.
Run `npm --prefix web ci` and
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
`pause`, and `resume`. Enqueue uses its stable XRPC request ID; other mutations
require the current `expectedRevision`. The committed `prompt.queue` event
projects the ordered pending messages and pause state. A queued message captures
its model, effort, and permission selection at admission. Normal completion
advances one message; failed, interrupted, or unknown completion pauses the
remaining queue. Host restart retains pending input but requires explicit resume.
A durably submitted user item is never replayed as another turn after a crash.

Stop holds pending messages before asking the native client to cancel. That
hold survives normal completion, a refused Stop, reconnect and restart; only
an explicit queue `resume` releases it. Messages admitted while Stop is in
flight are held too. Stopping an empty queue does not hold future messages.
Rejected commands and validation errors do not create a hold. The turn enters
`cancelling` only after the native client accepts Stop; a refusal keeps the turn
and its approvals active. A lost native client ends the turn as `unknown` and
holds the queue. `CloseSession` followed by `Reconnect` recovers a stalled Stop
without resending the prior prompt.

Queueing is separate from provider steering and from answering native approval
or question requests. Hosts opt into `AgentConversation.queueEnabled` and use
`AgentPromptQueue` for presentation callbacks. Ordinary direct prompt clients
retain their existing ready-only contract.

## Persistence

The deployment input is `storage-manifest.json`. The host supplies explicit
storage scopes and database/schema identities. Conversation metadata, ordered
events and submitted prompt identities commit in one CAS batch. Provider settings
use a separate shared scope and revision CAS. Storage owns database files and
FULL durability; the broker retains no transcript files.

`StorageBinding.JournalDelay` defaults to `DefaultJournalDelay` (200 ms).
Streamed deltas and other non-durable updates are published immediately and
batched within that delay; a host crash can lose the uncommitted tail. Submitted
user snapshots, queue commands, identity/metadata/runtime changes, input
submission/resolution and turn endings commit immediately; acting calls wait
for the required durable event. No broker or session lock is held across
storage calls. A failed commit stops that conversation's worker and reports
`unavailable`. `Broker.Close` flushes pending events with a 15-second bound.

Startup reads saved metadata and only the queue snapshots that still have
pending prompts. Empty queues restore their revision from session metadata
without reading history. Interrupted sessions are disconnected and retained
queued prompts are paused; resume is explicit. A lost
write reply is checked through its request receipt and never automatically
resubmitted. Missing storage or identity mismatches fail startup.

Limits are 128 active workers, 2048 retained conversations, 100000 events and
32 MiB of event JSON per conversation, and 384 MiB of retained event JSON per
Broker, including pending writes. Storage separately enforces its collection,
byte and receipt limits. Admission reports `resource_exhausted` at a limit.
An idle development conversation becomes eligible for age-based retirement
after 30 days without a journal event; count/byte pressure can make it eligible
earlier. This is not a 30-day deletion deadline: there is no expiry timer, and
an otherwise idle store can retain eligible history indefinitely. Startup,
subsequent lifecycle events, quota crossings, reader release or explicit cleanup
can advance retirement. Each automatic or explicit call performs at most 16
bounded steps (at most 256 events); unfinished work requires a later trigger or
explicit continuation. A failure also requires a later trigger or explicit retry.
Active turns, pending input, queued prompts and selected readers are protected.
`CleanupHistory` and `POST /sessions/{id}/cleanup-history` perform bounded work
and return whether retirement is complete. Only one cleanup is admitted per
Broker; a concurrent explicit request is refused immediately with
`resource_exhausted`, rather than queued. Cancelled requests release admission,
and continuation joins any in-flight driver close. Broker shutdown cancels and
joins admitted cleanup. Repeat an explicit cleanup to finish large histories.
Session metadata records interrupted cleanup progress so a reopen can resume
without exposing a partial journal.

`GET /sessions/{id}/current` supplies the latest in-memory turn, pending inputs,
queue, session facts and an exact cursor without reading a journal. A reopened
Broker exposes earlier turns through explicit history instead of fabricating a
current turn. A current projection that exceeds its 256-item / 2 MiB bound
returns `resource_exhausted`; the journal stays available.

`GET /sessions/{id}/history` reads a bounded newest suffix. Supply `before` from
`nextBefore` for an earlier page. Events in each page remain in sequence order;
`lastSeq` is the observed stream boundary. A page contains at most 32 events and
may be shorter at a storage or pending-journal boundary. Subscribers are capped
at eight per conversation and 32 per Broker and replay bounded chunks. A stream
with no cursor still replays from sequence zero. Storage backup/restore covers
current-format records. There is no old JSON/JSONL/schema loader or importer.

The provider's native state and scratch are separate from Broker journal quota.
Their grant inventory runs before launch and after exit; it does not limit a live
vendor process. Writable mount isolation and physical filesystem quotas belong
to the deployment.
