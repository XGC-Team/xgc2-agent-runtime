package agentruntime

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
)

// No executable is launched: the factory below is an in-process protocol seam.
type conversationDriver struct {
	sink   Sink
	ask    Ask
	fail   bool
	closed atomic.Bool
}

func (d *conversationDriver) Open(_ context.Context, _ string, nativeID string) error {
	if d.fail {
		return errors.New("fixture initial connection failure")
	}
	if nativeID == "" {
		nativeID = "fixture-thread"
	}
	return d.sink(Event{Kind: "session.identity", AgentSessionID: nativeID})
}
func (d *conversationDriver) Prompt(ctx context.Context, turn, prompt string) error {
	_, err := d.ask(ctx, Request{Kind: "permission", Title: "Run fixture", Text: "sensitive request detail", Options: []Option{{ID: "yes", Kind: "allow_once"}, {ID: "no", Kind: "reject_once"}}})
	if err != nil {
		return err
	}
	return d.sink(Event{Kind: "turn.end", TurnID: turn, Status: "completed"})
}
func (d *conversationDriver) Cancel(context.Context, string) error { return nil }
func (d *conversationDriver) Close() error                         { d.closed.Store(true); return nil }

func conversationBroker(t *testing.T, firstOpenFails bool) (*Broker, Profile, Factory) {
	t.Helper()
	root := t.TempDir()
	program := []byte("fixture: never execute\n")
	path := filepath.Join(root, "provider")
	if err := os.WriteFile(path, program, 0700); err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256(program)
	profile := Profile{ID: "fixture", Provider: "codex", Executable: path, SHA256: hex.EncodeToString(sum[:]), ReviewedVersion: "fixture", BillingReviewed: true}
	var builds atomic.Int64
	factory := func(_ Profile, sink Sink, ask Ask) (Driver, error) {
		return &conversationDriver{sink: sink, ask: ask, fail: firstOpenFails && builds.Add(1) == 1}, nil
	}
	b, err := NewBroker(testStorage(t), []Profile{profile}, func(context.Context, Create, string, bool) (string, error) { return root, nil }, factory)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { b.Close() })
	return b, profile, factory
}

func createConversation(t *testing.T, b *Broker, key string) Session {
	t.Helper()
	s, err := b.Create(context.Background(), key, scope("fixture"))
	if err != nil {
		t.Fatal(err)
	}
	waitState(t, b, s.ID, "ready")
	s, _ = b.Get(bg, s.ID)
	return s
}

func TestConversationMetadataArchiveAndResumeKeepIdentity(t *testing.T) {
	b, profile, factory := conversationBroker(t, false)
	s := createConversation(t, b, "first")
	title, archive := "运行前检查", true
	updated, err := b.UpdateMetadata(bg, s.ID, MetadataUpdate{ExpectedRevision: 1, Title: &title, Archived: &archive})
	if err != nil || !updated.Archived || updated.Title != title || updated.State != "closed" || updated.MetadataRevision != 2 {
		t.Fatalf("archive=%+v %v", updated, err)
	}
	if err = b.Reconnect(bg, s.ID); !errors.Is(err, ErrConflict) {
		t.Fatalf("archived resume=%v", err)
	}
	if _, err = b.UpdateMetadata(bg, s.ID, MetadataUpdate{ExpectedRevision: 1, Title: &title}); !errors.Is(err, ErrConflict) {
		t.Fatalf("stale metadata=%v", err)
	}
	if len(b.sessions[s.ID].turns) > 0 || len(b.sessions[s.ID].inputs) > 0 {
		t.Fatal("archived transcript retained live memory or writer")
	}
	b.Close()
	restored, err := NewBroker(b.store, []Profile{profile}, b.prepare, factory)
	if err != nil {
		t.Fatal(err)
	}
	defer restored.Close()
	restoredInfo, _ := restored.Get(bg, s.ID)
	if restoredInfo.ID != s.ID || restoredInfo.AgentSessionID != s.AgentSessionID || restoredInfo.RuntimeID != s.RuntimeID || restoredInfo.Title != title || !restoredInfo.Archived {
		t.Fatalf("restored=%+v", restoredInfo)
	}
	archive = false
	if _, err = restored.UpdateMetadata(bg, s.ID, MetadataUpdate{ExpectedRevision: 2, Archived: &archive}); err != nil {
		t.Fatal(err)
	}
	if err = restored.Reconnect(bg, s.ID); err != nil {
		t.Fatal(err)
	}
	waitState(t, restored, s.ID, "ready")
	resumed, _ := restored.Get(bg, s.ID)
	if resumed.ID != s.ID || resumed.RuntimeID == s.RuntimeID || resumed.AgentSessionID != s.AgentSessionID || resumed.Archived {
		t.Fatalf("resumed=%+v", resumed)
	}
	if err = restored.CloseSession(bg, s.ID); err != nil {
		t.Fatal(err)
	}
	if err = restored.Reconnect(bg, s.ID); err != nil {
		t.Fatal(err)
	}
	waitState(t, restored, s.ID, "ready")
}

