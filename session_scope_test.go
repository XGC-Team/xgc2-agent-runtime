package agentruntime

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"os"
	"sync"
	"testing"
	"time"
)

type scopeReceipt struct {
	scope  Create
	id     string
	native string
}
type scopedFixtureDriver struct {
	sink      Sink
	bound     chan scopeReceipt
	opened    chan scopeReceipt
	closed    chan struct{}
	bindError bool
	scope     Create
	id        string
	once      sync.Once
}

func (d *scopedFixtureDriver) BindSession(_ context.Context, scope Create, id string) error {
	d.scope = scope
	d.id = id
	d.bound <- scopeReceipt{scope: scope, id: id}
	if d.bindError {
		return errors.New("binding rejected")
	}
	return nil
}
func (d *scopedFixtureDriver) Open(_ context.Context, _ string, native string) error {
	if d.id == "" {
		return errors.New("opened without exact session scope")
	}
	d.opened <- scopeReceipt{scope: d.scope, id: d.id, native: native}
	return d.sink(Event{Kind: "session.identity", AgentSessionID: "native-retained"})
}
func (d *scopedFixtureDriver) Prompt(context.Context, string, string) error {
	return errors.New("unexpected prompt replay")
}
func (d *scopedFixtureDriver) Cancel(context.Context) error { return nil }
func (d *scopedFixtureDriver) Close() error                 { d.once.Do(func() { close(d.closed) }); return nil }
func scopeProfile(t *testing.T) Profile {
	t.Helper()
	executable, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(executable)
	if err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256(raw)
	return Profile{ID: "scoped", Provider: "codex", Executable: executable, SHA256: hex.EncodeToString(sum[:]), ReviewedVersion: "protocol-fixture", BillingReviewed: true}
}
func scopeWait(t *testing.T, ch <-chan scopeReceipt) scopeReceipt {
	t.Helper()
	select {
	case value := <-ch:
		return value
	case <-time.After(3 * time.Second):
		t.Fatal("scope hook did not complete")
		return scopeReceipt{}
	}
}
func scopeWaitState(t *testing.T, b *Broker, id, state string) {
	t.Helper()
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		session, err := b.Get(id)
		if err == nil && session.State == state {
			return
		}
		time.Sleep(time.Millisecond)
	}
	t.Fatalf("session did not reach %s", state)
}
func TestBrokerBindsPersistedScopeBeforeOpenAndOnResume(t *testing.T) {
	profile := scopeProfile(t)
	root := scopePrivateDirectory(t)
	workspace := t.TempDir()
	scope := Create{ProfileID: profile.ID, Context: ContextRef{Kind: "experiment", ID: "experiment-a"}, Workspace: WorkspaceRef{ID: "workspace", Revision: "reviewed-v1"}, AgentAccessConfirmed: true}
	bound := make(chan scopeReceipt, 4)
	opened := make(chan scopeReceipt, 4)
	closed := make(chan struct{})
	prepare := func(_ context.Context, request Create, _ string, _ bool) (string, error) {
		if request != scope {
			return "", errors.New("scope changed")
		}
		return workspace, nil
	}
	factory := func(_ Profile, sink Sink, _ Ask) (Driver, error) {
		return &scopedFixtureDriver{sink: sink, bound: bound, opened: opened, closed: closed}, nil
	}
	broker, err := NewBroker(root, []Profile{profile}, prepare, factory)
	if err != nil {
		t.Fatal(err)
	}
	session, err := broker.Create(context.Background(), "creation-key", scope)
	if err != nil {
		broker.Close()
		t.Fatal(err)
	}
	first := scopeWait(t, bound)
	if first.scope != scope || first.id != session.ID {
		t.Fatal("factory did not receive durable scope")
	}
	initial := scopeWait(t, opened)
	if initial.native != "" {
		t.Fatal("fresh session resumed something")
	}
	scopeWaitState(t, broker, session.ID, "ready")
	if err := broker.Close(); err != nil {
		t.Fatal(err)
	}
	select {
	case <-closed:
	default:
		t.Fatal("host shutdown did not close bound driver")
	}
	// The persisted identity, not current UI selection, authorizes resumed access.
	closed = make(chan struct{})
	resumed, err := NewBroker(root, []Profile{profile}, prepare, factory)
	if err != nil {
		t.Fatal(err)
	}
	defer resumed.Close()
	if err := resumed.Reconnect(session.ID); err != nil {
		t.Fatal(err)
	}
	second := scopeWait(t, bound)
	if second.scope != scope || second.id != session.ID {
		t.Fatal("resume changed scope")
	}
	receipt := scopeWait(t, opened)
	if receipt.native != "native-retained" {
		t.Fatal("native identity not resumed")
	}
	scopeWaitState(t, resumed, session.ID, "ready")
	if err := resumed.CloseSession(session.ID); err != nil {
		t.Fatal(err)
	}
	select {
	case <-closed:
	default:
		t.Fatal("session close did not revoke driver resources")
	}
}
func TestRejectedSessionBindingNeverOpensAndIsClosed(t *testing.T) {
	profile := scopeProfile(t)
	bound := make(chan scopeReceipt, 1)
	opened := make(chan scopeReceipt, 1)
	closed := make(chan struct{})
	broker, err := NewBroker(scopePrivateDirectory(t), []Profile{profile}, func(context.Context, Create, string, bool) (string, error) { return t.TempDir(), nil }, func(_ Profile, sink Sink, _ Ask) (Driver, error) {
		return &scopedFixtureDriver{sink: sink, bound: bound, opened: opened, closed: closed, bindError: true}, nil
	})
	if err != nil {
		t.Fatal(err)
	}
	defer broker.Close()
	session, err := broker.Create(context.Background(), "key", Create{ProfileID: profile.ID, Context: ContextRef{Kind: "experiment", ID: "experiment"}, Workspace: WorkspaceRef{ID: "workspace", Revision: "v1"}, AgentAccessConfirmed: true})
	if err != nil {
		t.Fatal(err)
	}
	scopeWait(t, bound)
	scopeWaitState(t, broker, session.ID, "disconnected")
	select {
	case <-opened:
		t.Fatal("opened after rejected scope")
	default:
	}
	select {
	case <-closed:
	case <-time.After(time.Second):
		t.Fatal("failed binding leaked driver")
	}
}

func scopePrivateDirectory(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	if err := os.Chmod(root, 0700); err != nil {
		t.Fatal(err)
	}
	return root
}
