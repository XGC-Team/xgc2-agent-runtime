package nativeagent

import (
	"context"
	"encoding/json"
	"io"
	"strings"
	"testing"
	"unicode/utf8"
)

func TestClaudePermissionCarriesExactReviewOutsideConversation(t *testing.T) {
	for _, sample := range []struct {
		tool, target, preview string
		input                 map[string]any
	}{
		{"Write", "/reviewed/experiment/config.yaml", "gain: 2", map[string]any{"file_path": "/reviewed/experiment/config.yaml", "content": "gain: 2"}},
		{"Edit", "/reviewed/experiment/config.yaml", "Replace all matching blocks\n--- Before\ngain: 1\n+++ After\ngain: 2", map[string]any{"file_path": "/reviewed/experiment/config.yaml", "old_string": "gain: 1", "new_string": "gain: 2", "replace_all": true}},
		{"WebFetch", "https://example.test/evidence", "Find the run status", map[string]any{"url": "https://example.test/evidence", "prompt": "Find the run status"}},
	} {
		t.Run(sample.tool, func(t *testing.T) {
			ctx, cancel := context.WithCancel(t.Context())
			defer cancel()
			reader, writer := io.Pipe()
			defer reader.Close()
			defer writer.Close()
			control := &claudeControl{ctx: ctx, input: writer, native: "thread", pending: map[string]context.CancelFunc{}, ask: func(_ context.Context, request Request) (Answer, error) {
				if request.Details.ToolName != sample.tool || request.Details.Target != sample.target || request.Details.Preview != sample.preview || request.Details.Truncated {
					t.Errorf("request cannot be reviewed independently: %+v", request.Details)
				}
				return Answer{OptionID: "allow"}, nil
			}}
			defer control.close()
			control.request(map[string]any{"request_id": "request", "request": map[string]any{"subtype": "can_use_tool", "tool_name": sample.tool, "tool_use_id": "item", "input": sample.input, "authorization": "do-not-copy"}})
			var response map[string]any
			if err := json.NewDecoder(reader).Decode(&response); err != nil {
				t.Fatal(err)
			}
			decision := obj(obj(response["response"])["response"])
			original, _ := json.Marshal(sample.input)
			approved, _ := json.Marshal(decision["updatedInput"])
			if decision["behavior"] != "allow" || string(original) != string(approved) {
				t.Fatal("review changed the approved provider input", decision)
			}
		})
	}
}

func TestClaudeReviewIsBoundedWithoutHidingEditDestination(t *testing.T) {
	details := claudePermissionDetails("Edit", map[string]any{"file_path": "/reviewed/config", "old_string": strings.Repeat("旧", approvalPreviewLimit), "new_string": "new destination"}, map[string]any{"authorization": "secret"})
	if !details.Truncated || len(details.Preview) > approvalPreviewLimit || !utf8.ValidString(details.Preview) || !strings.Contains(details.Preview, "+++ After\nnew destination") {
		t.Fatal("bounded edit review lost the replacement", details)
	}
	request := Request{Kind: "permission", Title: "Edit", Details: details, Options: []Option{{ID: "allow"}}}
	if err := validateRequest(request); err != nil {
		t.Fatal(err)
	}
	details.Target = strings.Repeat("x", 8193)
	if validateRequest(request) == nil {
		t.Fatal("an oversized exact target must be refused, not silently shortened")
	}
}
