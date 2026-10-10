package agentruntime

import (
	"context"
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
