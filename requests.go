package agentruntime

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
)

func (d *rpcDriver) onRequest(ctx context.Context, method string, p map[string]any) (any, error) {
	if !d.matches(p) {
		return nil, ErrStale
	}
	d.mu.Lock()
	turnContext := d.turnContext
	d.mu.Unlock()
	if turnContext == nil {
		return nil, ErrStale
	}
	ctx, cancel := context.WithCancel(ctx)
	stop := context.AfterFunc(turnContext, cancel)
	defer stop()
	defer cancel()
	r := Request{Kind: "permission", Title: method, Options: []Option{}, Questions: []Question{}, SourceMethod: method}
	if d.profile.Provider == "codex" {
		r.AgentThreadID = text(p, "threadId")
		r.AgentTurnID = text(p, "turnId")
		r.AgentItemID = text(p, "itemId")
	}
	switch method {
	case "session/request_permission":
		call := obj(p["toolCall"])
		r.Title = text(call, "title")
		if r.Title == "" {
			r.Title = "Tool permission"
		}
		r.Text = toolContent(arr(call["content"]))
		for _, v := range arr(p["options"]) {
			o := obj(v)
			kind := text(o, "kind")
			if kind == "allow_once" || kind == "reject_once" {
				r.Options = append(r.Options, Option{ID: text(o, "optionId"), Label: text(o, "name"), Kind: kind})
			}
		}
		if len(r.Options) == 0 {
			return map[string]any{"outcome": map[string]any{"outcome": "cancelled"}}, nil
		}
	case "item/commandExecution/requestApproval", "item/fileChange/requestApproval":
		r.Title = "Operation approval"
		r.Details = &RequestDetails{Command: text(p, "command"), Cwd: text(p, "cwd"), Reason: text(p, "reason"), GrantRoot: text(p, "grantRoot")}
		r.Text = text(p, "command") + "\n" + text(p, "reason")
		if n := obj(p["networkApprovalContext"]); n != nil {
			r.Title = "Network access"
			r.Details.Network = &RequestNetwork{Protocol: text(n, "protocol"), Host: text(n, "host")}
			r.Text = text(n, "protocol") + ": " + text(n, "host") + "\n" + text(p, "reason")
		}
		r.Options = []Option{{ID: "accept", Label: "Allow this operation", Kind: "allow_once"}, {ID: "decline", Label: "Decline", Kind: "reject_once"}}
		if decisions, ok := p["availableDecisions"]; ok {
			allowed := map[string]bool{}
			for _, v := range arr(decisions) {
				allowed[str(v)] = true
			}
			options := []Option{}
			for _, o := range r.Options {
				if allowed[o.ID] {
					options = append(options, o)
				}
			}
			r.Options = options
		}
	case "item/permissions/requestApproval":
		// The client may grant exactly the profile it was shown, or nothing.
		profile := obj(p["permissions"])
		summary := codexPermissionSummary(profile)
		if summary == "" {
			return nil, errors.New("permission request without a reviewable profile")
		}
		r.Title = "Permission request"
		r.Details = &RequestDetails{Cwd: text(p, "cwd"), Reason: text(p, "reason")}
		r.Text = summary + "\n" + text(p, "reason")
		r.Options = []Option{{ID: "accept", Label: "Grant these permissions", Kind: "allow_once"}, {ID: "decline", Label: "Decline", Kind: "reject_once"}}
	case "item/tool/requestUserInput", "tool/requestUserInput":
		r.Kind = "question"
		r.Title = "Input needed"
		for _, v := range arr(p["questions"]) {
			q := obj(v)
			secret, _ := q["isSecret"].(bool)
			if secret {
				return nil, errors.New("secret input requires the provider client")
			}
			question := Question{ID: text(q, "id"), Text: text(q, "question"), Header: text(q, "header"), Options: []Option{}}
			question.FreeText, _ = q["isOther"].(bool)
			for _, v := range arr(q["options"]) {
				o := obj(v)
				label := text(o, "label")
				question.Options = append(question.Options, Option{ID: label, Label: label, Kind: "answer", Description: text(o, "description")})
			}
			if len(question.Options) == 0 {
				question.FreeText = true
			}
			r.Questions = append(r.Questions, question)
		}
	case "cursor/ask_question":
		r.Kind = "question"
		r.Title = text(p, "title")
		for _, v := range arr(p["questions"]) {
			q := obj(v)
			question := Question{ID: text(q, "id"), Text: text(q, "prompt"), Options: []Option{}}
			question.Multiple, _ = q["allowMultiple"].(bool)
			for _, v := range arr(q["options"]) {
				o := obj(v)
				question.Options = append(question.Options, Option{ID: text(o, "id"), Label: text(o, "label"), Kind: "answer"})
			}
			r.Questions = append(r.Questions, question)
		}
	case "cursor/create_plan":
		r.Kind = "plan"
		r.Title = text(p, "name")
		r.Text = text(p, "plan")
		r.Options = []Option{{ID: "accepted", Label: "Accept plan", Kind: "allow_once"}, {ID: "rejected", Label: "Reject plan", Kind: "reject_once"}}
	default:
		d.emit(Event{Kind: "notice", Status: "blocked", Text: "The client requested an operation this host cannot handle. The operation was rejected.", SourceMethod: method})
		return nil, errors.New("unsupported client request")
	}
	if err := validateRequest(r); err != nil {
		return nil, err
	}
	answer, err := d.ask(ctx, r)
	if err != nil {
		answer = Answer{Cancel: true}
	}
	if err == nil {
		if err = r.ValidateAnswer(answer); err != nil {
			return nil, err
		}
	}
	if ctx.Err() != nil {
		answer = Answer{Cancel: true}
	}
	switch method {
	case "session/request_permission":
		outcome := map[string]any{"outcome": "cancelled"}
		if !answer.Cancel {
			outcome = map[string]any{"outcome": "selected", "optionId": answer.OptionID}
		}
		return map[string]any{"outcome": outcome}, nil
	case "item/commandExecution/requestApproval", "item/fileChange/requestApproval":
		decision := answer.OptionID
		if answer.Cancel {
			decision = "cancel"
		}
		return map[string]any{"decision": decision}, nil
	case "item/permissions/requestApproval":
		// Granting is for the turn only; declining answers with an empty grant,
		// which the client reads as the permission withheld.
		if answer.Cancel || answer.OptionID != "accept" {
			return map[string]any{"permissions": map[string]any{}}, nil
		}
		return map[string]any{"permissions": obj(p["permissions"]), "scope": "turn"}, nil
	case "cursor/create_plan":
		outcome := answer.OptionID
		if answer.Cancel {
			outcome = "cancelled"
		}
		return map[string]any{"outcome": map[string]any{"outcome": outcome}}, nil
	case "cursor/ask_question":
		if answer.Cancel {
			return map[string]any{"outcome": map[string]any{"outcome": "cancelled"}}, nil
		}
		answers := []any{}
		for _, q := range r.Questions {
			answers = append(answers, map[string]any{"questionId": q.ID, "selectedOptionIds": answer.Answers[q.ID]})
		}
		return map[string]any{"outcome": map[string]any{"outcome": "answered", "answers": answers}}, nil
	default:
		answers := map[string]any{}
		if !answer.Cancel {
			for key, values := range answer.Answers {
				answers[key] = map[string]any{"answers": values}
			}
		}
		return map[string]any{"answers": answers}, nil
	}
}
func validateRequest(r Request) error {
	for _, value := range []string{r.AgentThreadID, r.AgentTurnID, r.AgentItemID, r.SourceMethod} {
		if len(value) > 512 {
			return errors.New("request identity exceeds display limits")
		}
	}
	if r.Details != nil {
		if len(r.Details.ToolName) > 256 || len(r.Details.Target) > 8192 || len(r.Details.Preview) > approvalPreviewLimit {
			return errors.New("request review details exceed bounds")
		}
		for _, value := range []string{r.Details.Command, r.Details.Cwd, r.Details.Reason, r.Details.GrantRoot} {
			if len(value) > 8192 {
				return errors.New("approval detail exceeds display limits")
			}
		}
		if n := r.Details.Network; n != nil && (len(n.Protocol) > 64 || len(n.Host) > 4096) {
			return errors.New("network approval exceeds display limits")
		}
	}
	if len(r.Title) > 4096 || len(r.Text) > MaxText || len(r.Options) > 32 || len(r.Questions) > 16 {
		return errors.New("request exceeds display limits")
	}
	if len(r.Options) == 0 && len(r.Questions) == 0 {
		return errors.New("request has no supported response")
	}
	seen := map[string]bool{}
	for _, o := range r.Options {
		if o.ID == "" || len(o.ID) > 256 || len(o.Label) > 4096 || seen[o.ID] {
			return errors.New("invalid option")
		}
		seen[o.ID] = true
	}
	seen = map[string]bool{}
	for _, q := range r.Questions {
		if q.ID == "" || len(q.ID) > 256 || seen[q.ID] || len(q.Text) > 8192 || len(q.Header) > 256 || len(q.Options) > 32 || (!q.FreeText && len(q.Options) == 0) {
			return errors.New("invalid question")
		}
		seen[q.ID] = true
		options := map[string]bool{}
		for _, o := range q.Options {
			if strings.TrimSpace(o.ID) == "" || len(o.ID) > 8192 || len(o.Label) > 8192 || len(o.Description) > 8192 || options[o.ID] {
				return errors.New("invalid answer option")
			}
			options[o.ID] = true
		}
	}
	return nil
}

