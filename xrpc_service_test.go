package agentruntime

import (
	"context"
	"encoding/json"
	"errors"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/XGC-Team/xgc2-storage/api"
	xrpc "github.com/XGC-Team/xgc2-xrpc/go"
	"github.com/XGC-Team/xgc2-xrpc/go/httpx"
)

type exchange struct {
	status int
	header http.Header
	body   []byte
}

func (e exchange) error(t *testing.T) (code, message string, details map[string]string) {
	t.Helper()
	var envelope struct {
		Error struct {
			Code, Message string
			Details       map[string]string
		}
	}
	if err := json.Unmarshal(e.body, &envelope); err != nil || envelope.Error.Code == "" {
		t.Fatalf("status %d is not an XRPC error envelope: %s", e.status, e.body)
	}
	return envelope.Error.Code, envelope.Error.Message, envelope.Error.Details
}

func (e exchange) decode(t *testing.T, v any) {
	t.Helper()
	if e.status/100 != 2 || json.Unmarshal(e.body, v) != nil {
		t.Fatalf("status %d body %s", e.status, e.body)
	}
}

// serve sends one request through the XRPC service handler.
func serve(h http.Handler, method, path, body string, headers map[string]string) exchange {
	r := httptest.NewRequest(method, path, strings.NewReader(body))
	if body != "" {
		r.Header.Set("Content-Type", "application/json")
	}
	for name, value := range headers {
		r.Header.Set(name, value)
	}
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	return exchange{w.Code, w.Header(), w.Body.Bytes()}
}

func TestHandlerRequestIDIsTheIdempotencyKeyOfCreateAndPrompt(t *testing.T) {
	b, _, _ := conversationBroker(t, false)
	h := Handler(b, HandlerOptions{})
	create := func(id string, headers map[string]string) exchange {
		if id != "" {
			headers["X-Request-ID"] = id
		}
		return serve(h, "POST", "/sessions", `{"profileId":"fixture","context":{"kind":"fixture","id":"project"},"workspace":{"id":"workspace","revision":"reviewed-v1"},"accessConfirmed":true}`, headers)
	}
	var first, again, other Session
	response := create("create:one", map[string]string{})
	response.decode(t, &first)
	if response.status != http.StatusAccepted || response.header.Get("X-Request-ID") != "create:one" {
		t.Fatalf("create=%d id=%q", response.status, response.header.Get("X-Request-ID"))
	}
	create("create:one", map[string]string{}).decode(t, &again)
	create("create:two", map[string]string{}).decode(t, &other)
	if again.ID != first.ID || other.ID == first.ID {
		t.Fatalf("request id is not the idempotency key: %s %s %s", first.ID, again.ID, other.ID)
	}
	// The private header is gone: it neither names the operation nor conflicts with the request id.
	var viaHeader, viaHeaderAgain Session
	create("", map[string]string{"Idempotency-Key": "private"}).decode(t, &viaHeader)
	create("", map[string]string{"Idempotency-Key": "private"}).decode(t, &viaHeaderAgain)
	if viaHeader.ID == viaHeaderAgain.ID {
		t.Fatal("the private Idempotency-Key header still identifies an operation")
	}
	waitState(t, b, first.ID, "ready")

	prompt := func(id, text string) exchange {
		return serve(h, "POST", "/sessions/"+first.ID+"/prompts", `{"text":"`+text+`"}`, map[string]string{"X-Request-ID": id})
	}
	var turn, replay struct{ TurnID string }
	if r := prompt("prompt:1", "inspect"); r.status != http.StatusAccepted {
		t.Fatalf("prompt=%d %s", r.status, r.body)
	} else {
		r.decode(t, &turn)
	}
	prompt("prompt:1", "inspect").decode(t, &replay)
	if turn.TurnID == "" || replay.TurnID != turn.TurnID || turn.TurnID != "t_"+hash(first.ID + "\x00prompt:1")[:32] {
		t.Fatalf("turn identity %q replay %q", turn.TurnID, replay.TurnID)
	}
	code, _, _ := prompt("prompt:1", "changed text").error(t)
	if code != "conflict" {
		t.Fatalf("reused request id with different text: %s", code)
	}
}

