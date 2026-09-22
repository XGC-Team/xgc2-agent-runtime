package nativeagent

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
)

func settingsBroker(t *testing.T, path string) *Broker {
	t.Helper()
	root := t.TempDir()
	b, err := NewBroker(filepath.Join(root, "journal"), nil, func(context.Context, Create, string, bool) (string, error) { return root, nil }, nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { b.Close() })
	if err = ConfigureBroker(b, BrokerOptions{SettingsFile: path}); err != nil {
		t.Fatal(err)
	}
	return b
}
func settingsFixtureUpdate(t *testing.T, revision string, defaults NativeOptions) SettingsUpdate {
	t.Helper()
	p := testProfile(t, "codex")
	return SettingsUpdate{Revision: revision, Provider: ProviderUpdate{ID: "codex", Provider: "codex", Enabled: true, BinaryPath: p.Executable, Defaults: defaults}}
}
func TestSettingsDiscoveryCASAndCrossBrokerVisibility(t *testing.T) {
	path := filepath.Join(t.TempDir(), "native-agents.json")
	a, b := settingsBroker(t, path), settingsBroker(t, path)
	initial, err := a.Settings()
	if err != nil || len(initial.Providers) != 5 {
		t.Fatalf("initial: %+v %v", initial, err)
	}
	for _, p := range initial.Providers {
		if p.Enabled || p.Login.Status != "unknown" || len(p.Models) != 0 {
			t.Fatalf("discovery asserted unprobed state: %+v", p)
		}
	}
	defaults := NativeOptions{Model: "fixture-luna", Effort: "low", Permission: "approval-required"}
	saved, err := a.UpdateSettings(context.Background(), settingsFixtureUpdate(t, initial.Revision, defaults))
	if err != nil {
		t.Fatal(err)
	}
	if saved.Revision == initial.Revision {
		t.Fatal("revision did not change")
	}
	raw, _ := json.Marshal(saved)
	if strings.Contains(string(raw), "never-surface-this") || strings.Contains(string(raw), "sha256") {
		t.Fatalf("private probe data leaked: %s", raw)
	}
	other, err := b.Settings()
	if err != nil || other.Revision != saved.Revision {
		t.Fatalf("cross-broker revision: %+v %v", other, err)
	}
	found := false
	for _, p := range other.Providers {
		if p.ID == "codex" {
			found = p.Enabled && p.Defaults == defaults
		}
	}
	if !found {
		t.Fatal("second broker did not read committed settings")
	}
	if _, err = b.UpdateSettings(context.Background(), settingsFixtureUpdate(t, initial.Revision, defaults)); !errors.Is(err, ErrConflict) {
		t.Fatalf("stale update: %v", err)
	}
	mode, err := os.Stat(path)
	if err != nil || mode.Mode().Perm() != 0600 {
		t.Fatalf("settings mode: %v %v", mode, err)
	}
	profiles, err := LoadConfig(path)
	if err != nil || len(profiles) != 1 || profiles[0].Defaults != defaults {
		t.Fatalf("legacy loader lost settings: %+v %v", profiles, err)
	}
}
func TestSettingsConcurrentIndependentBrokersOnlyOneCASWinner(t *testing.T) {
	path := filepath.Join(t.TempDir(), "native-agents.json")
	a, b := settingsBroker(t, path), settingsBroker(t, path)
	initial, _ := a.Settings()
	start := make(chan struct{})
	results := make(chan error, 2)
	var wg sync.WaitGroup
	for _, broker := range []*Broker{a, b} {
		wg.Add(1)
		go func(broker *Broker) {
			defer wg.Done()
			<-start
			_, err := broker.UpdateSettings(context.Background(), settingsFixtureUpdate(t, initial.Revision, NativeOptions{}))
			results <- err
		}(broker)
	}
	close(start)
	wg.Wait()
	close(results)
	success, conflict := 0, 0
	for err := range results {
		if err == nil {
			success++
		} else if errors.Is(err, ErrConflict) {
			conflict++
		} else {
			t.Fatal(err)
		}
	}
	if success != 1 || conflict != 1 {
		t.Fatalf("success=%d conflict=%d", success, conflict)
	}
}
func TestNativeSelectionsCrossBrokerSnapshotAndIdempotency(t *testing.T) {
	path := filepath.Join(t.TempDir(), "native-agents.json")
	settingsOwner, consumer := settingsBroker(t, path), settingsBroker(t, path)
	initial, _ := settingsOwner.Settings()
	defaults := NativeOptions{Model: "fixture-native", Effort: "medium", Permission: "approval-required"}
	saved, err := settingsOwner.UpdateSettings(context.Background(), settingsFixtureUpdate(t, initial.Revision, defaults))
	if err != nil {
		t.Fatal(err)
	}
	s, err := consumer.Create(context.Background(), "snapshot", scope("codex"))
	if err != nil {
		t.Fatal(err)
	}
	waitState(t, consumer, s.ID, "ready")
	changed := NativeOptions{Model: "fixture-luna", Effort: "low", Permission: "full-access"}
	if _, err = settingsOwner.UpdateSettings(context.Background(), settingsFixtureUpdate(t, saved.Revision, changed)); err != nil {
		t.Fatal(err)
	}
	// Existing session is pinned even after the other broker commits new defaults.
	existing, _ := consumer.Get(s.ID)
	if existing.Options != defaults {
		t.Fatalf("session changed: %+v", existing.Options)
	}
	selected := NativeOptions{Model: "fixture-luna", Effort: "low", Permission: "approval-required"}
	turn, err := consumer.PromptWithOptions(s.ID, "options-turn", "require-options", selected)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = consumer.PromptWithOptions(s.ID, "options-turn", "require-options", NativeOptions{Model: "fixture-native", Effort: "medium"}); !errors.Is(err, ErrConflict) {
		t.Fatalf("options changed replay must conflict: %v", err)
	}
	replay, err := consumer.PromptWithOptions(s.ID, "options-turn", "require-options", selected)
	if err != nil || replay != turn {
		t.Fatalf("replay %s %v", replay, err)
	}
	request := waitRequest(t, consumer, s.ID)
	if err = consumer.Answer(s.ID, request.ID, Answer{OptionID: request.Options[0].ID}); err != nil {
		t.Fatal(err)
	}
	waitState(t, consumer, s.ID, "ready")
	events, _, _ := consumer.Replay(s.ID, 0)
	found := false
	for _, e := range events {
		if e.Kind == "turn.end" && e.Status == "completed" {
			found = true
		}
	}
	if !found {
		t.Fatal("selected model/effort/permission did not pass actual native wire fixture")
	}
	current, _ := consumer.Get(s.ID)
	if current.Options != selected {
		t.Fatal("session did not retain the latest selection")
	}
	if _, err = consumer.Prompt(s.ID, "inherit-options", "require-options"); err != nil {
		t.Fatal(err)
	}
	waitState(t, consumer, s.ID, "awaiting-input")
	events, _, _ = consumer.Replay(s.ID, current.LastSeq)
	for _, event := range events {
		if event.Kind == "input.request" {
			request = *event.Request
		}
	}
	if err = consumer.Answer(s.ID, request.ID, Answer{OptionID: request.Options[0].ID}); err != nil {
		t.Fatal(err)
	}
	waitState(t, consumer, s.ID, "ready")
	fresh, err := consumer.Create(context.Background(), "new-defaults", scope("codex"))
	if err != nil {
		t.Fatal(err)
	}
	if fresh.Options != changed {
		t.Fatalf("new session ignored shared settings: %+v", fresh.Options)
	}
	recordBytes, err := os.ReadFile(filepath.Join(consumer.root, s.ID+".jsonl"))
	if err != nil {
		t.Fatal(err)
	}
	var record sessionRecord
	line, _, _ := strings.Cut(string(recordBytes), "\n")
	if json.Unmarshal([]byte(line), &record) != nil || record.ProfileSnapshot == nil || record.ProfileSnapshot.Defaults != defaults {
		t.Fatal("private journal snapshot missing")
	}
	apiBytes, _ := json.Marshal(current)
	if strings.Contains(string(apiBytes), "profileSnapshot") || strings.Contains(string(apiBytes), testProfile(t, "codex").Executable) {
		t.Fatal("private executable snapshot entered API")
	}
}
func TestUnsupportedProviderOptionsAndSettingsSecurity(t *testing.T) {
	path := filepath.Join(t.TempDir(), "native-agents.json")
	b := settingsBroker(t, path)
	initial, _ := b.Settings()
	update := settingsFixtureUpdate(t, initial.Revision, NativeOptions{Model: "invented"})
	if _, err := b.UpdateSettings(context.Background(), update); err == nil {
		t.Fatal("unadvertised model accepted")
	}
	update.Provider.Provider = "cursor"
	update.Provider.Defaults = NativeOptions{Permission: "full-access"}
	if _, err := b.UpdateSettings(context.Background(), update); err == nil {
		t.Fatal("Codex permissions leaked into ACP")
	}
	mux := http.NewServeMux()
	if err := RegisterRoutes(mux, b, "/api/native-agents"); err != nil {
		t.Fatal(err)
	}
	for _, body := range []string{`{"revision":"x","provider":{"id":"codex","provider":"codex","enabled":true,"defaults":{},"token":"bad"}}`, `{"id":"codex","args":["secret"]}`} {
		path := "/api/native-agents/settings"
		if strings.Contains(body, "args") {
			path += "/refresh"
		}
		request := httptest.NewRequest(http.MethodPost, "http://127.0.0.1"+path, strings.NewReader(body))
		request.RemoteAddr = "127.0.0.1:12345"
		request.Header.Set(ClientHeader, "1")
		request.Header.Set("Content-Type", "application/json")
		response := httptest.NewRecorder()
		mux.ServeHTTP(response, request)
		if response.Code != http.StatusBadRequest {
			t.Fatalf("unknown settings field status %d: %s", response.Code, response.Body.String())
		}
	}
}

