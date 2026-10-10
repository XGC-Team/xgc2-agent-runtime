package agentruntime

import (
	"fmt"
	"strings"
	"sync"
	"testing"
)

// toolStreamFixture drives the ACP tool updates of one turn into a recorded journal.
type toolStreamFixture struct {
	t      *testing.T
	d      *rpcDriver
	mu     sync.Mutex
	events []Event
}

func newToolStream(t *testing.T) *toolStreamFixture {
	f := &toolStreamFixture{t: t}
	f.d = &rpcDriver{profile: Profile{Provider: "opencode"}, turn: "t"}
	f.d.sink = func(e Event) error { f.mu.Lock(); f.events = append(f.events, e); f.mu.Unlock(); return nil }
	return f
}

func (f *toolStreamFixture) update(kind, id, status, output string) {
	u := map[string]any{"sessionUpdate": kind, "toolCallId": id, "content": []any{map[string]any{"type": "content", "content": map[string]any{"type": "text", "text": output}}}}
	if status != "" {
		u["status"] = status
	}
	f.d.acpEvent(u)
}

// shown is what the conversation view reconstructs: snapshots replace, deltas append.
func (f *toolStreamFixture) shown() map[string]string {
	f.mu.Lock()
	defer f.mu.Unlock()
	items := map[string]string{}
	for _, e := range f.events {
		switch e.Kind {
		case "item.snapshot":
			items[e.ItemID] = e.Text
		case "item.delta":
			items[e.ItemID] += e.Text
		}
	}
	return items
}

func (f *toolStreamFixture) stats() (count, bytes int) {
	f.mu.Lock()
	defer f.mu.Unlock()
	for _, e := range f.events {
		count++
		bytes += len(e.Text)
	}
	return
}

func TestGrowingToolOutputIsJournaledLinearly(t *testing.T) {
	f := newToolStream(t)
	var output string
	f.update("tool_call", "build", "in_progress", "")
	for i := 0; i < 400; i++ {
		output += fmt.Sprintf("line %03d of the build log\n", i)
		f.update("tool_call_update", "build", "in_progress", strings.TrimSuffix(output, "\n"))
	}
	f.update("tool_call_update", "build", "completed", strings.TrimSuffix(output, "\n"))
	count, bytes := f.stats()
	if want := strings.TrimSuffix(output, "\n"); f.shown()["build"] != want {
		t.Fatalf("view differs from the agent's output:\n%q", f.shown()["build"])
	}
	// Quadratic journaling would store ~200 times the output; appends cost the output once, plus the final snapshot.
	if bytes > 2*len(output)+64 || count != 402 {
		t.Fatalf("%d events carry %d bytes for %d bytes of output", count, bytes, len(output))
	}
	if f.events[1].Kind != "item.delta" || f.events[len(f.events)-1].Kind != "item.snapshot" {
		t.Fatalf("kinds: %s ... %s", f.events[1].Kind, f.events[len(f.events)-1].Kind)
	}
}

func TestUnchangedToolOutputSendsNothingAndStatusChangesAlwaysDo(t *testing.T) {
	f := newToolStream(t)
	f.update("tool_call", "read", "pending", "same")
	for i := 0; i < 5; i++ {
		f.update("tool_call_update", "read", "pending", "same")
	}
	if count, _ := f.stats(); count != 1 {
		t.Fatalf("%d events for an update that changed nothing", count)
	}
	f.update("tool_call_update", "read", "in_progress", "same")
	f.d.acpEvent(map[string]any{"sessionUpdate": "tool_call_update", "toolCallId": "read", "status": "completed"})
	if count, _ := f.stats(); count != 3 || f.events[1].Status != "in_progress" || f.events[2].Kind != "item.patch" {
		t.Fatalf("events %+v", f.events)
	}
}

func TestRewrittenToolOutputIsThrottledAndTheLastRewriteIsNeverLost(t *testing.T) {
	f := newToolStream(t)
	f.update("tool_call", "diff", "in_progress", "rewrite 0")
	for i := 1; i <= 25; i++ {
		f.update("tool_call_update", "diff", "in_progress", fmt.Sprintf("rewrite %d", i)) // not an extension of the previous text
	}
	if count, _ := f.stats(); count > 4 {
		t.Fatalf("%d events for 25 rewrites", count)
	}
	// The turn ends before the agent says the tool is done: nothing the agent last showed is lost.
	f.d.flushTools()
	if got := f.shown()["diff"]; got != "rewrite 25" {
		t.Fatalf("view ends with %q", got)
	}
	f.d.flushTools() // flushing twice sends nothing new
	if count, _ := f.stats(); count > 5 {
		t.Fatalf("%d events", count)
	}
}

func TestCodexTurnDiffUsesTheSameEconomy(t *testing.T) {
	f := newToolStream(t)
	f.d.profile = Profile{Provider: "codex"}
	f.d.nativeSession, f.d.nativeTurn = "n", "nt"
	diff := ""
	for i := 0; i < 200; i++ {
		diff += fmt.Sprintf("+ changed line %d\n", i)
		f.d.codexEvent("turn/diff/updated", map[string]any{"threadId": "n", "turnId": "nt", "diff": diff})
	}
	count, bytes := f.stats()
	if f.shown()["turn-diff"] != diff || bytes > 2*len(diff) || count != 200 {
		t.Fatalf("%d events carry %d bytes for %d bytes of diff", count, bytes, len(diff))
	}
}
