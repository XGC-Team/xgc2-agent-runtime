package agentruntime

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestContextSeparatesIdempotencyAndWorkspaceRemainsProductOwned(t *testing.T) {
	b, first := testBroker(t, "claude")
	for _, ref := range []ContextRef{{Kind: "experiment", ID: "project"}, {Kind: "fixture", ID: "another"}} {
		c := scope("claude")
		c.Context = ref
		// The shared broker accepts a reviewed revision reference; the product
		// prepare callback owns stricter Git/workspace policy.
		c.Workspace.Revision = "reviewed-snapshot-v1"
		s, err := b.Create(context.Background(), "create-1", c)
		if err != nil || s.ID == first.ID || s.Scope != c {
			t.Fatalf("context was not preserved and isolated: %+v, %v", s, err)
		}
		first = s
	}
}

func TestCustomRouteBaseRetainsOriginAndMutationGuards(t *testing.T) {
	b, _ := testBroker(t, "claude")
	mux := http.NewServeMux()
	if err := RegisterRoutes(mux, b, "/api/ground-station/native-agents/"); err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		method, path, header string
		want                 int
	}{
		{"GET", "/api/ground-station/native-agents/providers", "", 200},
		{"GET", "/api/v1/agent-runtime/providers", "", 404},
		{"POST", "/api/ground-station/native-agents/sessions", "", 403},
		{"POST", "/api/ground-station/native-agents/sessions", "X-Research-Native-Client", 403},
		{"POST", "/api/ground-station/native-agents/sessions", ClientHeader, 415},
	} {
		r := httptest.NewRequest(tc.method, "http://localhost:7000"+tc.path, nil)
		r.RemoteAddr = "127.0.0.1:1234"
		if tc.header != "" {
			r.Header.Set(tc.header, "1")
		}
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, r)
		if w.Code != tc.want {
			t.Fatalf("%s %s (%s): got %d, want %d", tc.method, tc.path, tc.header, w.Code, tc.want)
		}
	}
	for _, path := range []string{"", "//remote/api", "/api/../other", "/api?other"} {
		if err := RegisterRoutes(http.NewServeMux(), b, path); err == nil {
			t.Fatalf("invalid route accepted: %q", path)
		}
	}
}