func TestHandlerRequestIDAndBudgetValidation(t *testing.T) {
	b, _, _ := conversationBroker(t, false)
	h := Handler(b, HandlerOptions{})
	generated := serve(h, "GET", "/providers", "", nil)
	if id := generated.header.Get("X-Request-ID"); generated.status != 200 || !xrpc.ValidID(id) {
		t.Fatalf("no generated request id: %d %q", generated.status, id)
	}
	for name, headers := range map[string]map[string]string{
		"bad request id":      {"X-Request-ID": "has space"},
		"long request id":     {"X-Request-ID": strings.Repeat("a", 129)},
		"zero budget":         {"X-Xrpc-Timeout-Ms": "0"},
		"fractional budget":   {"X-Xrpc-Timeout-Ms": "1.5"},
		"leading zero budget": {"X-Xrpc-Timeout-Ms": "05"},
		"huge budget":         {"X-Xrpc-Timeout-Ms": "86400001"},
	} {
		t.Run(name, func(t *testing.T) {
			response := serve(h, "GET", "/providers", "", headers)
			if code, _, _ := response.error(t); response.status != 400 || code != "invalid_argument" {
				t.Fatalf("status %d code %s", response.status, code)
			}
		})
	}
}

// slowService blocks reads until the call budget ends.
type slowService struct{ Service }

func (slowService) Providers(ctx context.Context) ([]Provider, error) {
	<-ctx.Done()
	return nil, ctx.Err()
}

func TestHandlerAppliesTheCallersBudgetCappedByTheServer(t *testing.T) {
	h := Handler(slowService{}, HandlerOptions{MaxCallTime: 5 * time.Second})
	started := time.Now()
	response := serve(h, "GET", "/providers", "", map[string]string{"X-Xrpc-Timeout-Ms": "30"})
	if code, _, _ := response.error(t); response.status != http.StatusGatewayTimeout || code != "deadline_exceeded" || time.Since(started) > 2*time.Second {
		t.Fatalf("status %d code %s after %s", response.status, code, time.Since(started))
	}
	capped := Handler(slowService{}, HandlerOptions{MaxCallTime: 40 * time.Millisecond})
	started = time.Now()
	response = serve(capped, "GET", "/providers", "", map[string]string{"X-Xrpc-Timeout-Ms": "60000"})
	if code, _, _ := response.error(t); code != "deadline_exceeded" || time.Since(started) > 2*time.Second {
		t.Fatalf("caller budget was not capped: %s after %s", code, time.Since(started))
	}
}

func TestErrorsUseTheXRPCVocabulary(t *testing.T) {
	for _, tc := range []struct {
		err    error
		code   string
		status int
	}{
		{invalid("bad"), "invalid_argument", 400},
		{ErrNotFound, "not_found", 404},
		{errors.Join(ErrConflict, errors.New("storage")), "conflict", 409},
		{ErrStale, "conflict", 409},
		{ErrCursor, "conflict", 409},
		{exhausted("full"), "resource_exhausted", 429},
		{ErrUnavailable, "unavailable", 503},
		{context.DeadlineExceeded, "deadline_exceeded", 504},
		{context.Canceled, "cancelled", 499},
		{xrpc.Failure("permission_denied", xrpc.ResponseReceived, errors.New("denied")), "permission_denied", 403},
		{&api.Error{Code: "resource_exhausted", Message: "quota"}, "resource_exhausted", 429},
		{&api.Error{Code: "io_error", Message: "disk"}, "internal", 500},
		{errors.New("unclassified"), "internal", 500},
	} {
		w := httptest.NewRecorder()
		writeError(w, "id-1", tc.err)
		code, _, _ := exchange{w.Code, w.Header(), w.Body.Bytes()}.error(t)
		if code != tc.code || w.Code != tc.status || w.Header().Get("X-Request-ID") != "id-1" {
			t.Fatalf("%v: code %s status %d", tc.err, code, w.Code)
		}
	}
}

func TestStaleAnswerIsAConflictWithAReason(t *testing.T) {
	b, s := testBroker(t, "cursor")
	response := serve(Handler(b, HandlerOptions{}), "POST", "/sessions/"+s.ID+"/inputs/q_gone", `{"optionId":"allow"}`, nil)
	if code, _, details := response.error(t); response.status != 409 || code != "conflict" || details["reason"] != "stale" {
		t.Fatalf("status %d code %s details %v", response.status, code, details)
	}
}

// edge serves h on a Unix socket through the XRPC edge host, as Core does.
func edge(t *testing.T, h http.Handler) (xrpc.ServiceRef, *httpx.Host) {
	t.Helper()
	dir, err := os.MkdirTemp("", "ar-")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.RemoveAll(dir) })
	socket := filepath.Join(dir, "edge.sock")
	listener, err := net.Listen("unix", socket)
	if err != nil {
		t.Fatal(err)
	}
	host, err := httpx.ServeEdge(listener, h, httpx.HostOptions{})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		_ = host.Shutdown(ctx)
	})
	return xrpc.ServiceRef{TargetID: "test", Service: ServiceName, APIVersion: APIVersion, Profile: xrpc.HTTP, Endpoint: xrpc.Endpoint{Kind: "unix", Address: socket}}, host
}

