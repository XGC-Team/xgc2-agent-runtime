package agentruntime

import (
	"bufio"
	"context"
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"
)

//go:embed upstream/t3/claude-model-manifest.json
var claudeManifestJSON []byte

func claudeManifest() map[string]any {
	var value map[string]any
	_ = json.Unmarshal(claudeManifestJSON, &value)
	return value
}

var versionPattern = regexp.MustCompile(`\b(\d+)\.(\d+)\.(\d+)\b`)

func versionAtLeast(actual, minimum string) bool {
	a, b := versionPattern.FindStringSubmatch(actual), versionPattern.FindStringSubmatch(minimum)
	if len(a) != 4 || len(b) != 4 {
		return false
	}
	for i := 1; i < 4; i++ {
		x, _ := strconv.Atoi(a[i])
		y, _ := strconv.Atoi(b[i])
		if x != y {
			return x > y
		}
	}
	return true
}
func claudeCatalog(version string) []Model {
	manifest := claudeManifest()
	profiles := obj(manifest["profiles"])
	models := []Model{}
	for _, raw := range arr(manifest["models"]) {
		entry := obj(raw)
		minimum := text(obj(obj(entry["adapter"])["claudeCode"]), "minVersion")
		if minimum != "" && !versionAtLeast(version, minimum) {
			continue
		}
		model := Model{ID: text(entry, "slug"), Label: text(entry, "name"), Efforts: []NamedValue{}}
		profile := obj(profiles[text(entry, "profile")])
		for _, rawDescriptor := range arr(obj(profile["capabilities"])["optionDescriptors"]) {
			descriptor := obj(rawDescriptor)
			if text(descriptor, "id") != "effort" {
				continue
			}
			for _, rawOption := range arr(descriptor["options"]) {
				option := obj(rawOption)
				id := text(option, "id")
				switch id {
				case "low", "medium", "high", "xhigh", "max":
					model.Efforts = append(model.Efforts, NamedValue{id, text(option, "label")})
					if isDefault, _ := option["isDefault"].(bool); isDefault {
						model.DefaultEffort = id
					}
				}
			}
		}
		models = append(models, model)
	}
	return models
}
func claudeEffort(model, effort string) string {
	manifest := claudeManifest()
	for _, raw := range arr(manifest["models"]) {
		entry := obj(raw)
		if text(entry, "slug") == model {
			profile := obj(obj(manifest["profiles"])[text(entry, "profile")])
			mapping := obj(obj(obj(profile["adapter"])["claudeCode"])["effortMap"])
			if value := text(mapping, effort); value != "" {
				return value
			}
		}
	}
	return effort
}
func inspectClaude(ctx context.Context, p Profile, result *ProviderSetting) {
	help, err := cliOutput(ctx, p, "--help")
	if err != nil {
		return
	}
	supported := string(help)
	if strings.Contains(supported, "--model") {
		result.Models = claudeCatalog(result.Version)
	}
	if !strings.Contains(supported, "--effort") {
		for i := range result.Models {
			result.Models[i].Efforts = []NamedValue{}
			result.Models[i].DefaultEffort = ""
		}
	}
	if strings.Contains(supported, "--permission-mode") && strings.Contains(supported, "--input-format") {
		result.Permissions = []Permission{{"approval-required", "Ask for approval", "Use permission prompts with the shared approval surface."}, {"auto-accept-edits", "Accept edits", "Claude accepts edits and asks before other restricted actions."}, {"full-access", "Full access", "Bypass permission prompts for this explicit selection."}, {"plan", "Plan", "Use the plan permission mode."}}
		result.Defaults.Permission = "approval-required"
	}
	if output, _ := cliOutput(ctx, p, "auth", "status", "--json"); len(output) > 0 {
		var value map[string]any
		if json.Unmarshal(output, &value) == nil {
			if logged, ok := value["loggedIn"].(bool); ok {
				if logged {
					result.Login = LoginStatus{"authenticated", "Using the existing Claude login."}
				} else {
					result.Login = LoginStatus{"unauthenticated", "Log in using the Claude client."}
				}
			}
		}
	}
	result.Detail = "Model capabilities use the fixed T3 Code Claude catalog, filtered by installed CLI version. CLI flags carry selections; no model fallback is configured."
	result.Defaults = mergeOptions(result.Defaults, p.Defaults)
}
func claudeOptionArgs(options AgentOptions) ([]string, error) {
	args := []string{"--print", "--verbose", "--output-format", "stream-json", "--include-partial-messages", "--input-format", "stream-json", "--permission-prompt-tool", "stdio"}
	if options.Model != "" {
		args = append(args, "--model="+options.Model)
	}
	if options.Effort != "" {
		args = append(args, "--effort="+claudeEffort(options.Model, options.Effort))
	}
	switch options.Permission {
	case "", "approval-required": // Native default changed its label; omit rather than send the obsolete "default" enum.
	case "auto-accept-edits":
		args = append(args, "--permission-mode=acceptEdits")
	case "full-access":
		args = append(args, "--permission-mode=bypassPermissions", "--allow-dangerously-skip-permissions")
	case "plan":
		args = append(args, "--permission-mode=plan")
	default:
		return nil, errors.New("unsupported Claude permission mode")
	}
	return args, nil
}

