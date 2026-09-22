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
			result.Login = LoginStatus{"unauthenticated", "Log in using the native Grok client."}
		} else if strings.Contains(value, "you are logged in") {
			result.Login = LoginStatus{"authenticated", "Using the native Grok login."}
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
				result.Login = LoginStatus{"unauthenticated", "Log in using the native Cursor client."}
			} else {
				result.Login = LoginStatus{"authenticated", "Using the native Cursor login."}
			}
		}
	}
}
func grokPermissionArgs(permission string) ([]string, error) {
	switch permission {
	case "":
		return []string{"--no-auto-update", "agent", "stdio"}, nil
	case "approval-required":
		return []string{"--no-auto-update", "--permission-mode", "default", "agent", "stdio"}, nil
	case "auto-accept-edits":
		return []string{"--no-auto-update", "--permission-mode", "acceptEdits", "agent", "stdio"}, nil
	case "full-access":
		return []string{"--no-auto-update", "agent", "--always-approve", "stdio"}, nil
	default:
		return nil, errors.New("unsupported native Grok permission")
	}
}
func grokPermissions() []Permission {
	return []Permission{{"approval-required", "Ask for approval", "Use Grok's native default permission mode."}, {"auto-accept-edits", "Accept edits", "Use Grok's native acceptEdits permission mode."}, {"full-access", "Full access", "Use Grok's explicit always-approve agent mode."}}
}

func cursorPermissionArgs(permission string) ([]string, error) {
	switch permission {
	case "", "agent", "ask", "plan", "approval-required":
		return []string{"acp"}, nil
	case "full-access":
		return []string{"--force", "acp"}, nil
	case "auto":
		return []string{"--auto-review", "acp"}, nil
	default:
		return nil, errors.New("unsupported native Cursor permission")
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
