package agentruntime

import (
	"strings"
	"testing"
)

func TestCodexTerminalRetainsUserFacingErrorWithoutDiagnostics(t *testing.T) {
	d := &rpcDriver{profile: Profile{Provider: "codex"}, turn: "local", nativeSession: "thread", nativeTurn: "turn", turnDone: make(chan nativeTerminal, 1)}
	message := "You've hit your usage limit. Try again at 9:10 PM."
	d.onNotification("turn/completed", map[string]any{"threadId": "other", "turn": map[string]any{"id": "turn", "status": "failed", "error": map[string]any{"message": "foreign"}}})
	if len(d.turnDone) != 0 {
		t.Fatal("foreign terminal accepted")
	}
	d.onNotification("turn/completed", map[string]any{"threadId": "thread", "turn": map[string]any{"id": "turn", "status": "failed", "error": map[string]any{"message": message, "additionalDetails": "private diagnostic"}}})
	terminal := <-d.turnDone
	if terminal.status != "failed" || terminal.message != message {
		t.Fatalf("terminal=%+v", terminal)
	}
	if got := nativeErrorMessage(map[string]any{"message": strings.Repeat("界", 5000)}); len([]rune(got)) != 4096 {
		t.Fatal("unbounded error")
	}
}