type claudeControl struct {
	ctx     context.Context
	input   io.WriteCloser
	ask     Ask
	native  string
	writeMu sync.Mutex
	mu      sync.Mutex
	pending map[string]context.CancelFunc
	wg      sync.WaitGroup
}

func (c *claudeControl) write(value any) error {
	c.writeMu.Lock()
	defer c.writeMu.Unlock()
	return json.NewEncoder(c.input).Encode(value)
}
func (c *claudeControl) cancel(id string) {
	c.mu.Lock()
	cancel := c.pending[id]
	c.mu.Unlock()
	if cancel != nil {
		cancel()
	}
}
func (c *claudeControl) close() {
	c.mu.Lock()
	for _, cancel := range c.pending {
		cancel()
	}
	c.mu.Unlock()
	c.wg.Wait()
}
func (c *claudeControl) request(frame map[string]any) {
	id := text(frame, "request_id")
	body := obj(frame["request"])
	if id == "" || len(id) > 512 {
		return
	}
	c.mu.Lock()
	if _, exists := c.pending[id]; exists || len(c.pending) >= 16 {
		c.mu.Unlock()
		return
	}
	native := c.native
	ctx, cancel := context.WithCancel(c.ctx)
	c.pending[id] = cancel
	c.wg.Add(1)
	c.mu.Unlock()
	go func() {
		defer c.wg.Done()
		defer cancel()
		defer func() { c.mu.Lock(); delete(c.pending, id); c.mu.Unlock() }()
		response := map[string]any{"subtype": "success", "request_id": id}
		if text(body, "subtype") != "can_use_tool" || c.ask == nil {
			response["subtype"] = "error"
			response["error"] = "Unsupported control request"
		} else {
			request := Request{Kind: "permission", Title: text(body, "title"), Text: text(body, "description"), Options: []Option{{ID: "allow", Label: "Allow this operation", Kind: "allow_once"}, {ID: "deny", Label: "Decline", Kind: "reject_once"}}, Questions: []Question{}, SourceMethod: "claude:can_use_tool", AgentThreadID: native, AgentItemID: text(body, "tool_use_id")}
			if request.Title == "" {
				request.Title = text(body, "tool_name")
			}
			input := obj(body["input"])
			request.Details = claudePermissionDetails(text(body, "tool_name"), input, body)
			if text(body, "tool_name") == "AskUserQuestion" {
				request.Kind, request.Title = "question", "Input needed"
				request.Options = []Option{}
				for _, raw := range arr(input["questions"]) {
					value := obj(raw)
					question := Question{ID: text(value, "question"), Text: text(value, "question"), Header: text(value, "header"), FreeText: true, Options: []Option{}}
					question.Multiple, _ = value["multiSelect"].(bool)
					for _, rawOption := range arr(value["options"]) {
						option := obj(rawOption)
						question.Options = append(question.Options, Option{ID: text(option, "label"), Label: text(option, "label"), Description: text(option, "description"), Kind: "answer"})
					}
					request.Questions = append(request.Questions, question)
				}
			}
			var answer Answer
			var err error
			if err = validateRequest(request); err == nil {
				answer, err = c.ask(ctx, request)
			}
			if err == nil {
				err = request.ValidateAnswer(answer)
			}
			decision := map[string]any{"behavior": "deny", "message": "The operator declined this operation."}
			if err == nil && !answer.Cancel && request.Kind == "question" {
				// Fixed T3/Claude native protocol indexes answers by full question
				// text and joins multi-select labels into its string answer value.
				answers := map[string]string{}
				for key, values := range answer.Answers {
					answers[key] = strings.Join(values, ", ")
				}
				decision = map[string]any{"behavior": "allow", "updatedInput": map[string]any{"questions": input["questions"], "answers": answers}}
			} else if err == nil && !answer.Cancel && answer.OptionID == "allow" {
				decision = map[string]any{"behavior": "allow", "updatedInput": input}
			}
			if answer.Cancel {
				decision["interrupt"] = true
			}
			response["response"] = decision
		}
		if ctx.Err() == nil {
			_ = c.write(map[string]any{"type": "control_response", "response": response})
		}
	}()
}
func (d *claudeDriver) PromptWithOptions(ctx context.Context, turn, prompt string, options AgentOptions) error {
	if options == (AgentOptions{}) {
		return d.Prompt(ctx, turn, prompt)
	}
	return d.promptWithControl(ctx, turn, prompt, options, false)
}

