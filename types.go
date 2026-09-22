// Package nativeagent is a client of native agents, not a model gateway or agent loop.
package nativeagent

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

const Schema = "xgc.native-agent/v1"
const MaxFrame = 4 << 20
const MaxText = 256 << 10

var ErrConflict = errors.New("request identity conflict")
var ErrUnavailable = errors.New("native agent unavailable")
var ErrNotFound = errors.New("native session not found")
var ErrStale = errors.New("request is no longer pending")
var safeID = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9._-]{0,95}$`)
var digest = regexp.MustCompile(`^[a-f0-9]{64}$`)

// Profiles are operator-owned. Settings may select a local executable; its real
// path, version and digest are resolved by the host. No API accepts credentials,
// argv, environment, shell commands or arbitrary provider endpoints.
type Profile struct {
	ID              string        `json:"id"`
	Provider        string        `json:"provider"`
	Executable      string        `json:"executable"`
	SHA256          string        `json:"sha256"`
	ReviewedVersion string        `json:"reviewedVersion"`
	BillingReviewed bool          `json:"billingReviewed"`
	Disabled        bool          `json:"disabled,omitempty"`
	Defaults        NativeOptions `json:"defaults,omitempty"`
}
type Config struct {
	SchemaVersion string    `json:"schemaVersion"`
	Profiles      []Profile `json:"profiles"`
}
type Provider struct {
	ID                  string `json:"id"`
	Provider            string `json:"provider"`
	Protocol            string `json:"protocol"`
	Available           bool   `json:"available"`
	Detail              string `json:"detail"`
	ReviewedVersion     string `json:"reviewedVersion"`
	InteractiveRequests bool   `json:"interactiveRequests"`
	ToolMode            string `json:"toolMode"`
}

func LoadConfig(path string) ([]Profile, error) {
	if path == "" {
		return nil, nil
	}
	f, err := os.Open(path)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, errors.New("cannot read native agent configuration")
	}
	defer f.Close()
	return decodeConfig(f)
}
func decodeConfig(r io.Reader) ([]Profile, error) {
	var err error
	var c Config
	d := json.NewDecoder(io.LimitReader(r, 64<<10))
	d.DisallowUnknownFields()
	if err = d.Decode(&c); err != nil {
		return nil, errors.New("invalid native agent configuration")
	}
	if err = d.Decode(new(any)); err != io.EOF {
		return nil, errors.New("trailing native agent configuration")
	}
	if c.SchemaVersion != Schema || len(c.Profiles) > 16 {
		return nil, errors.New("unsupported native agent configuration")
	}
	seen := map[string]bool{}
	for _, p := range c.Profiles {
		if !safeID.MatchString(p.ID) || seen[p.ID] || (!p.Disabled && (!filepath.IsAbs(p.Executable) || !digest.MatchString(p.SHA256) || strings.TrimSpace(p.ReviewedVersion) == "" || !p.BillingReviewed)) || len(p.ReviewedVersion) > 256 {
			return nil, errors.New("profiles require distinct IDs, an absolute binary, digest, reviewed version and billing review")
		}
		if _, err = commandArgs(p.Provider); err != nil {
			return nil, err
		}
		seen[p.ID] = true
	}
	return c.Profiles, nil
}
func commandArgs(provider string) ([]string, error) {
	switch provider {
	case "codex":
		return []string{"-c", `model_provider="openai"`, "app-server"}, nil
	case "cursor":
		return []string{"acp"}, nil
	case "opencode":
		return []string{"acp"}, nil
	case "grok":
		return []string{"--no-auto-update", "agent", "stdio"}, nil
	case "claude":
		return []string{"--print", "--verbose", "--output-format", "stream-json", "--include-partial-messages", "--tools", "Read,Glob,Grep", "--allowedTools", "Read,Glob,Grep"}, nil
	default:
		return nil, errors.New("unsupported native provider")
	}
}
func checkExecutable(p Profile) error {
	if !supportedNativeHost {
		return ErrUnavailable
	}
	info, err := os.Lstat(p.Executable)
	if err != nil || !info.Mode().IsRegular() || info.Mode()&0111 == 0 {
		return ErrUnavailable
	}
	f, err := os.Open(p.Executable)
	if err != nil {
		return ErrUnavailable
	}
	defer f.Close()
	h := sha256.New()
	if _, err = io.Copy(h, f); err != nil || hex.EncodeToString(h.Sum(nil)) != p.SHA256 {
		return errors.New("native executable digest mismatch")
	}
	return nil
}

// Native login files remain owned/read by the CLI. In particular, the Go
// service's API keys, tokens, endpoint overrides and service credentials are
// never inherited. This does not inspect or certify the CLI's own billing config.
func NativeEnvironment(source []string) []string {
	allowed := map[string]bool{"PATH": true, "HOME": true, "USER": true, "LOGNAME": true, "SHELL": true, "LANG": true, "LC_ALL": true, "TMPDIR": true, "SYSTEMROOT": true, "WINDIR": true, "USERPROFILE": true, "LOCALAPPDATA": true, "APPDATA": true}
	result := []string{}
	for _, v := range source {
		k, _, ok := strings.Cut(v, "=")
		if ok && allowed[k] {
			result = append(result, v)
		}
	}
	return result
}

type Option struct {
	ID          string `json:"id"`
	Label       string `json:"label"`
	Kind        string `json:"kind"`
	Description string `json:"description,omitempty"`
}
type Question struct {
	ID       string   `json:"id"`
	Text     string   `json:"text"`
	Options  []Option `json:"options"`
	Multiple bool     `json:"multiple"`
	FreeText bool     `json:"freeText"`
	Header   string   `json:"header,omitempty"`
}
type Request struct {
	ID             string          `json:"id"`
	Kind           string          `json:"kind"`
	Title          string          `json:"title"`
	Text           string          `json:"text,omitempty"`
	Options        []Option        `json:"options"`
	Questions      []Question      `json:"questions"`
	SourceMethod   string          `json:"sourceMethod,omitempty"`
	NativeThreadID string          `json:"nativeThreadId,omitempty"`
	NativeTurnID   string          `json:"nativeTurnId,omitempty"`
	NativeItemID   string          `json:"nativeItemId,omitempty"`
	CreatedAt      string          `json:"createdAt,omitempty"`
	Details        *RequestDetails `json:"details,omitempty"`
}
type RequestDetails struct {
	ToolName  string          `json:"toolName,omitempty"`
	Target    string          `json:"target,omitempty"`
	Preview   string          `json:"preview,omitempty"`
	Truncated bool            `json:"truncated,omitempty"`
	Command   string          `json:"command,omitempty"`
	Cwd       string          `json:"cwd,omitempty"`
	Reason    string          `json:"reason,omitempty"`
	GrantRoot string          `json:"grantRoot,omitempty"`
	Network   *RequestNetwork `json:"network,omitempty"`
}
type RequestNetwork struct {
	Protocol string `json:"protocol"`
	Host     string `json:"host"`
}
type Answer struct {
	OptionID string              `json:"optionId,omitempty"`
	Answers  map[string][]string `json:"answers,omitempty"`
	Cancel   bool                `json:"cancel,omitempty"`
}

func (r Request) ValidateAnswer(a Answer) error {
	if a.Cancel {
		if a.OptionID != "" || len(a.Answers) > 0 {
			return errors.New("cancel cannot contain answers")
		}
		return nil
	}
	if len(r.Questions) > 0 {
		if a.OptionID != "" || len(a.Answers) != len(r.Questions) {
			return errors.New("answer every requested question")
		}
		for _, q := range r.Questions {
			values, ok := a.Answers[q.ID]
			if !ok || len(values) == 0 || len(values) > 32 || (!q.Multiple && len(values) != 1) {
				return errors.New("invalid question answer")
			}
			seen := map[string]bool{}
			for _, v := range values {
				if v == "" || len(v) > 8192 || seen[v] {
					return errors.New("invalid answer value")
				}
				seen[v] = true
				if !q.FreeText {
					found := false
					for _, o := range q.Options {
						if o.ID == v {
							found = true
						}
					}
					if !found {
						return errors.New("answer is not an offered option")
					}
				}
			}
		}
		return nil
	}
	if len(a.Answers) > 0 {
		return errors.New("unexpected question answers")
	}
	for _, o := range r.Options {
		if o.ID == a.OptionID {
			return nil
		}
	}
	return errors.New("select an offered decision")
}

// Event is a presentation record, never a product judgement or trusted model receipt.
// Native method names identify origin; credentials and unfiltered RPC frames are not stored.
type Event struct {
	Queue           *PromptQueue     `json:"queue,omitempty"`
	SchemaVersion   string           `json:"schemaVersion"`
	SessionID       string           `json:"sessionId"`
	Seq             uint64           `json:"seq"`
	Provider        string           `json:"provider"`
	TurnID          string           `json:"turnId,omitempty"`
	ItemID          string           `json:"itemId,omitempty"`
	Kind            string           `json:"kind"`
	Role            string           `json:"role,omitempty"`
	Text            string           `json:"text,omitempty"`
	Title           string           `json:"title,omitempty"`
	Status          string           `json:"status,omitempty"`
	SourceMethod    string           `json:"sourceMethod,omitempty"`
	NativeSessionID string           `json:"nativeSessionId,omitempty"`
	Request         *Request         `json:"request,omitempty"`
	CreatedAt       string           `json:"createdAt,omitempty"`
	NativeThreadID  string           `json:"nativeThreadId,omitempty"`
	NativeTurnID    string           `json:"nativeTurnId,omitempty"`
	Details         map[string]any   `json:"details,omitempty"`
	RuntimeID       string           `json:"runtimeId,omitempty"`
	Metadata        *SessionMetadata `json:"metadata,omitempty"`
	Decision        *DecisionReceipt `json:"decision,omitempty"`
}

// ContextRef identifies the product-owned resource; the broker never interprets it.
type ContextRef struct {
	Kind string `json:"kind"`
	ID   string `json:"id"`
}

// WorkspaceRef identifies a reviewed workspace snapshot. The product Prepare
// callback resolves and validates this reference; it is never a client path.
type WorkspaceRef struct {
	ID       string `json:"id"`
	Revision string `json:"revision"`
}
type Create struct {
	ProfileID             string        `json:"profileId"`
	Context               ContextRef    `json:"context"`
	Workspace             WorkspaceRef  `json:"workspace"`
	NativeAccessConfirmed bool          `json:"nativeAccessConfirmed"`
	Options               NativeOptions `json:"options,omitempty"`
}

func (c Create) Validate() error {
	if !safeID.MatchString(c.ProfileID) || !safeID.MatchString(c.Context.Kind) || !safeID.MatchString(c.Context.ID) || !safeID.MatchString(c.Workspace.ID) || c.Workspace.Revision == "" || len(c.Workspace.Revision) > 256 || strings.ContainsAny(c.Workspace.Revision, "\x00\r\n") || !c.NativeAccessConfirmed {
		return errors.New("a profile, typed context, reviewed workspace revision and native-access consent are required")
	}
	return nil
}

type Session struct {
	SchemaVersion    string        `json:"schemaVersion"`
	ID               string        `json:"id"`
	Scope            Create        `json:"scope"`
	Provider         string        `json:"provider"`
	State            string        `json:"state"`
	NativeSessionID  string        `json:"nativeSessionId,omitempty"`
	CreatedAt        string        `json:"createdAt"`
	LastSeq          uint64        `json:"lastSeq"`
	Options          NativeOptions `json:"options,omitempty"`
	Title            string        `json:"title"`
	Archived         bool          `json:"archived"`
	MetadataRevision uint64        `json:"metadataRevision"`
	RuntimeID        string        `json:"runtimeId,omitempty"`
	// No cwd or secrets in API read models.
}
type Sink func(Event) error
type Ask func(context.Context, Request) (Answer, error)
type Driver interface {
	Open(context.Context, string, string) error
	Prompt(context.Context, string, string) error
	Cancel(context.Context) error
	Close() error
}
type Factory func(Profile, Sink, Ask) (Driver, error)
type Prepare func(context.Context, Create, string, bool) (string, error)

func obj(v any) map[string]any               { m, _ := v.(map[string]any); return m }
func arr(v any) []any                        { a, _ := v.([]any); return a }
func str(v any) string                       { s, _ := v.(string); return s }
func text(m map[string]any, k string) string { return str(m[k]) }
func protocolError(code int) error {
	return fmt.Errorf("native protocol request failed (code %d); inspect the native client, not service credentials", code)
}
