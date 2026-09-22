package agentruntime

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func promptReplayFixture(t *testing.T) []Event {
	t.Helper()
	data, err := os.ReadFile("testdata/prompt-options-replay.json")
	if err != nil {
		t.Fatal(err)
	}
	var events []Event
	if err = json.Unmarshal(data, &events); err != nil || len(events) != 9 {
		t.Fatalf("invalid shared replay fixture: %v", err)
	}
	return events
}

func TestPromptReceiptDetailsHaveCanonicalTypeAndPreserveOptions(t *testing.T) {
	legacy := promptReplayFixture(t)[2]
	selected := optionsFromDetails(legacy.Details)
	if selected != (AgentOptions{Model: "gpt-5.3-codex-spark", Effort: "high"}) {
		t.Fatal("shared fixture lost the original options shape")
	}
	legacy.Details["type"] = "userMessage"
	if !reflect.DeepEqual(promptDetails(selected), legacy.Details) {
		t.Fatalf("producer differs from the Web replay contract: %+v", promptDetails(selected))
	}
	selected.Permission = "approval-required"
	encoded, err := json.Marshal(promptDetails(selected))
	if err != nil {
		t.Fatal(err)
	}
	var decoded map[string]any
	if err = json.Unmarshal(encoded, &decoded); err != nil || optionsFromDetails(decoded) != selected {
		t.Fatalf("serialized receipt lost idempotency selections: %s %v", encoded, err)
	}
	if promptDetails(AgentOptions{}) != nil {
		t.Fatal("a prompt without selections acquired invented metadata")
	}
}

func TestPersistedPromptOptionsReplayKeepsJournalAndIdempotency(t *testing.T) {
	for _, typed := range []bool{false, true} {
		name := "existing-untagged-journal"
		if typed {
			name = "typed-journal"
		}
		t.Run(name, func(t *testing.T) {
			events := promptReplayFixture(t)
			id, key := events[0].SessionID, "fixture-retry"
			turn := "t_" + hash(id + "\x00" + key)[:32]
			selected := optionsFromDetails(events[2].Details)
			if typed {
				events[2].Details = promptDetails(selected)
			}
			for i := range events {
				if events[i].TurnID != "" {
					events[i].TurnID = turn
				}
			}
			var journal bytes.Buffer
			encoder := json.NewEncoder(&journal)
			if err := encoder.Encode(sessionRecord{Session: Session{SchemaVersion: Schema, ID: id, Scope: scope("codex"), Provider: "codex", State: "starting", CreatedAt: events[0].CreatedAt}}); err != nil {
				t.Fatal(err)
			}
			for _, event := range events {
				if err := encoder.Encode(event); err != nil {
					t.Fatal(err)
				}
			}
			root := filepath.Join(t.TempDir(), "journal")
			if err := os.Mkdir(root, 0700); err != nil {
				t.Fatal(err)
			}
			path := filepath.Join(root, id+".jsonl")
			if err := os.WriteFile(path, journal.Bytes(), 0600); err != nil {
				t.Fatal(err)
			}
			broker, err := NewBroker(root, nil,
				func(context.Context, Create, string, bool) (string, error) {
					t.Error("journal replay must not prepare a workspace")
					return "", errors.New("unexpected preparation")
				}, func(Profile, Sink, Ask) (Driver, error) {
					t.Error("journal replay must not launch a native client")
					return nil, errors.New("unexpected native launch")
				})
			if err != nil {
				t.Fatal(err)
			}
			defer broker.Close()
			replayed, _, err := broker.Replay(id, 0)
			if err != nil || len(replayed) != len(events)+1 || !reflect.DeepEqual(replayed[:len(events)], events) || replayed[len(events)].Status != "disconnected" {
				t.Fatalf("persisted events changed during restart: %v", err)
			}
			if got, err := broker.PromptWithOptions(id, key, events[2].Text, selected); err != nil || got != turn {
				t.Fatalf("identical retry was not restored: %s %v", got, err)
			}
			changed := selected
			changed.Effort = "low"
			if _, err := broker.PromptWithOptions(id, key, events[2].Text, changed); !errors.Is(err, ErrConflict) {
				t.Fatalf("changed selections did not conflict: %v", err)
			}
			if _, err := broker.PromptWithOptions(id, key, "Changed fixture prompt", selected); !errors.Is(err, ErrConflict) {
				t.Fatalf("changed text did not conflict: %v", err)
			}
			if _, err := broker.PromptWithOptions(id, key, events[2].Text, AgentOptions{}); !errors.Is(err, ErrConflict) {
				t.Fatalf("dropped selections did not conflict: %v", err)
			}
			after, _, _ := broker.Replay(id, 0)
			if !reflect.DeepEqual(after, replayed) {
				t.Fatal("idempotent retries appended or resent a prompt")
			}
			preserved, err := os.ReadFile(path)
			if err != nil || !bytes.HasPrefix(preserved, journal.Bytes()) {
				t.Fatalf("existing journal bytes were rewritten: %v", err)
			}
		})
	}
}
