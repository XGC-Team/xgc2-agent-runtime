package nativeagent

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

type privateBindingKey struct{}
type bindingValueDriver struct {
	*scopedFixtureDriver
	value chan string
}

func (d *bindingValueDriver) BindSession(ctx context.Context, scope Create, id string) error {
	value, _ := ctx.Value(privateBindingKey{}).(string)
	if ctx.Err() != nil {
		return ctx.Err()
	}
	d.value <- value
	return d.scopedFixtureDriver.BindSession(ctx, scope, id)
}
func TestBrokerCarriesEphemeralBindingValuesWithoutRequestLifetimeOrJournal(t *testing.T) {
	root := scopePrivateDirectory(t)
	workspace := t.TempDir()
	profile := scopeProfile(t)
	bound := make(chan scopeReceipt, 4)
	opened := make(chan scopeReceipt, 4)
	values := make(chan string, 4)
	prepareEntered := make(chan struct{})
	prepareRelease := make(chan struct{})
	freshPrepare := true
	prepare := func(ctx context.Context, _ Create, _ string, _ bool) (string, error) {
		if freshPrepare {
			freshPrepare = false
			close(prepareEntered)
			<-prepareRelease
		}
		return workspace, ctx.Err()
	}
	factory := func(_ Profile, sink Sink, _ Ask) (Driver, error) {
		return &bindingValueDriver{scopedFixtureDriver: &scopedFixtureDriver{sink: sink, bound: bound, opened: opened, closed: make(chan struct{})}, value: values}, nil
	}
	broker, err := NewBroker(root, []Profile{profile}, prepare, factory)
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.WithValue(context.Background(), privateBindingKey{}, "one-use-private-enrollment"))
	scope := Create{ProfileID: profile.ID, Context: ContextRef{Kind: "experiment", ID: "exp-a"}, Workspace: WorkspaceRef{ID: "workspace", Revision: "v1"}, NativeAccessConfirmed: true}
	session, err := broker.Create(ctx, "context-test", scope)
	if err != nil {
		broker.Close()
		t.Fatal(err)
	}
	<-prepareEntered
	cancel()
	close(prepareRelease)
	scopeWait(t, bound)
	scopeWait(t, opened)
	scopeWaitState(t, broker, session.ID, "ready")
	if got := <-values; got != "one-use-private-enrollment" {
		t.Fatalf("lost ephemeral binding: %q", got)
	}
	if err := broker.Close(); err != nil {
		t.Fatal(err)
	}
	journal, err := os.ReadFile(filepath.Join(root, session.ID+".jsonl"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(journal), "one-use-private-enrollment") {
		t.Fatal("enrollment persisted")
	}
	restarted, err := NewBroker(root, []Profile{profile}, prepare, factory)
	if err != nil {
		t.Fatal(err)
	}
	defer restarted.Close()
	if err := restarted.ReconnectContext(context.WithValue(context.Background(), privateBindingKey{}, "renewed-private-enrollment"), session.ID); err != nil {
		t.Fatal(err)
	}
	scopeWait(t, bound)
	scopeWait(t, opened)
	scopeWaitState(t, restarted, session.ID, "ready")
	if got := <-values; got != "renewed-private-enrollment" {
		t.Fatalf("resume did not carry fresh context: %q", got)
	}
	restarted.Close()
	last, err := NewBroker(root, []Profile{profile}, prepare, factory)
	if err != nil {
		t.Fatal(err)
	}
	defer last.Close()
	if err := last.Reconnect(session.ID); err != nil {
		t.Fatal(err)
	}
	scopeWait(t, bound)
	scopeWait(t, opened)
	scopeWaitState(t, last, session.ID, "ready")
	if got := <-values; got != "" {
		t.Fatal("reconnect recovered a prior private grant")
	}
}
func TestSessionBindingLifetimeBelongsToBroker(t *testing.T) {
	parent, cancel := context.WithCancel(context.Background())
	ctx := sessionBindingContext{Context: parent, values: context.WithValue(context.Background(), privateBindingKey{}, "value")}
	cancel()
	if ctx.Err() == nil {
		t.Fatal("broker cancellation lost")
	}
}
