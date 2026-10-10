package agentruntime

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"mime"
	"net/http"
	"strconv"
	"time"

	xrpc "github.com/XGC-Team/xgc2-xrpc/go"
	"github.com/XGC-Team/xgc2-xrpc/go/httpx"
)

// Defaults of HandlerOptions. The call budget covers the 65 s native connect
// and the 60 s CLI inspection that Create, Prompt and the settings calls wait for.
const (
	DefaultCallTimeout  = 70 * time.Second
	DefaultMaxBodyBytes = 256 << 10
)

// HandlerOptions are plain limits; a zero field selects the default.
type HandlerOptions struct {
	// MaxCallTime caps the caller's X-Xrpc-Timeout-Ms and is the budget of a
	// call that sends none (default DefaultCallTimeout).
	MaxCallTime time.Duration
	// MaxBodyBytes bounds one request body (default DefaultMaxBodyBytes).
	MaxBodyBytes int64
	// Events carries the heartbeat, frame write timeout and drain signal of the
	// session event streams.
	Events httpx.EventsOptions
}

// Handler serves service as the XRPC http.v1 service ServiceName. The host
// mounts it under a prefix of its public edge (httpx.ServeEdge) behind its own
// authentication, CORS and CSRF policy; this package checks none of them.
//
// Unary routes speak JSON. A success body is the result itself; a failure is
// {"error":{"code","message","details"?}} with a code of the XRPC vocabulary.
// The request id (X-Request-ID, generated when absent) is the idempotency key of
// Create, Prompt and Queue enqueue, so a retried call repeats no effect. The
// optional X-Xrpc-Timeout-Ms header sets the call budget. deadline_exceeded on a
// mutation means its outcome is unknown: repeat it with the same request id.
//
//	GET  /providers
//	GET  /settings                     POST /settings          POST /settings/refresh
//	GET  /attention
//	GET  /sessions                     POST /sessions
//	GET  /sessions/{id}                POST /sessions/{id}/metadata
//	POST /sessions/{id}/prompts        POST /sessions/{id}/queue
//	POST /sessions/{id}/cancel         POST /sessions/{id}/reconnect   POST /sessions/{id}/close
//	GET  /sessions/{id}/inputs         POST /sessions/{id}/inputs/{requestId}
//	POST /sessions/{id}/evaluate-inputs
//	GET  /sessions/{id}/events         (event stream)
//
// The event stream (httpx.ServeEvents) has no call deadline. Its cursor is the
// event sequence number: the client resumes with Last-Event-ID (or ?after=), and
// a cursor beyond the journal is answered with a reset that replays from the
// first event. Ending a stream never stops the worker.
func Handler(service Service, options HandlerOptions) http.Handler {
	if options.MaxCallTime <= 0 {
		options.MaxCallTime = DefaultCallTimeout
	}
	if options.MaxBodyBytes <= 0 {
		options.MaxBodyBytes = DefaultMaxBodyBytes
	}
	h := &handler{service: service, options: options, mux: http.NewServeMux()}
	h.route("GET /providers", func(c *call) (int, any, error) {
		providers, err := service.Providers(c.ctx)
		return http.StatusOK, providers, err
	})
	h.route("GET /settings", func(c *call) (int, any, error) {
		settings, err := service.Settings(c.ctx)
		return http.StatusOK, settings, err
	})
	h.route("POST /settings", func(c *call) (int, any, error) {
		var update SettingsUpdate
		if err := c.decode(&update); err != nil {
			return 0, nil, err
		}
		settings, err := service.UpdateSettings(c.ctx, update)
		return http.StatusOK, settings, err
	})
	h.route("POST /settings/refresh", func(c *call) (int, any, error) {
		var refresh struct {
			ID string `json:"id"`
		}
		if err := c.decode(&refresh); err != nil {
			return 0, nil, err
		}
		settings, err := service.RefreshSettings(c.ctx, refresh.ID)
		return http.StatusOK, settings, err
	})
	h.route("GET /attention", func(c *call) (int, any, error) {
		attention, err := service.Attention(c.ctx)
		return http.StatusOK, attention, err
	})
	h.route("GET /sessions", func(c *call) (int, any, error) {
		query := c.r.URL.Query()
		list := SessionListOptions{After: query.Get("after"), Context: ContextRef{Kind: query.Get("contextKind"), ID: query.Get("contextId")}}
		if limit := query.Get("limit"); limit != "" {
			n, err := strconv.Atoi(limit)
			if err != nil {
				return 0, nil, invalid("invalid conversation page limit")
			}
			list.Limit = n
		}
		page, err := service.List(c.ctx, list)
		return http.StatusOK, page, err
	})
	h.route("POST /sessions", func(c *call) (int, any, error) {
		var create Create
		if err := c.decode(&create); err != nil {
			return 0, nil, err
		}
		session, err := service.Create(c.ctx, c.id, create)
		return http.StatusAccepted, session, err
	})
	h.route("GET /sessions/{id}", func(c *call) (int, any, error) {
		session, err := service.Get(c.ctx, c.r.PathValue("id"))
		return http.StatusOK, session, err
	})
	h.route("POST /sessions/{id}/metadata", func(c *call) (int, any, error) {
		var update MetadataUpdate
		if err := c.decode(&update); err != nil {
			return 0, nil, err
		}
		session, err := service.UpdateMetadata(c.ctx, c.r.PathValue("id"), update)
		return http.StatusOK, session, err
	})
	h.route("GET /sessions/{id}/inputs", func(c *call) (int, any, error) {
		inputs, err := service.Inputs(c.ctx, c.r.PathValue("id"))
		return http.StatusOK, inputs, err
	})
	h.route("POST /sessions/{id}/evaluate-inputs", func(c *call) (int, any, error) {
		return http.StatusAccepted, requested, service.EvaluateInputs(c.ctx, c.r.PathValue("id"))
	})
	h.route("POST /sessions/{id}/queue", func(c *call) (int, any, error) {
		var command QueueCommand
		if err := c.decode(&command); err != nil {
			return 0, nil, err
		}
		queue, err := service.Queue(c.ctx, c.r.PathValue("id"), c.id, command)
		return http.StatusAccepted, queue, err
	})
	h.route("POST /sessions/{id}/prompts", func(c *call) (int, any, error) {
		var prompt PromptRequest
		if err := c.decode(&prompt); err != nil {
			return 0, nil, err
		}
		turn, err := service.Prompt(c.ctx, c.r.PathValue("id"), c.id, prompt)
		return http.StatusAccepted, struct {
			TurnID string `json:"turnId"`
		}{turn}, err
	})
	h.route("POST /sessions/{id}/inputs/{requestId}", func(c *call) (int, any, error) {
		var answer Answer
		if err := c.decode(&answer); err != nil {
			return 0, nil, err
		}
		return http.StatusAccepted, struct {
			Submitted bool `json:"submitted"`
		}{true}, service.Answer(c.ctx, c.r.PathValue("id"), c.r.PathValue("requestId"), answer)
	})
	h.route("POST /sessions/{id}/cancel", func(c *call) (int, any, error) {
		return http.StatusAccepted, requested, service.Cancel(c.ctx, c.r.PathValue("id"))
	})
	h.route("POST /sessions/{id}/reconnect", func(c *call) (int, any, error) {
		return http.StatusAccepted, requested, service.Reconnect(c.ctx, c.r.PathValue("id"))
	})
	h.route("POST /sessions/{id}/close", func(c *call) (int, any, error) {
		return http.StatusOK, struct {
			Closed bool `json:"closed"`
		}{true}, service.CloseSession(c.ctx, c.r.PathValue("id"))
	})
	h.mux.HandleFunc("GET /sessions/{id}/events", h.events)
	return h
}