func nextEvent(t *testing.T, events <-chan httpx.Event) httpx.Event {
	t.Helper()
	select {
	case e, ok := <-events:
		if !ok || e.Err != nil {
			t.Fatalf("stream ended: %+v", e)
		}
		return e
	case <-time.After(5 * time.Second):
		t.Fatal("no event")
		return httpx.Event{}
	}
}

func TestEventStreamResumesFromTheSequenceCursorAndResetsWhenItIsGone(t *testing.T) {
	b, s := testBroker(t, "cursor")
	_, _ = b.Prompt(bg, s.ID, "p1", PromptRequest{Text: "hello"})
	waitRequest(t, b, s.ID)
	journal, _, err := b.replay(s.ID, 0)
	if err != nil || len(journal) < 4 {
		t.Fatalf("journal=%d %v", len(journal), err)
	}
	ref, _ := edge(t, Handler(b, HandlerOptions{Events: httpx.EventsOptions{Heartbeat: 50 * time.Millisecond}}))
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	path := "/sessions/" + s.ID + "/events"

	full, err := httpx.SubscribeEvents(ctx, nil, ref, path, "")
	if err != nil {
		t.Fatal(err)
	}
	for i, want := range journal {
		got := nextEvent(t, full)
		var decoded Event
		if got.Type != "event" || got.ID != itoa(want.Seq) || json.Unmarshal(got.Data, &decoded) != nil || decoded.Seq != want.Seq || decoded.Kind != want.Kind {
			t.Fatalf("event %d: %+v want seq %d kind %s", i, got, want.Seq, want.Kind)
		}
	}

	// A client that saw the first events resumes after them with its cursor.
	resumed, err := httpx.SubscribeEvents(ctx, nil, ref, path, itoa(journal[1].Seq))
	if err != nil {
		t.Fatal(err)
	}
	if got := nextEvent(t, resumed); got.ID != itoa(journal[2].Seq) {
		t.Fatalf("resumed at %s, want %d", got.ID, journal[2].Seq)
	}

	// The stream outlives several heartbeats, then still delivers live events.
	time.Sleep(300 * time.Millisecond)
	r := waitRequest(t, b, s.ID)
	if err = b.Answer(bg, s.ID, r.ID, Answer{OptionID: r.Options[0].ID}); err != nil {
		t.Fatal(err)
	}
	last := journal[len(journal)-1].Seq
	if got := nextEvent(t, full); got.ID != itoa(last+1) {
		t.Fatalf("live event %s, want %d", got.ID, last+1)
	}

	// A cursor beyond the journal is gone: the stream resets and starts over.
	ahead, err := httpx.SubscribeEvents(ctx, nil, ref, path, "999999")
	if err != nil {
		t.Fatal(err)
	}
	if got := nextEvent(t, ahead); got.Type != httpx.EventReset {
		t.Fatalf("no reset: %+v", got)
	}
	if got := nextEvent(t, ahead); got.ID != itoa(journal[0].Seq) {
		t.Fatalf("restarted at %s, want %d", got.ID, journal[0].Seq)
	}
	// A cursor that is not a sequence number is unknown to the source as well.
	garbled, err := httpx.SubscribeEvents(ctx, nil, ref, path, "not-a-number")
	if err != nil {
		t.Fatal(err)
	}
	if got := nextEvent(t, garbled); got.Type != httpx.EventReset {
		t.Fatalf("no reset: %+v", got)
	}
	// Leaving never stops the worker.
	cancel()
	waitState(t, b, s.ID, "ready")
}

func TestEventStreamRefusesWhatItCannotServeBeforeCommitting(t *testing.T) {
	b, s := testBroker(t, "cursor")
	ref, _ := edge(t, Handler(b, HandlerOptions{}))
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_, err := httpx.SubscribeEvents(ctx, nil, ref, "/sessions/s_missing/events", "")
	var failure *xrpc.CallError
	if !errors.As(err, &failure) || failure.Code != "not_found" {
		t.Fatalf("missing session: %v", err)
	}
	r := httptest.NewRequest("GET", "/sessions/"+s.ID+"/events", nil)
	r.Header.Set("Last-Event-ID", "1")
	r.Header.Add("Last-Event-ID", "2")
	w := httptest.NewRecorder()
	Handler(b, HandlerOptions{}).ServeHTTP(w, r)
	if w.Code != 400 {
		t.Fatalf("two cursors: status %d", w.Code)
	}
}

func itoa(n uint64) string {
	data, _ := json.Marshal(n)
	return string(data)
}