// codexPermissionSummary lists what a Codex permission profile would grant, for
// the operator to review. It is empty when the profile names nothing, or
// anything the summary cannot show: a grant is never wider than what was read.
func codexPermissionSummary(profile map[string]any) string {
	if !onlyKeys(profile, "network", "fileSystem") {
		return ""
	}
	lines := []string{}
	if raw, present := profile["network"]; present && raw != nil {
		network := obj(raw)
		enabled, ok := network["enabled"].(bool)
		if network == nil || !onlyKeys(network, "enabled") || (!ok && network["enabled"] != nil) {
			return ""
		}
		if ok {
			state := "disabled"
			if enabled {
				state = "enabled"
			}
			lines = append(lines, "Network access: "+state)
		}
	}
	if raw, present := profile["fileSystem"]; present && raw != nil {
		files := obj(raw)
		if files == nil || !onlyKeys(files, "entries", "read", "write", "globScanMaxDepth") {
			return ""
		}
		for _, access := range []string{"read", "write"} {
			values, present := files[access]
			if !present || values == nil {
				continue
			}
			list, ok := values.([]any)
			if !ok {
				return ""
			}
			var paths []string
			for _, v := range list {
				path, ok := v.(string)
				if !ok || path == "" {
					return ""
				}
				paths = append(paths, path)
			}
			if len(paths) > 0 {
				lines = append(lines, strings.ToUpper(access[:1])+access[1:]+": "+strings.Join(paths, ", "))
			}
		}
		entries, _ := files["entries"].([]any)
		if len(entries) > 64 {
			return ""
		}
		for _, raw := range entries {
			entry, path := obj(raw), obj(obj(raw)["path"])
			target := ""
			switch text(path, "type") {
			case "path":
				target = text(path, "path")
			case "glob_pattern":
				target = text(path, "pattern")
			case "special":
				encoded, _ := json.Marshal(path["value"])
				target = string(encoded)
			}
			if access := text(entry, "access"); target == "" || (access != "read" && access != "write" && access != "deny") {
				return ""
			}
			lines = append(lines, "File access ("+text(entry, "access")+"): "+target)
		}
	}
	return strings.Join(lines, "\n")
}

// onlyKeys reports whether m has no key other than the listed ones.
func onlyKeys(m map[string]any, keys ...string) bool {
	for key := range m {
		known := false
		for _, k := range keys {
			known = known || k == key
		}
		if !known {
			return false
		}
	}
	return true
}