func TestRestartKeepsPrivateProfileSnapshotAndReplaySkipsDiscovery(t *testing.T) {
	path := filepath.Join(t.TempDir(), "native-agents.json")
	b := settingsBroker(t, path)
	initial, _ := b.Settings()
	defaults := NativeOptions{Model: "fixture-native", Effort: "medium", Permission: "approval-required"}
	saved, err := b.UpdateSettings(context.Background(), settingsFixtureUpdate(t, initial.Revision, defaults))
	if err != nil {
		t.Fatal(err)
	}
	s, err := b.Create(context.Background(), "snapshot-restart", scope("codex"))
	if err != nil {
		t.Fatal(err)
	}
	waitState(t, b, s.ID, "ready")
	selected := NativeOptions{Permission: "full-access"}
	retained := mergeOptions(defaults, selected)
	turn, err := b.PromptWithOptions(s.ID, "replay-without-probe", "fixture", selected)
	if err != nil {
		t.Fatal(err)
	}
	_ = waitRequest(t, b, s.ID)
	b.settings.mu.Lock()
	b.settings.inventory = map[string]ProviderSetting{}
	b.settings.mu.Unlock()
	if replay, err := b.PromptWithOptions(s.ID, "replay-without-probe", "fixture", selected); err != nil || replay != turn {
		t.Fatalf("prompt replay: %s %v", replay, err)
	}
	if _, err = b.Create(context.Background(), "snapshot-restart", scope("codex")); err != nil {
		t.Fatal(err)
	}
	if _, err = b.PromptWithOptions(s.ID, "", "fixture", defaults); err == nil {
		t.Fatal("invalid key accepted")
	}
	b.settings.mu.Lock()
	count := len(b.settings.inventory)
	b.settings.mu.Unlock()
	if count != 0 {
		t.Fatal("invalid or idempotent request launched metadata inspection")
	}
	changed := NativeOptions{Model: "fixture-luna", Effort: "low", Permission: "full-access"}
	if _, err = b.UpdateSettings(context.Background(), settingsFixtureUpdate(t, saved.Revision, changed)); err != nil {
		t.Fatal(err)
	}
	b.Close()
	profiles, err := LoadConfig(path)
	if err != nil {
		t.Fatal(err)
	}
	restored, err := NewBroker(b.root, profiles, b.prepare, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer restored.Close()
	if err = ConfigureBroker(restored, BrokerOptions{SettingsFile: path}); err != nil {
		t.Fatal(err)
	}
	info, err := restored.Get(s.ID)
	if err != nil || info.Options != retained || restored.sessions[s.ID].profile.Defaults != retained {
		t.Fatalf("restart changed retained selection: %+v %v", info, err)
	}
	if restored.sessions[s.ID].profile.Executable == "" {
		t.Fatal("private native binary snapshot lost")
	}
}
