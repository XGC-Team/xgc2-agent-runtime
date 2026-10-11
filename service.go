package agentruntime

import "context"

// ServiceName and APIVersion identify the XRPC http.v1 service Handler serves.
const (
	ServiceName = "xgc2.agent-runtime.v1.AgentRuntime"
	APIVersion  = "1"
)

// PromptRequest is one user message and the selections for its turn.
type PromptRequest struct {
	Text    string       `json:"text"`
	Options AgentOptions `json:"options,omitempty"`
}

// Service is the function-first API of the agent runtime. A host in the same
// process calls it directly; Handler exposes the same functions as an XRPC
// http.v1 service for other processes and browsers. *Broker implements it.
//
// Every call takes the caller's context. Calls that need a key (Create, Prompt,
// Queue enqueue) treat it as the idempotency identity of the operation: the
// XRPC service passes the request id. Replaying a key with the same content
// returns the original result without repeating the effect; replaying it with
// different content is ErrConflict. Reconnecting a session, replaying a stream
// or retrying a call never sends a prompt twice.
//
// Authentication and authorization belong to the host. The service trusts its
// caller and never infers an actor from a request.
type Service interface {
	// Providers lists the configured native clients and whether each can start.
	Providers(ctx context.Context) ([]Provider, error)
	// Settings reads provider settings; UpdateSettings replaces one provider
	// under the revision of the document read; RefreshSettings probes one
	// provider's installed CLI, its login and its models.
	Settings(ctx context.Context) (Settings, error)
	UpdateSettings(ctx context.Context, update SettingsUpdate) (Settings, error)
	RefreshSettings(ctx context.Context, id string) (Settings, error)

	// Create starts a conversation and returns it before its worker is ready.
	Create(ctx context.Context, key string, create Create) (Session, error)
	Get(ctx context.Context, id string) (Session, error)
	List(ctx context.Context, options SessionListOptions) (SessionPage, error)
	// UpdateMetadata renames or archives a conversation under its metadata revision.
	UpdateMetadata(ctx context.Context, id string, update MetadataUpdate) (Session, error)
	// Reconnect starts a new worker for a disconnected conversation. No prior
	// prompt is replayed and a held queue stays held.
	Reconnect(ctx context.Context, id string) error
	CloseSession(ctx context.Context, id string) error

	// Prompt starts a turn and returns its identity once the user message is
	// durable. Queue edits the durable queue of prompts that start after the
	// current turn; Cancel stops the current turn and holds the queue.
	Prompt(ctx context.Context, id, key string, prompt PromptRequest) (turnID string, err error)
	Queue(ctx context.Context, id, key string, command QueueCommand) (PromptQueue, error)
	Cancel(ctx context.Context, id string) error

	// Inputs lists the approvals and questions awaiting an answer; Answer
	// resolves one; EvaluateInputs asks the host's decision evaluator to look
	// at them again.
	Inputs(ctx context.Context, id string) ([]PendingRequest, error)
	Answer(ctx context.Context, id, requestID string, answer Answer) error
	EvaluateInputs(ctx context.Context, id string) error
	// Attention summarizes the live workers and their pending requests.
	Attention(ctx context.Context) (AttentionSnapshot, error)

	// Subscribe calls emit, in order and from one goroutine, with every event
	// of the conversation after the sequence number after, then with live
	// events, until ctx ends (it returns ctx.Err()), emit fails (it returns
	// that error) or the runtime closes (ErrUnavailable). A cursor beyond the
	// journal is ErrCursor; a missing conversation is ErrNotFound. Ending a
	// subscription never stops the worker.
	Subscribe(ctx context.Context, id string, after uint64, emit func(Event) error) error
}

var _ Service = (*Broker)(nil)
