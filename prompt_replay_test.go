package agentruntime

import (
	"context"
	"encoding/json"
	"errors"
	"os"
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
	receipt := promptReplayFixture(t)[2]
	selected := optionsFromDetails(receipt.Details)
	if selected != (AgentOptions{Model: "gpt-5.3-codex-spark", Effort: "high"}) {
		t.Fatal("shared fixture lost the original options shape")
	}
	receipt.Details["type"] = "userMessage"
	if !reflect.DeepEqual(promptDetails(selected), receipt.Details) {
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
	t.Run("new-model-typed-events", func(t *testing.T) {
		events := promptReplayFixture(t)
		id, key := events[0].SessionID, "fixture-retry"
		turn := "t_" + hash(id + "\x00" + key)[:32]
		selected := optionsFromDetails(events[2].Details)
		for i := range events {
			if events[i].TurnID != "" {
				events[i].TurnID = turn
			}
		}
		store := testStorage(t)
		profile := testProfile(t, "codex")
		live := newSession(Session{SchemaVersion: Schema, ID: id, Scope: scope("codex"), Provider: "codex", State: "starting", CreatedAt: events[0].CreatedAt, MetadataRevision: 1}, store)
		live.profile = profile
		version, err := store.create(context.Background(), live.record())
		if err != nil {
			t.Fatal(err)
		}
		live.version = version
		for _, event := range events {
			if err := live.appendLocked(event); err != nil {
				t.Fatal(err)
			}
		}
		commit(t, live)
		events = stored(t, live, 0, live.info.LastSeq)
		broker, err := NewBroker(store, nil,
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
		replayed, _, err := broker.replay(id, 0)
		if err != nil || !reflect.DeepEqual(replayed, events) {
			t.Fatalf("persisted events changed during restart: %v", err)
		}
		if restarted, err := broker.Get(bg, id); err != nil || restarted.State != "disconnected" {
			t.Fatalf("restart state %+v: %v", restarted, err)
		}
		if got, err := broker.Prompt(bg, id, key, PromptRequest{Text: events[2].Text, Options: selected}); err != nil || got != turn {
			t.Fatalf("identical retry was not restored: %s %v", got, err)
		}
		changed := selected
		changed.Effort = "low"
		if _, err := broker.Prompt(bg, id, key, PromptRequest{Text: events[2].Text, Options: changed}); !errors.Is(err, ErrConflict) {
			t.Fatalf("changed selections did not conflict: %v", err)
		}
		if _, err := broker.Prompt(bg, id, key, PromptRequest{Text: "Changed fixture prompt", Options: selected}); !errors.Is(err, ErrConflict) {
			t.Fatalf("changed text did not conflict: %v", err)
		}
		if _, err := broker.Prompt(bg, id, key, PromptRequest{Text: events[2].Text, Options: AgentOptions{}}); !errors.Is(err, ErrConflict) {
			t.Fatalf("dropped selections did not conflict: %v", err)
		}
		after, _, _ := broker.replay(id, 0)
		if !reflect.DeepEqual(after, replayed) {
			t.Fatal("idempotent retries appended or resent a prompt")
		}

	})
}