func TestInitialOpenFailureCanExplicitlyRetrySameConversation(t *testing.T) {
	b, _, _ := conversationBroker(t, true)
	s, err := b.Create(context.Background(), "first", scope("fixture"))
	if err != nil {
		t.Fatal(err)
	}
	waitState(t, b, s.ID, "disconnected")
	failed, _ := b.Get(bg, s.ID)
	if failed.AgentSessionID != "" {
		t.Fatal("failed open fabricated native identity")
	}
	if err = b.Reconnect(bg, s.ID); err != nil {
		t.Fatal(err)
	}
	waitState(t, b, s.ID, "ready")
	retried, _ := b.Get(bg, s.ID)
	if retried.ID != s.ID || retried.RuntimeID == failed.RuntimeID || retried.AgentSessionID == "" {
		t.Fatalf("retry=%+v", retried)
	}
	if err = b.receive(b.sessions[s.ID], Event{Kind: "notice", RuntimeID: failed.RuntimeID, Text: "stale old provider"}); !errors.Is(err, ErrStale) {
		t.Fatalf("stale runtime event=%v", err)
	}
}

func TestMetadataRevisionSerializesConcurrentEditors(t *testing.T) {
	b, _, _ := conversationBroker(t, false)
	s := createConversation(t, b, "edit")
	var success, conflict atomic.Int64
	var group sync.WaitGroup
	for _, title := range []string{"first editor", "second editor"} {
		group.Add(1)
		go func(title string) {
			defer group.Done()
			_, err := b.UpdateMetadata(bg, s.ID, MetadataUpdate{ExpectedRevision: 1, Title: &title})
			if err == nil {
				success.Add(1)
			} else if errors.Is(err, ErrConflict) {
				conflict.Add(1)
			} else {
				t.Errorf("metadata error: %v", err)
			}
		}(title)
	}
	group.Wait()
	if success.Load() != 1 || conflict.Load() != 1 {
		t.Fatalf("success=%d conflict=%d", success.Load(), conflict.Load())
	}
}

func TestClosedHistoryDoesNotConsumeWorkerCapacityAndPagesStayScoped(t *testing.T) {
	b, profile, factory := conversationBroker(t, false)
	for i := 0; i < MaxSessions+2; i++ {
		s := createConversation(t, b, fmt.Sprintf("history-%03d", i))
		if err := b.CloseSession(bg, s.ID); err != nil {
			t.Fatal(err)
		}
	}
	foreign := scope("fixture")
	foreign.Context.ID = "another-project"
	other, err := b.Create(context.Background(), "foreign", foreign)
	if err != nil {
		t.Fatal(err)
	}
	waitState(t, b, other.ID, "ready")
	b.Close()
	restored, err := NewBroker(b.store, []Profile{profile}, b.prepare, factory)
	if err != nil {
		t.Fatal(err)
	}
	defer restored.Close()
	for _, s := range restored.sessions {
		if len(s.turns) > 0 || len(s.inputs) > 0 {
			t.Fatal("startup retained historical content or descriptors")
		}
	}
	seen := map[string]bool{}
	cursor := ""
	for {
		page, err := restored.List(bg, SessionListOptions{Limit: 17, After: cursor, Context: scope("fixture").Context})
		if err != nil {
			t.Fatal(err)
		}
		if len(page.Sessions) > 17 {
			t.Fatal("page exceeded requested bound")
		}
		for _, s := range page.Sessions {
			if seen[s.ID] || s.ID == other.ID {
				t.Fatal("duplicate or foreign conversation")
			}
			seen[s.ID] = true
		}
		if page.NextCursor == "" {
			break
		}
		cursor = page.NextCursor
	}
	if len(seen) != MaxSessions+2 {
		t.Fatalf("history count=%d", len(seen))
	}
	if _, err := restored.List(bg, SessionListOptions{Limit: 101}); err == nil {
		t.Fatal("unbounded page accepted")
	}
	createConversation(t, restored, "new-after-history")
}

func TestWorkerCapacityIsStillEnforced(t *testing.T) {
	b, _, _ := conversationBroker(t, false)
	var first string
	for i := 0; i < MaxSessions; i++ {
		s := createConversation(t, b, fmt.Sprintf("active-%03d", i))
		if first == "" {
			first = s.ID
		}
	}
	if _, err := b.Create(context.Background(), "over-capacity", scope("fixture")); err == nil {
		t.Fatal("active capacity was bypassed")
	}
	if err := b.CloseSession(bg, first); err != nil {
		t.Fatal(err)
	}
	createConversation(t, b, "reclaimed-slot")
}