func (d *claudeDriver) promptWithControl(ctx context.Context, turn, prompt string, options AgentOptions, restrictedTools bool) error {
	args, err := claudeOptionArgs(options)
	if err != nil {
		return err
	}
	if restrictedTools {
		args = append(args, "--tools", "Read,Glob,Grep", "--allowedTools", "Read,Glob,Grep")
	}
	d.mu.Lock()
	native, cwd := d.nativeSession, d.cwd
	d.mu.Unlock()
	if native != "" {
		args = append(args, "--resume="+native)
	}
	args, environment, cleanup, err := claudeLocalMCPLaunch(ctx, args)
	if err != nil {
		return err
	}
	defer cleanup()
	child, err := startChild(d.profile, args, cwd, environment...)
	if err != nil {
		return err
	}
	d.mu.Lock()
	d.process = child
	d.cancelled = false
	d.mu.Unlock()
	stop := context.AfterFunc(ctx, func() { child.Stop() })
	defer stop()
	controlCtx, cancel := context.WithCancel(ctx)
	defer cancel()
	control := &claudeControl{ctx: controlCtx, input: child.stdin, ask: d.ask, native: native, pending: map[string]context.CancelFunc{}}
	defer control.close()
	if err = control.write(map[string]any{"type": "control_request", "request_id": "xgc-initialize", "request": map[string]any{"subtype": "initialize", "hooks": nil}}); err != nil {
		child.Stop()
		return err
	}
	handshake := time.AfterFunc(30*time.Second, func() { child.Stop() })
	defer handshake.Stop()
	decoder := claudeDecoder{turn: turn, native: native, sink: d.sink, blocks: map[int]claudeBlock{}}
	scanner := bufio.NewScanner(child.stdout)
	scanner.Buffer(make([]byte, 4096), MaxFrame)
	initialized := false
	for scanner.Scan() {
		var message map[string]any
		if json.Unmarshal(scanner.Bytes(), &message) != nil {
			err = errors.New("invalid Claude control frame")
			break
		}
		switch text(message, "type") {
		case "control_response":
			response := obj(message["response"])
			if text(response, "request_id") == "xgc-initialize" && !initialized {
				if text(response, "subtype") != "success" {
					err = errors.New("Claude initialization rejected")
					break
				}
				initialized = true
				handshake.Stop()
				err = control.write(map[string]any{"type": "user", "message": map[string]any{"role": "user", "content": prompt}, "parent_tool_use_id": nil, "session_id": native})
			}
		case "control_request":
			control.request(message)
		case "control_cancel_request":
			control.cancel(text(message, "request_id"))
		default:
			err = decoder.consume(message)
			control.mu.Lock()
			control.native = decoder.native
			control.mu.Unlock()
		}
		if err != nil {
			break
		}
		if decoder.terminal {
			cancel()
			control.close()
			_ = child.stdin.Close()
		}
	}
	if scanner.Err() != nil {
		err = errors.New("Claude frame exceeds transport limits")
	}
	cancel()
	control.close()
	if err != nil {
		go child.Stop()
	}
	waitErr := child.Wait()
	d.mu.Lock()
	d.process = nil
	cancelled := d.cancelled
	if decoder.native != "" {
		d.nativeSession = decoder.native
	}
	d.mu.Unlock()
	if cancelled {
		return d.sink(Event{Kind: "turn.end", TurnID: turn, Status: "cancelled", SourceMethod: "claude:process-exit"})
	}
	if ctx.Err() != nil {
		return ctx.Err()
	}
	if err != nil {
		return err
	}
	if !decoder.terminal {
		return errors.New("Claude transport ended without a terminal result")
	}
	if waitErr != nil && decoder.status == "completed" {
		return fmt.Errorf("Claude exited unsuccessfully after its result")
	}
	return d.sink(Event{Kind: "turn.end", TurnID: turn, Status: decoder.status, SourceMethod: "claude:result"})
}
