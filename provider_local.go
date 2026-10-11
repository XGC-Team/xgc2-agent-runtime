package agentruntime

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
)

// Native login status is read through each CLI's public status command. Account
// identifiers, credentials and arbitrary output are never returned or persisted.
func inspectLocalLogin(ctx context.Context, p Profile, result *ProviderSetting) {
	switch p.Provider {
	case "grok":
		output, err := cliOutput(ctx, p, "--no-auto-update", "models")
		if err != nil {
			return
		}
		value := strings.ToLower(string(output))
		if strings.Contains(value, "not authenticated") || strings.Contains(value, "not logged in") {
			result.Login = LoginStatus{"unauthenticated", "Log in using the Grok client."}
		} else if strings.Contains(value, "you are logged in") {
			result.Login = LoginStatus{"authenticated", "Using the Grok login."}
		}
		if len(result.Models) == 0 {
			for _, line := range strings.Split(string(output), "\n") {
				parts := strings.Fields(line)
				if len(parts) >= 2 && (parts[0] == "*" || parts[0] == "-") {
					id := parts[1]
					result.Models = append(result.Models, Model{ID: id, Label: id, Efforts: []NamedValue{}})
					if strings.Contains(line, "(default)") {
						result.Defaults.Model = id
					}
				}
			}
		}
	case "cursor":
		output, err := cliOutput(ctx, p, "about", "--json")
		if err != nil {
			output, err = cliOutput(ctx, p, "about")
		}
		if err != nil {
			return
		}
		var value map[string]any
		email := ""
		known := false
		if json.Unmarshal(output, &value) == nil {
			raw, ok := value["userEmail"]
			known = ok
			email = str(raw)
		} else {
			for _, line := range strings.Split(string(output), "\n") {
				line = strings.TrimSpace(line)
				if strings.HasPrefix(line, "User Email") {
					known = true
					email = strings.TrimSpace(strings.TrimPrefix(line, "User Email"))
					break
				}
			}
		}
		if known {
			if email == "" || strings.Contains(strings.ToLower(email), "not logged in") || strings.Contains(strings.ToLower(email), "login required") {
				result.Login = LoginStatus{"unauthenticated", "Log in using the Cursor client."}
			} else {
				result.Login = LoginStatus{"authenticated", "Using the Cursor login."}
			}
		}
	}
}

// grokPermissionArgs is the launch of `grok agent` for a permission mode. The
// argument beats the user's Grok configuration, so asking cannot inherit a
// configured always-approve. Grok has no accept-edits launch mode (acceptEdits
// exists only as a settings-file default and `grok agent` treats it as ask), so a
// conversation that stored that value launches asking, as it always did.
func grokPermissionArgs(permission string) ([]string, error) {
	switch permission {
	case "":
		return []string{"--no-auto-update", "agent", "stdio"}, nil
	case "approval-required", "auto-accept-edits":
		return []string{"--no-auto-update", "--permission-mode", "default", "agent", "stdio"}, nil
	case "auto":
		return []string{"--no-auto-update", "--permission-mode", "auto", "agent", "stdio"}, nil
	case "full-access":
		return []string{"--no-auto-update", "agent", "--always-approve", "stdio"}, nil
	default:
		return nil, errors.New("unsupported Grok permission")
	}
}
func grokPermissions() []Permission {
	return []Permission{{"approval-required", "Ask for approval", "Use Grok's default permission mode."}, {"auto", "Auto review", "Grok's classifier decides; the operator is asked about what it blocks."}, {"full-access", "Full access", "Use Grok's explicit always-approve agent mode."}}
}

// Grok reads these metadata fields on its ACP messages (t3code provider-grok).
// With the client type "extension" its auto mode asks the client about an action
// its classifier blocks; the default type gets a silent denial. A cancel with
// the ctrl_c trigger also silences stale background-task wake prompts until the
// next real turn.
var (
	grokInitializeMeta = map[string]any{"clientType": "extension"}
	grokCancelMeta     = map[string]any{"cancelTrigger": "ctrl_c"}
)

func cursorPermissionArgs(permission string) ([]string, error) {
	switch permission {
	case "", "agent", "ask", "plan", "approval-required":
		return []string{"acp"}, nil
	case "full-access":
		return []string{"--force", "acp"}, nil
	case "auto":
		return []string{"--auto-review", "acp"}, nil
	default:
		return nil, errors.New("unsupported Cursor permission")
	}
}
func acpLaunchPermission(provider, permission string) string {
	if provider == "cursor" {
		if permission == "full-access" || permission == "auto" {
			return permission
		}
		return "approval-required"
	}
	return permission
}

// acpInitializeParams is the ACP initialize request of a session.
func acpInitializeParams(provider string, capabilities map[string]any) map[string]any {
	params := map[string]any{"protocolVersion": 1, "clientInfo": map[string]any{"name": "xgc-agent-runtime", "version": "0.1.0"}, "clientCapabilities": capabilities}
	if provider == "grok" {
		params["_meta"] = grokInitializeMeta
	}
	return params
}