func TestDecisionAuditAndGlobalAttentionRemainSeparateFromFreeText(t *testing.T) {
	b, profile, factory := conversationBroker(t, false)
	s := createConversation(t, b, "audit")
	for index, test := range []struct {
		answer  Answer
		outcome string
	}{{Answer{OptionID: "yes"}, "allow"}, {Answer{OptionID: "no"}, "deny"}, {Answer{Cancel: true}, "cancel"}} {
		if _, err := b.Prompt(bg, s.ID, fmt.Sprintf("prompt-%d", index), PromptRequest{Text: "request fixture"}); err != nil {
			t.Fatal(err)
		}
		waitState(t, b, s.ID, "awaiting-input")
		pending, err := b.Inputs(bg, s.ID)
		if err != nil || len(pending) != 1 {
			t.Fatalf("pending=%+v %v", pending, err)
		}
		request := pending[0].Request
		attention, _ := b.Attention(bg)
		if len(attention.Sessions) != 1 || len(attention.Sessions[0].Pending) != 1 || attention.Sessions[0].Pending[0].ID != request.ID {
			t.Fatalf("attention=%+v", attention)
		}
		data, _ := json.Marshal(attention)
		if strings.Contains(string(data), "sensitive request detail") {
			t.Fatal("attention copied original request text")
		}
		inputs, err := b.Inputs(bg, s.ID)
		if err != nil || len(inputs) != 1 || inputs[0].Request.Text != "sensitive request detail" {
			t.Fatalf("inputs=%+v %v", inputs, err)
		}
		ctx := WithDecisionPolicy(WithDecisionActor(context.Background(), DecisionActor{ID: "station-user", Label: "Operator"}), DecisionPolicy{ID: "manual", Revision: "r1"})
		if err = b.Answer(ctx, s.ID, request.ID, test.answer); err != nil {
			t.Fatal(err)
		}
		if err = b.Answer(ctx, s.ID, request.ID, test.answer); err != nil {
			t.Fatalf("duplicate answer=%v", err)
		}
		waitState(t, b, s.ID, "ready")
	}
	b.Close()
	restored, err := NewBroker(b.store, []Profile{profile}, b.prepare, factory)
	if err != nil {
		t.Fatal(err)
	}
	defer restored.Close()
	events, _, err := restored.replay(s.ID, 0)
	if err != nil {
		t.Fatal(err)
	}
	decisions := []string{}
	for _, event := range events {
		if event.Kind == "input.submitted" {
			if event.Decision == nil || event.Decision.Actor == nil || event.Decision.Actor.ID != "station-user" || event.Decision.PolicyRevision != "r1" {
				t.Fatalf("decision=%+v", event.Decision)
			}
			decisions = append(decisions, event.Decision.Outcome)
		}
	}
	if strings.Join(decisions, ",") != "allow,deny,cancel" {
		t.Fatalf("decisions=%v", decisions)
	}
	secret := decisionReceipt(context.Background(), Request{Questions: []Question{{ID: "q"}}}, Answer{Answers: map[string][]string{"q": {"do not journal this answer"}}})
	data, _ := json.Marshal(secret)
	if strings.Contains(string(data), "do not journal") || secret.Actor != nil {
		t.Fatal("free text or fabricated actor persisted")
	}
}

func TestConversationServiceRejectsForgedActorAndPagesConversations(t *testing.T) {
	b, _, _ := conversationBroker(t, false)
	s := createConversation(t, b, "http")
	handler := Handler(b, HandlerOptions{})
	call := func(method, path, body string) *httptest.ResponseRecorder {
		r := httptest.NewRequest(method, path, strings.NewReader(body))
		r.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, r)
		return w
	}
	response := call("GET", "/sessions?limit=1", "")
	if response.Code != 200 || !strings.Contains(response.Body.String(), `"sessions"`) {
		t.Fatalf("page=%s", response.Body.String())
	}
	response = call("GET", "/attention", "")
	var attention AttentionSnapshot
	if response.Code != 200 || json.Unmarshal(response.Body.Bytes(), &attention) != nil || attention.Revision == "" {
		t.Fatalf("attention missing revision: %s", response.Body.String())
	}
	response = call("POST", "/sessions/"+s.ID+"/inputs/q_forged", `{"optionId":"yes","actor":{"id":"admin"}}`)
	if response.Code != 400 || !strings.Contains(response.Body.String(), `"invalid_argument"`) {
		t.Fatalf("forged actor status=%d", response.Code)
	}
}
