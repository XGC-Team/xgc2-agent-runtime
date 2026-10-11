package agentruntime

import (
	"context"
	"encoding/json"
	"errors"
)

// AgentOptions are selections from the provider's discovered capabilities.
// Empty values preserve the session snapshot; no model alias or fallback is invented.
type AgentOptions struct {
	Model      string `json:"model,omitempty"`
	Effort     string `json:"effort,omitempty"`
	Permission string `json:"permission,omitempty"`
}

func mergeOptions(base, override AgentOptions) AgentOptions {
	if override.Model != "" {
		if override.Model != base.Model {
			base.Effort = ""
		}
		base.Model = override.Model
	}
	if override.Effort != "" {
		base.Effort = override.Effort
	}
	if override.Permission != "" {
		base.Permission = override.Permission
	}
	return base
}

type optionDriver interface {
	PromptWithOptions(context.Context, string, string, AgentOptions) error
}

// Adapted from the T3 Code Codex adapter reviewed at 477282263a9115f911380b5aec7d4973267e4edc.
// The original contract and its successor hashes are recorded in upstream/t3/PROVENANCE.json.
// Keep explicit permission mappings; never default an unknown value to full access.
func codexPermissions(permission string) (approval, sandbox, turnSandbox string, err error) {
	switch permission {
	case "":
		return "untrusted", "workspace-write", "workspaceWrite", nil // legacy integration default
	case "approval-required":
		return "untrusted", "read-only", "readOnly", nil
	case "auto-accept-edits":
		return "on-request", "workspace-write", "workspaceWrite", nil
	case "full-access":
		return "never", "danger-full-access", "dangerFullAccess", nil
	default:
		return "", "", "", errors.New("unsupported permission")
	}
}
func codexThreadOptions(params map[string]any, o AgentOptions) error {
	a, s, _, err := codexPermissions(o.Permission)
	if err != nil {
		return err
	}
	params["approvalPolicy"] = a
	params["sandbox"] = s
	params["approvalsReviewer"] = "user"
	if o.Model != "" {
		params["model"] = o.Model
	}
	if o.Effort != "" {
		params["config"] = map[string]any{"model_reasoning_effort": o.Effort}
	}
	return nil
}
func codexTurnOptions(params map[string]any, o AgentOptions) error {
	a, _, s, err := codexPermissions(o.Permission)
	if err != nil {
		return err
	}
	params["approvalPolicy"] = a
	params["sandboxPolicy"] = map[string]any{"type": s}
	params["approvalsReviewer"] = "user"
	if o.Model != "" {
		params["model"] = o.Model
	}
	if o.Effort != "" {
		params["effort"] = o.Effort
	}
	return nil
}

func promptFingerprint(prompt string, options AgentOptions) string {
	if options == (AgentOptions{}) {
		return hash(prompt)
	}
	encoded, _ := json.Marshal(options)
	return hash(prompt + "\x00native-options\x00" + string(encoded))
}
func promptDetails(options AgentOptions) map[string]any {
	if options == (AgentOptions{}) {
		return nil
	}
	encoded, _ := json.Marshal(options)
	var value map[string]any
	_ = json.Unmarshal(encoded, &value)
	return map[string]any{"type": "userMessage", "providerOptions": value}
}
func optionsFromDetails(details map[string]any) AgentOptions {
	value := obj(details["providerOptions"])
	return AgentOptions{Model: text(value, "model"), Effort: text(value, "effort"), Permission: text(value, "permission")}
}
