package agentruntime

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/XGC-Team/xgc2-storage/api"
	"github.com/XGC-Team/xgc2-storage/client"
	"github.com/XGC-Team/xgc2-storage/engine"
	"github.com/XGC-Team/xgc2-storage/server"
	xrpc "github.com/XGC-Team/xgc2-xrpc/go"
	"github.com/XGC-Team/xgc2-xrpc/go/httpx"
	unixlease "github.com/XGC-Team/xgc2-xrpc/go/unix"
)

func testStorage(t *testing.T) *Store { return testStorageDelay(t, 0) }

// testStorageDelay is testStorage with the given journal delay (zero: the default).
func testStorageDelay(t *testing.T, delay time.Duration) *Store {
	t.Helper()
	dir, err := os.MkdirTemp("", "sol21-storage-")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.RemoveAll(dir) })
	var manifest api.Manifest
	raw, err := os.ReadFile("storage-manifest.json")
	if err != nil {
		t.Fatal(err)
	}
	if err = json.Unmarshal(raw, &manifest); err != nil {
		t.Fatal(err)
	}
	engineStore, err := engine.Open(context.Background(), engine.Config{Path: filepath.Join(dir, "storage.db"), Create: true, Manifest: manifest})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { engineStore.Close() })
	lease, err := unixlease.Reserve(context.Background(), filepath.Join(dir, "rpc.sock"), unixlease.Options{})
	if err != nil {
		t.Fatal(err)
	}
	listener, err := lease.Listen()
	if err != nil {
		lease.Close()
		t.Fatal(err)
	}
	grant := "0123456789abcdef0123456789abcdef"
	scope := api.Scope{Namespace: StorageNamespace, User: "fixture", Workspace: "conversations"}
	handler, err := server.HTTP(engineStore, []server.Grant{{Token: grant, Namespace: scope.Namespace, User: scope.User, Workspace: scope.Workspace}})
	if err != nil {
		t.Fatal(err)
	}
	ref := xrpc.ServiceRef{TargetID: "fixture", Service: api.Service, APIVersion: api.Version, InstanceID: "fixture-instance", Profile: xrpc.HTTP, Endpoint: xrpc.Endpoint{Kind: "unix", Address: filepath.Join(dir, "rpc.sock")}}
	host, err := httpx.Serve(listener, lease, handler, httpx.HostOptions{InstanceID: ref.InstanceID, MaxBodyBytes: api.MaxRequestBytes, MaxResponseBytes: api.MaxResponseBytes})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		host.Shutdown(ctx)
	})
	transport, err := httpx.New(httpx.Config{LocalTargetID: ref.TargetID, Service: ref, MaxRequestBytes: api.MaxRequestBytes, MaxResponseBytes: api.MaxResponseBytes})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(transport.Close)
	consumer, err := client.New(client.HTTPCaller{Transport: transport, Grant: grant}, ref)
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	read, err := consumer.Snapshot(ctx, "bootstrap", api.SnapshotRequest{Scope: scope, Queries: []api.Query{{Collection: "sessions", Keys: []string{"unused"}}}})
	if err != nil {
		t.Fatal(err)
	}
	store, err := NewStore(StorageBinding{Client: consumer, Scope: scope, DatabaseID: read.Token.DatabaseID, Schema: StorageSchema, JournalDelay: delay})
	if err != nil {
		t.Fatal(err)
	}
	return store
}

func testSession(t *testing.T, id string) *liveSession { return testSessionDelay(t, id, 0) }

func testSessionDelay(t *testing.T, id string, delay time.Duration) *liveSession {
	t.Helper()
	store := testStorageDelay(t, delay)
	s := newSession(Session{SchemaVersion: Schema, ID: id, Provider: "codex", Scope: scope("codex"), MetadataRevision: 1}, store)
	s.profile = testProfile(t, "codex")
	version, err := store.create(context.Background(), s.record())
	if err != nil {
		t.Fatal(err)
	}
	s.version = version
	return s
}

// commit makes everything the session appended so far durable, the way a
// command that is acknowledged only once its events are on disk does.
func commit(t testing.TB, s *liveSession) {
	t.Helper()
	if err := s.flushAll(bg); err != nil {
		t.Fatal(err)
	}
}

// stored reads events straight from storage, bypassing the journal's pending list.
func stored(t testing.TB, s *liveSession, after, end uint64) []Event {
	t.Helper()
	events, err := s.store.readEvents(s.id, after, end)
	if err != nil {
		t.Fatal(err)
	}
	return events
}
