package nativeagent

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

const NativeSettingsTimeout = 60 * time.Second

type NamedValue struct {
	ID    string `json:"id"`
	Label string `json:"label"`
}
type Model struct {
	ID            string       `json:"id"`
	Label         string       `json:"label"`
	Efforts       []NamedValue `json:"efforts"`
	DefaultEffort string       `json:"defaultEffort,omitempty"`
}
type Permission struct {
	ID          string `json:"id"`
	Label       string `json:"label"`
	Description string `json:"description"`
}
type LoginStatus struct {
	Status string `json:"status"`
	Detail string `json:"detail"`
}
type ProviderSetting struct {
	ID          string        `json:"id"`
	Provider    string        `json:"provider"`
	Enabled     bool          `json:"enabled"`
	BinaryPath  string        `json:"binaryPath"`
	Available   bool          `json:"available"`
	Version     string        `json:"version"`
	Login       LoginStatus   `json:"login"`
	Detail      string        `json:"detail"`
	Defaults    NativeOptions `json:"defaults"`
	Models      []Model       `json:"models"`
	Permissions []Permission  `json:"permissions"`
}
type Settings struct {
	Revision  string            `json:"revision"`
	Providers []ProviderSetting `json:"providers"`
}
type ProviderUpdate struct {
	ID         string        `json:"id"`
	Provider   string        `json:"provider"`
	Enabled    bool          `json:"enabled"`
	BinaryPath string        `json:"binaryPath,omitempty"`
	Defaults   NativeOptions `json:"defaults"`
}
type SettingsUpdate struct {
	Revision string         `json:"revision"`
	Provider ProviderUpdate `json:"provider"`
}
type BrokerOptions struct{ SettingsFile string }
type settingsState struct {
	file      string
	initial   []Profile
	mu        sync.Mutex
	inventory map[string]ProviderSetting
}

