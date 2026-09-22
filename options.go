package nativeagent

import (
	"context"
	"encoding/json"
	"errors"
)

// NativeOptions are selections from the provider's discovered capabilities.
// Empty values preserve the session snapshot; no model alias or fallback is invented.
type NativeOptions struct {
	Model      string `json:"model,omitempty"`
	Effort     string `json:"effort,omitempty"`
	Permission string `json:"permission,omitempty"`
}

func mergeOptions(base, override NativeOptions) NativeOptions {
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
	PromptWithOptions(context.Context, string, string, NativeOptions) error
}

// Adapted from T3 Code CodexSessionRuntime.ts at 6349a0e68a958cc51b7b5198683c1d1db88b8d28.
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
		return "", "", "", errors.New("unsupported native permission")
	}
}
func codexThreadOptions(params map[string]any, o NativeOptions) error {
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
func codexTurnOptions(params map[string]any, o NativeOptions) error {
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

func promptFingerprint(prompt string, options NativeOptions) string {
	if options == (NativeOptions{}) {
		return hash(prompt)
	}
	encoded, _ := json.Marshal(options)
	return hash(prompt + "\x00native-options\x00" + string(encoded))
}
func promptDetails(options NativeOptions) map[string]any {
	if options == (NativeOptions{}) {
		return nil
	}
	encoded, _ := json.Marshal(options)
	var value map[string]any
	_ = json.Unmarshal(encoded, &value)
	return map[string]any{"type": "userMessage", "nativeOptions": value}
}
func optionsFromDetails(details map[string]any) NativeOptions {
	value := obj(details["nativeOptions"])
	return NativeOptions{Model: text(value, "model"), Effort: text(value, "effort"), Permission: text(value, "permission")}
}
