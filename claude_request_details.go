package nativeagent

import (
	"encoding/json"
	"strings"
	"unicode/utf8"
)

const approvalPreviewLimit = 16 << 10

// Summarize the exact tool input on the request itself: a notification may be
// reviewed without the conversation's earlier tool events. Never copy the
// arbitrary control frame, credentials, or hidden provider configuration.
func claudePermissionDetails(tool string, input, body map[string]any) *RequestDetails {
	details := &RequestDetails{ToolName: tool, Command: text(input, "command"), Cwd: text(input, "cwd"), Reason: text(body, "decision_reason")}
	preview := ""
	switch tool {
	case "Write":
		details.Target = text(input, "file_path")
		preview = text(input, "content")
	case "Edit":
		details.Target = text(input, "file_path")
		before, beforeCut := boundedApprovalPreview(text(input, "old_string"), (approvalPreviewLimit-128)/2)
		after, afterCut := boundedApprovalPreview(text(input, "new_string"), (approvalPreviewLimit-128)/2)
		scope := "Replace the first matching block"
		if input["replace_all"] == true {
			scope = "Replace all matching blocks"
		}
		preview = scope + "\n--- Before\n" + before + "\n+++ After\n" + after
		details.Truncated = beforeCut || afterCut
	case "Read":
		details.Target = text(input, "file_path")
	case "WebFetch":
		details.Target = text(input, "url")
		preview = text(input, "prompt")
	case "WebSearch":
		preview = text(input, "query")
	default:
		if strings.HasPrefix(tool, "mcp__") {
			// Review only the requested arguments, never the server binding or
			// the CLI control envelope that carries transport information.
			encoded, _ := json.MarshalIndent(input, "", "  ")
			preview = string(encoded)
		}
	}
	var cut bool
	details.Preview, cut = boundedApprovalPreview(preview, approvalPreviewLimit)
	details.Truncated = details.Truncated || cut
	return details
}

func boundedApprovalPreview(value string, limit int) (string, bool) {
	if len(value) <= limit {
		return value, false
	}
	const suffix = "\n… preview truncated"
	end := limit - len(suffix)
	for end > 0 && !utf8.RuneStart(value[end]) {
		end--
	}
	return strings.TrimRight(value[:end], "\n") + suffix, true
}