// ConfigureBroker enables one host-owned settings file shared by product brokers.
// Existing NewBroker callers remain valid and retain immutable supplied profiles.
func ConfigureBroker(b *Broker, options BrokerOptions) error {
	if !filepath.IsAbs(options.SettingsFile) {
		return errors.New("native settings require an absolute shared configuration path")
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.settings != nil {
		return errors.New("native settings are already configured")
	}
	initial := []Profile{}
	for _, p := range b.profiles {
		initial = append(initial, p)
	}
	b.settings = &settingsState{file: options.SettingsFile, initial: initial, inventory: map[string]ProviderSetting{}}
	return nil
}

// DefaultSettingsPath is shared by every local product consumer.
func DefaultSettingsPath() (string, error) {
	root, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(root, "xgc", "native-agents.json"), nil
}
func lookupProvider(provider, command string) (string, error) {
	if command != "" {
		return exec.LookPath(command)
	}
	path, err := exec.LookPath(builtinCommand(provider))
	if err != nil && provider == "cursor" {
		return exec.LookPath("agent")
	}
	return path, err
}
func builtinCommand(provider string) string {
	switch provider {
	case "cursor":
		return "cursor-agent"
	default:
		return provider
	}
}

var providerKinds = []string{"codex", "claude", "cursor", "grok", "opencode"}

func emptySetting(p Profile) ProviderSetting {
	path := p.Executable
	if path == "" {
		if found, err := lookupProvider(p.Provider, ""); err == nil {
			path, _ = filepath.EvalSymlinks(found)
		}
	}
	available := false
	if path != "" {
		if info, err := os.Stat(path); err == nil {
			available = info.Mode().IsRegular() && info.Mode()&0111 != 0
		}
	}
	detail := "Refresh to inspect the installed CLI and its native login."
	if !available {
		detail = "Native CLI not found. Install and log in using the provider's own client."
	}
	return ProviderSetting{ID: p.ID, Provider: p.Provider, Enabled: !p.Disabled, BinaryPath: path, Available: available, Version: p.ReviewedVersion, Login: LoginStatus{"unknown", "Native login has not been checked."}, Detail: detail, Defaults: p.Defaults, Models: []Model{}, Permissions: []Permission{}}
}
func (s *settingsState) read() ([]Profile, string, error) {
	info, err := os.Lstat(s.file)
	if os.IsNotExist(err) {
		data, _ := json.Marshal(Config{SchemaVersion: Schema, Profiles: s.initial})
		return append([]Profile{}, s.initial...), hash(string(data)), nil
	}
	if err != nil || !info.Mode().IsRegular() || info.Size() > 64<<10 {
		return nil, "", errors.New("invalid native settings file")
	}
	data, err := os.ReadFile(s.file)
	if err != nil {
		return nil, "", err
	}
	profiles, err := decodeConfig(strings.NewReader(string(data)))
	return profiles, hash(string(data)), err
}
func (b *Broker) reloadSettingsProfiles() error {
	if b.settings == nil {
		return nil
	}
	s := b.settings
	s.mu.Lock()
	profiles, _, err := s.read()
	s.mu.Unlock()
	if err != nil {
		return err
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	b.profiles = map[string]Profile{}
	for _, p := range profiles {
		b.profiles[p.ID] = p
	}
	return nil
}
func (b *Broker) Settings() (Settings, error) {
	if b.settings == nil {
		return Settings{}, ErrUnavailable
	}
	if err := b.reloadSettingsProfiles(); err != nil {
		return Settings{}, err
	}
	s := b.settings
	s.mu.Lock()
	defer s.mu.Unlock()
	profiles, revision, err := s.read()
	if err != nil {
		return Settings{}, err
	}
	seen := map[string]bool{}
	result := Settings{Revision: revision, Providers: []ProviderSetting{}}
	for _, p := range profiles {
		seen[p.Provider] = true
		v := emptySetting(p)
		if cached, ok := s.inventory[p.Executable+":"+p.SHA256]; ok {
			v = cached
			v.ID = p.ID
			v.Enabled = !p.Disabled
			v.Defaults = mergeOptions(cached.Defaults, p.Defaults)
		}
		result.Providers = append(result.Providers, v)
	}
	for _, kind := range providerKinds {
		if !seen[kind] {
			v := emptySetting(Profile{ID: kind, Provider: kind, Disabled: true})
			if cached, ok := s.inventory["discovered:"+kind]; ok && cached.BinaryPath == v.BinaryPath {
				v = cached
				v.ID = kind
				v.Enabled = false
			}
			result.Providers = append(result.Providers, v)
		}
	}
	sort.SliceStable(result.Providers, func(i, j int) bool { return result.Providers[i].ID < result.Providers[j].ID })
	return result, nil
}
func resolveProfile(input ProviderUpdate) (Profile, error) {
	if !safeID.MatchString(input.ID) {
		return Profile{}, errors.New("invalid native provider ID")
	}
	if _, err := commandArgs(input.Provider); err != nil {
		return Profile{}, err
	}
	p := Profile{ID: input.ID, Provider: input.Provider, Disabled: !input.Enabled, Defaults: input.Defaults}
	command := strings.TrimSpace(input.BinaryPath)

	if strings.ContainsAny(command, "\x00\r\n") {
		return p, errors.New("invalid native binary path")
	}
	path, err := lookupProvider(input.Provider, command)
	if err != nil {
		if p.Disabled && input.Defaults == (NativeOptions{}) {
			return p, nil
		}
		return p, errors.New("native executable not found")
	}
	path, err = filepath.EvalSymlinks(path)
	if err != nil {
		return p, errors.New("native executable cannot be resolved")
	}
	path, err = filepath.Abs(path)
	if err != nil {
		return p, err
	}
	f, err := os.Open(path)
	if err != nil {
		return p, err
	}
	defer f.Close()
	info, err := f.Stat()
	if err != nil || !info.Mode().IsRegular() || info.Mode()&0111 == 0 {
		return p, errors.New("native executable must be a regular executable file")
	}
	h := sha256.New()
	if _, err = io.Copy(h, f); err != nil {
		return p, err
	}
	p.Executable = path
	p.SHA256 = hex.EncodeToString(h.Sum(nil))
	p.BillingReviewed = true
	p.ReviewedVersion = "Unverified native CLI"
	return p, nil
}
func (b *Broker) RefreshSettings(ctx context.Context, id string) (Settings, error) {
	ctx, cancel := context.WithTimeout(ctx, NativeSettingsTimeout)
	defer cancel()
	current, err := b.Settings()
	if err != nil {
		return Settings{}, err
	}
	var update ProviderUpdate
	found := false
	for _, p := range current.Providers {
		if p.ID == id {
			update = ProviderUpdate{ID: id, Provider: p.Provider, Enabled: p.Enabled, BinaryPath: p.BinaryPath, Defaults: p.Defaults}
			found = true
			break
		}
	}
	if !found {
		return Settings{}, ErrNotFound
	}
	profile, err := resolveProfile(update)
	if err != nil {
		return Settings{}, err
	}
	inventory := inspectProvider(ctx, profile)
	s := b.settings
	s.mu.Lock()
	s.inventory[profile.Executable+":"+profile.SHA256] = inventory
	// Discovery-only entries have no profile yet; preserve their bounded probe result.
	s.inventory["discovered:"+id] = inventory
	s.mu.Unlock()
	return b.Settings()
}
func (b *Broker) UpdateSettings(ctx context.Context, input SettingsUpdate) (Settings, error) {
	ctx, cancel := context.WithTimeout(ctx, NativeSettingsTimeout)
	defer cancel()
	s := b.settings
	if s == nil {
		return Settings{}, ErrUnavailable
	}
	if input.Revision == "" {
		return Settings{}, ErrConflict
	}
	// Reject stale revisions before invoking a newly selected executable.
	s.mu.Lock()
	prior, revision, err := s.read()
	s.mu.Unlock()
	if err != nil {
		return Settings{}, err
	}
	if revision != input.Revision {
		return Settings{}, ErrConflict
	}
	for _, p := range prior {
		if p.ID == input.Provider.ID && p.Provider != input.Provider.Provider {
			return Settings{}, errors.New("provider identity cannot change")
		}
	}
	profile, err := resolveProfile(input.Provider)
	var preserved *Profile
	if !input.Provider.Enabled {
		for _, p := range prior {
			if p.ID == input.Provider.ID && p.Provider == input.Provider.Provider && (input.Provider.BinaryPath == "" || input.Provider.BinaryPath == p.Executable) && input.Provider.Defaults == p.Defaults {
				copy := p
				copy.Disabled = true
				preserved = &copy
				profile = copy
				err = nil
				break
			}
		}
	}
	if err != nil {
		return Settings{}, err
	}
	inventory := emptySetting(profile)
	if preserved == nil {
		s.mu.Lock()
		cached, exists := s.inventory[profile.Executable+":"+profile.SHA256]
		s.mu.Unlock()
		if exists && cached.Version != "" {
			inventory = cached
		} else {
			inventory = inspectProvider(ctx, profile)
		}
	}
	if err := ctx.Err(); err != nil {
		return Settings{}, err
	}
	if profile.Executable != "" && preserved == nil {
		profile.ReviewedVersion = inventory.Version
		if profile.ReviewedVersion == "" {
			return Settings{}, errors.New("native CLI version could not be verified")
		}
	}
	if err = validateOptions(input.Provider.Defaults, inventory); err != nil && preserved == nil {
		return Settings{}, err
	}
	if err = os.MkdirAll(filepath.Dir(s.file), 0700); err != nil {
		return Settings{}, err
	}
	lockPath := s.file + ".lock"
	if info, e := os.Lstat(lockPath); e == nil && !info.Mode().IsRegular() {
		return Settings{}, errors.New("invalid native settings lock")
	}
	lock, err := os.OpenFile(lockPath, os.O_CREATE|os.O_RDWR, 0600)
	if err != nil {
		return Settings{}, err
	}
	defer lock.Close()
	if err = lockSettings(ctx, lock); err != nil {
		return Settings{}, err
	}
	defer unlockSettings(lock)
	s.mu.Lock()
	profiles, revision, err := s.read()
	if err != nil {
		s.mu.Unlock()
		return Settings{}, err
	}
	if revision != input.Revision {
		s.mu.Unlock()
		return Settings{}, ErrConflict
	}
	found := false
	for i, p := range profiles {
		if p.ID == profile.ID {
			if p.Provider != profile.Provider {
				s.mu.Unlock()
				return Settings{}, errors.New("provider identity cannot change")
			}
			profiles[i] = profile
			found = true
		}
	}
	if !found {
		if len(profiles) >= 16 {
			s.mu.Unlock()
			return Settings{}, errors.New("native provider limit reached")
		}
		profiles = append(profiles, profile)
	}
	data, _ := json.MarshalIndent(Config{SchemaVersion: Schema, Profiles: profiles}, "", "  ")
	data = append(data, '\n')
	temp, err := os.CreateTemp(filepath.Dir(s.file), ".native-settings-*")
	if err == nil {
		name := temp.Name()
		defer os.Remove(name)
		err = temp.Chmod(0600)
		if err == nil {
			_, err = temp.Write(data)
		}
		if err == nil {
			err = temp.Sync()
		}
		closeErr := temp.Close()
		if err == nil {
			err = closeErr
		}
		if err == nil {
			err = os.Rename(name, s.file)
			if err == nil {
				if dir, e := os.Open(filepath.Dir(s.file)); e == nil {
					err = dir.Sync()
					_ = dir.Close()
				} else {
					err = e
				}
			}
		}
	}
	if err == nil {
		s.inventory[profile.Executable+":"+profile.SHA256] = inventory
	}
	s.mu.Unlock()
	if err != nil {
		return Settings{}, err
	}
	return b.Settings()
}
func validateOptions(o NativeOptions, p ProviderSetting) error {
	if o.Model == "" && o.Effort != "" {
		return errors.New("select a native model before its reasoning effort")
	}
	if o.Model != "" {
		found := false
		for _, m := range p.Models {
			if m.ID == o.Model {
				found = true
				if o.Effort != "" {
					valid := false
					for _, effort := range m.Efforts {
						valid = valid || effort.ID == o.Effort
					}
					if !valid {
						return errors.New("reasoning effort is not advertised for this model")
					}
				}
				break
			}
		}
		if !found {
			return errors.New("model is not advertised by this native provider; refresh provider capabilities")
		}
	}
	if o.Permission != "" {
		found := false
		for _, v := range p.Permissions {
			found = found || v.ID == o.Permission
		}
		if !found {
			return errors.New("permission mode is not supported by this native integration")
		}
	}
	return nil
}
func (b *Broker) warmSelection(ctx context.Context, p Profile) {
	ctx, cancel := context.WithTimeout(ctx, NativeSettingsTimeout)
	defer cancel()
	if b.settings == nil || p.Executable == "" {
		return
	}
	s := b.settings
	s.mu.Lock()
	_, ok := s.inventory[p.Executable+":"+p.SHA256]
	s.mu.Unlock()
	if ok {
		return
	}
	inventory := inspectProvider(ctx, p)
	s.mu.Lock()
	s.inventory[p.Executable+":"+p.SHA256] = inventory
	s.mu.Unlock()
}
func (b *Broker) selection(profile Profile, override NativeOptions) (NativeOptions, error) {
	selected := mergeOptions(profile.Defaults, override)
	if b.settings == nil {
		if selected == (NativeOptions{}) {
			return selected, nil
		}
		return selected, errors.New("native selections require configured provider capabilities")
	}
	b.settings.mu.Lock()
	inventory, ok := b.settings.inventory[profile.Executable+":"+profile.SHA256]
	b.settings.mu.Unlock()
	if !ok {
		if selected == (NativeOptions{}) {
			return selected, nil
		}
		return selected, errors.New("refresh native provider capabilities before selecting model or permissions")
	}
	// Explicit per-session defaults win over native defaults. Defaults are resolved
	// once into the private session snapshot, not reread on each existing turn.
	selected = mergeOptions(mergeOptions(inventory.Defaults, profile.Defaults), override)
	if selected.Model != "" && selected.Effort == "" {
		for _, m := range inventory.Models {
			if m.ID == selected.Model {
				selected.Effort = m.DefaultEffort
				break
			}
		}
	}
	return selected, validateOptions(selected, inventory)
}

// CLI output is bounded and only recognized version/status/model fields are surfaced.
func cliOutput(ctx context.Context, p Profile, args ...string) ([]byte, error) {
	ctx, cancel := context.WithTimeout(ctx, 8*time.Second)
	defer cancel()
	if p.Provider == "grok" && (len(args) == 0 || args[0] != "--no-auto-update") {
		args = append([]string{"--no-auto-update"}, args...)
	}
	cmd := exec.CommandContext(ctx, p.Executable, args...)
	cmd.Env = NativeEnvironment(os.Environ())
	isolateProcess(cmd)
	cmd.WaitDelay = 2 * time.Second
	cmd.Cancel = func() error { return terminateProcess(cmd, true) }
	cmd.Stderr = io.Discard
	out := &limitedOutput{limit: 1 << 20}
	cmd.Stdout = out
	err := cmd.Run()
	return out.data, err
}

type limitedOutput struct {
	data  []byte
	limit int
}

func (b *limitedOutput) Write(p []byte) (int, error) {
	if len(b.data)+len(p) > b.limit {
		return 0, errors.New("native inspection output exceeded limit")
	}
	b.data = append(b.data, p...)
	return len(p), nil
}
func inspectProvider(ctx context.Context, p Profile) ProviderSetting {
	ctx, cancel := context.WithTimeout(ctx, NativeSettingsTimeout)
	defer cancel()
	result := emptySetting(p)
	if p.Executable == "" {
		return result
	}
	if err := checkExecutable(p); err != nil {
		result.Available = false
		result.Detail = "Native executable changed; review its binary path again."
		return result
	}
	data, err := cliOutput(ctx, p, "--version")
	if err != nil {
		result.Version = ""
		result.Detail = "Native CLI version check failed."
		return result
	}
	version := strings.TrimSpace(string(data))
	if len(version) > 256 || strings.ContainsAny(version, "\x00\r\n") {
		result.Version = ""
		result.Detail = "Native CLI returned an unsupported version response."
		return result
	}
	result.Version = version
	if p.Provider == "codex" {
		inspectCodex(ctx, p, &result)
	} else if p.Provider != "claude" {
		inspectLocalLogin(ctx, p, &result)
		inspectACP(ctx, p, &result)
		if p.Provider == "opencode" {
			inspectOpenCodeVariants(ctx, p, &result)
		}
		if p.Provider == "cursor" && len(result.Permissions) > 0 {
			modes := []Permission{{"approval-required", "Ask for approval", "Native Cursor agent mode with approval prompts."}, {"auto", "Auto review", "Use native Cursor automatic review."}, {"full-access", "Full access", "Use native Cursor force approval mode."}}
			for _, v := range result.Permissions {
				if v.ID != "agent" {
					modes = append(modes, v)
				}
			}
			result.Permissions = modes
			if result.Defaults.Permission == "agent" {
				result.Defaults.Permission = "approval-required"
			}
		}
		if p.Provider == "grok" {
			if help, e := cliOutput(ctx, p, "--help"); e == nil && strings.Contains(string(help), "--permission-mode") {
				result.Permissions = grokPermissions()
				result.Defaults.Permission = "approval-required"
			}
		}
	} else {
		inspectClaude(ctx, p, &result)
	}
	return result
}
func inspectCodex(ctx context.Context, p Profile, result *ProviderSetting) {
	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	args, _ := commandArgs("codex")
	child, err := startChild(p, args, os.TempDir())
	if err != nil {
		result.Detail = "Codex metadata inspection could not start."
		return
	}
	peer := newPeer(child.stdin, child.stdout, false, func(string, map[string]any) {}, func(context.Context, string, map[string]any) (any, error) { return nil, ErrUnavailable })
	go func() { <-peer.done; _ = child.Wait() }()
	defer func() { _ = peer.Close(); child.Stop() }()
	if _, err = peer.Call(ctx, "initialize", map[string]any{"clientInfo": map[string]any{"name": "xgc-native-settings", "version": "0.1.0"}, "capabilities": map[string]any{"experimentalApi": false}}); err != nil {
		result.Detail = "Codex metadata handshake failed."
		return
	}
	if err = peer.Notify("initialized", map[string]any{}); err != nil {
		return
	}
	account, err := peer.Call(ctx, "account/read", map[string]any{"refreshToken": false})
	if err == nil {
		if text(obj(account["account"]), "type") == "chatgpt" {
			result.Login = LoginStatus{"authenticated", "Using the native ChatGPT login."}
		} else {
			result.Login = LoginStatus{"unauthenticated", "Native ChatGPT login is required; no API-billing fallback is used."}
		}
	}
	models, err := peer.Call(ctx, "model/list", map[string]any{"limit": 100, "includeHidden": false})
	if err != nil {
		result.Detail = "Codex model catalog could not be read."
		return
	}
	for _, raw := range arr(models["data"]) {
		m := obj(raw)
		id := text(m, "model")
		if id == "" || len(id) > 256 {
			continue
		}
		model := Model{ID: id, Label: text(m, "displayName"), Efforts: []NamedValue{}, DefaultEffort: text(m, "defaultReasoningEffort")}
		if model.Label == "" {
			model.Label = id
		}
		for _, rawEffort := range arr(m["supportedReasoningEfforts"]) {
			v := obj(rawEffort)
			id := text(v, "reasoningEffort")
			if id != "" {
				model.Efforts = append(model.Efforts, NamedValue{id, id})
			}
		}
		result.Models = append(result.Models, model)
	}
	result.Permissions = []Permission{{"approval-required", "Read only", "Tools require approval; the sandbox is read only."}, {"auto-accept-edits", "Workspace writes", "Allow workspace edits; request approval for actions outside the sandbox."}, {"full-access", "Full access", "No sandbox or approval prompts. Only select for trusted tasks."}}
	result.Detail = "Models and reasoning options were read from the installed Codex app-server."
	// config/read returns a broad native document. Only these allowlisted defaults
	// are used; credentials, endpoints and arbitrary config never enter API/journal.
	if config, e := peer.Call(ctx, "config/read", map[string]any{"includeLayers": false}); e == nil {
		native := obj(config["config"])
		candidate := NativeOptions{Model: text(native, "model"), Effort: text(native, "model_reasoning_effort")}
		if validateOptions(candidate, *result) == nil {
			result.Defaults = mergeOptions(candidate, p.Defaults)
		}
	}
}