var requested = struct {
	Requested bool `json:"requested"`
}{true}

type handler struct {
	service Service
	options HandlerOptions
	mux     *http.ServeMux
}

// call is one admitted unary request.
type call struct {
	r *http.Request
	w http.ResponseWriter
	// ctx carries the call budget.
	ctx context.Context
	// id is the request id, which doubles as the idempotency key.
	id      string
	maxBody int64
}

// decode reads the JSON request body into v: one value, no unknown fields.
func (c *call) decode(v any) error {
	if media, _, err := mime.ParseMediaType(c.r.Header.Get("Content-Type")); err != nil || media != "application/json" {
		return invalid("the request body must be application/json")
	}
	decoder := json.NewDecoder(http.MaxBytesReader(c.w, c.r.Body, c.maxBody))
	decoder.DisallowUnknownFields()
	if decoder.Decode(v) != nil || decoder.Decode(new(any)) != io.EOF {
		return invalid("invalid or oversized JSON body")
	}
	return nil
}

func (h *handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if _, pattern := h.mux.Handler(r); pattern != "" {
		h.mux.ServeHTTP(w, r)
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	for _, method := range []string{http.MethodGet, http.MethodPost} {
		other := r.Clone(r.Context())
		other.Method = method
		if _, pattern := h.mux.Handler(other); pattern != "" {
			w.Header().Set("Allow", method)
			writeFailure(w, http.StatusMethodNotAllowed, "", "invalid_argument", invalid("%s is not served with %s", r.URL.Path, r.Method), nil)
			return
		}
	}
	writeError(w, "", classified{ErrNotFound, "unknown route"})
}

// route registers one unary route.
func (h *handler) route(pattern string, serve func(*call) (int, any, error)) {
	h.mux.HandleFunc(pattern, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-store")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		id, err := requestID(r)
		if err == nil {
			w.Header().Set(httpx.RequestIDHeader, id)
		}
		budget := h.options.MaxCallTime
		if err == nil {
			budget, err = callBudget(r, budget)
		}
		if err != nil {
			writeError(w, id, err)
			return
		}
		ctx, cancel := context.WithTimeout(r.Context(), budget)
		defer cancel()
		status, result, err := serve(&call{r: r, w: w, ctx: ctx, id: id, maxBody: h.options.MaxBodyBytes})
		if err != nil {
			writeError(w, id, err)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		_ = json.NewEncoder(w).Encode(result)
	})
}

// requestID returns the request id of r, or a fresh random one when it has none.
func requestID(r *http.Request) (string, error) {
	switch ids := r.Header.Values(httpx.RequestIDHeader); len(ids) {
	case 0:
		var raw [16]byte
		if _, err := rand.Read(raw[:]); err != nil {
			return "", errors.New("request identity unavailable")
		}
		return hex.EncodeToString(raw[:]), nil
	case 1:
		if xrpc.ValidID(ids[0]) {
			return ids[0], nil
		}
	}
	return "", invalid("X-Request-ID must be one value matching [A-Za-z0-9._:-]{1,128}")
}

// callBudget is the caller's X-Xrpc-Timeout-Ms capped by max, or max.
func callBudget(r *http.Request, max time.Duration) (time.Duration, error) {
	values := r.Header.Values(httpx.TimeoutHeader)
	if len(values) == 0 {
		return max, nil
	}
	ms, err := xrpc.ParseTimeoutMS(values[0])
	if err != nil || len(values) > 1 {
		return 0, invalid("X-Xrpc-Timeout-Ms must be one canonical decimal of 1 to 86400000")
	}
	return min(time.Duration(ms)*time.Millisecond, max), nil
}

var statuses = map[string]int{
	"invalid_argument": http.StatusBadRequest, "unauthenticated": http.StatusUnauthorized, "permission_denied": http.StatusForbidden,
	"not_found": http.StatusNotFound, "conflict": http.StatusConflict, "resource_exhausted": http.StatusTooManyRequests,
	"cancelled": 499, "internal": http.StatusInternalServerError, "unavailable": http.StatusServiceUnavailable,
	"deadline_exceeded": http.StatusGatewayTimeout,
}

// writeError answers with the standard XRPC error envelope. A stale request is
// a conflict whose details say so, so a client can refresh instead of retrying.
func writeError(w http.ResponseWriter, id string, err error) {
	code := errorCode(err)
	var details map[string]string
	if errors.Is(err, ErrStale) {
		details = map[string]string{"reason": "stale"}
	}
	writeFailure(w, statuses[code], id, code, err, details)
}

func writeFailure(w http.ResponseWriter, status int, id, code string, err error, details map[string]string) {
	if id != "" {
		w.Header().Set(httpx.RequestIDHeader, id)
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(struct {
		Error any `json:"error"`
	}{struct {
		Code    string            `json:"code"`
		Message string            `json:"message"`
		Details map[string]string `json:"details,omitempty"`
	}{code, err.Error(), details}})
}

// events serves GET /sessions/{id}/events through httpx.ServeEvents.
func (h *handler) events(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	rid, err := requestID(r)
	if err == nil {
		w.Header().Set(httpx.RequestIDHeader, rid)
		// The response is committed as 200 on entry: refuse here what the stream cannot serve.
		_, err = h.service.Get(r.Context(), id)
	}
	if err != nil {
		writeError(w, rid, err)
		return
	}
	_ = httpx.ServeEvents(w, r, eventSource{h.service, id}, h.options.Events)
}

// eventSource adapts Service.Subscribe to httpx.EventSource: the cursor is the
// event sequence number, and a cursor that is not one, or lies beyond the
// journal, is unknown to the source, which streams answer with a reset.
type eventSource struct {
	service Service
	id      string
}

func (s eventSource) Subscribe(ctx context.Context, after string, emit func(httpx.Event) error) error {
	var cursor uint64
	if after != "" {
		var err error
		if cursor, err = strconv.ParseUint(after, 10, 64); err != nil {
			return httpx.ErrCursorGone
		}
	}
	err := s.service.Subscribe(ctx, s.id, cursor, func(e Event) error {
		data, err := json.Marshal(e)
		if err != nil {
			return err
		}
		return emit(httpx.Event{ID: strconv.FormatUint(e.Seq, 10), Type: "event", Data: data})
	})
	if errors.Is(err, ErrCursor) {
		return httpx.ErrCursorGone
	}
	return err
}
