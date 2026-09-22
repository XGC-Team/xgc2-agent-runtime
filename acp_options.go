package agentruntime

// ACP discovery/configuration is adapted from the fixed T3 Code provider
// adapters (GrokAcpSupport, CursorProvider, CursorAdapter), not Codex aliases.
import (
	"context"
	"errors"
	"os"
	"strings"
)

func acpConfigOptions(setup map[string]any) []any { return arr(setup["configOptions"]) }
func configChoices(option map[string]any) []NamedValue {
	result := []NamedValue{}
	for _, raw := range arr(option["options"]) {
		v := obj(raw)
		if children := arr(v["options"]); len(children) > 0 {
			result = append(result, configChoices(map[string]any{"options": children})...)
			continue
		}
		id := text(v, "value")
		if id != "" {
			label := text(v, "name")
			if label == "" {
				label = id
			}
			result = append(result, NamedValue{id, label})
		}
	}
	return result
}
func effortConfig(options []any) map[string]any {
	for _, raw := range options {
		v := obj(raw)
		id := strings.ToLower(text(v, "id"))
		name := strings.ToLower(text(v, "name"))
		category := text(v, "category")
		if text(v, "type") == "select" && (category == "thought_level" || id == "effort" || id == "reasoning" || strings.Contains(name, "effort") || strings.Contains(name, "reasoning")) {
			return v
		}
	}
	return nil
}
func modelConfig(options []any) map[string]any {
	for _, raw := range options {
		v := obj(raw)
		if text(v, "type") == "select" && (text(v, "category") == "model" || text(v, "id") == "model") {
			return v
		}
	}
	return nil
}
func modeConfig(options []any) map[string]any {
	for _, raw := range options {
		v := obj(raw)
		if text(v, "type") == "select" && (text(v, "category") == "mode" || text(v, "id") == "mode") {
			return v
		}
	}
	return nil
}
func parseACPInventory(result *ProviderSetting, setup map[string]any) {
	modelState := obj(setup["models"])
	if len(modelState) == 0 {
		modelState = obj(obj(setup["_meta"])["modelState"])
	}
	models := []Model{}
	for _, raw := range arr(modelState["availableModels"]) {
		v := obj(raw)
		id := text(v, "modelId")
		if id == "" {
			continue
		}
		m := Model{ID: id, Label: text(v, "name"), Efforts: []NamedValue{}}
		if m.Label == "" {
			m.Label = id
		}
		meta := obj(v["_meta"])
		for _, entry := range arr(meta["reasoningEfforts"]) {
			e := obj(entry)
			id := text(e, "value")
			if id == "" {
				id = text(e, "id")
			}
			if id != "" {
				label := text(e, "label")
				if label == "" {
					label = id
				}
				m.Efforts = append(m.Efforts, NamedValue{id, label})
			}
		}
		m.DefaultEffort = text(meta, "reasoningEffort")
		models = append(models, m)
	}
	options := acpConfigOptions(setup)
	modelChoice := modelConfig(options)
	if modelChoice != nil {
		models = []Model{}
		for _, choice := range configChoices(modelChoice) {
			models = append(models, Model{ID: choice.ID, Label: choice.Label, Efforts: []NamedValue{}})
		}
	}
	if len(models) > 0 {
		result.Models = models
	}
	currentModel := text(modelState, "currentModelId")
	if modelChoice != nil {
		currentModel = text(modelChoice, "currentValue")
	}
	if currentModel != "" {
		result.Defaults.Model = currentModel
	}
	if effort := effortConfig(options); effort != nil {
		for i := range result.Models {
			if result.Models[i].ID == currentModel {
				result.Models[i].Efforts = configChoices(effort)
				result.Models[i].DefaultEffort = text(effort, "currentValue")
				result.Defaults.Effort = text(effort, "currentValue")
			}
		}
	}
	if config := modeConfig(options); config != nil {
		for _, choice := range configChoices(config) {
			result.Permissions = append(result.Permissions, Permission{ID: choice.ID, Label: choice.Label, Description: choice.Label + " mode."})
		}
		result.Defaults.Permission = text(config, "currentValue")
	}
	modeState := obj(setup["modes"])
	for _, raw := range arr(modeState["availableModes"]) {
		mode := obj(raw)
		id := text(mode, "id")
		duplicate := false
		for _, existing := range result.Permissions {
			duplicate = duplicate || existing.ID == id
		}
		if duplicate {
			continue
		}
		if id == "" {
			continue
		}
		label := text(mode, "name")
		if label == "" {
			label = id
		}
		result.Permissions = append(result.Permissions, Permission{ID: id, Label: label, Description: text(mode, "description")})
	}
	if text(modeState, "currentModeId") != "" {
		result.Defaults.Permission = text(modeState, "currentModeId")
	}
}
func inspectACP(ctx context.Context, p Profile, result *ProviderSetting) {
	cwd, err := os.MkdirTemp("", "xgc-native-capabilities-*")
	if err != nil {
		return
	}
	defer os.RemoveAll(cwd)
	args, _ := commandArgs(p.Provider)
	child, err := startChild(p, args, cwd)
	if err != nil {
		result.Detail = "ACP capability inspection could not start."
		return
	}
	peer := newPeer(child.stdin, child.stdout, true, func(string, map[string]any) {}, func(context.Context, string, map[string]any) (any, error) { return nil, ErrUnavailable })
	go func() { <-peer.done; _ = child.Wait() }()
	defer func() { _ = peer.Close(); child.Stop() }()
	capabilities := map[string]any{"fs": map[string]any{"readTextFile": false, "writeTextFile": false}, "terminal": false}
	if p.Provider == "cursor" {
		capabilities["_meta"] = map[string]any{"parameterizedModelPicker": true}
	}
	init, err := peer.Call(ctx, "initialize", map[string]any{"protocolVersion": 1, "clientInfo": map[string]any{"name": "xgc-agent-runtime", "version": "0.1.0"}, "clientCapabilities": capabilities})
	if err != nil {
		result.Detail = "ACP metadata handshake failed."
		return
	}
	parseACPInventory(result, init)
	if p.Provider == "cursor" {
		if result.Login.Status != "authenticated" {
			result.Detail = "Cursor login is required before its model catalog can be queried."
			return
		}
		if _, err = peer.Call(ctx, "authenticate", map[string]any{"methodId": "cursor_login"}); err != nil {
			result.Detail = "The existing Cursor login could not be opened."
			return
		}
	}
	if p.Provider == "grok" && len(result.Models) > 0 {
		result.Detail = "Model and reasoning metadata came from ACP initialize; no prompt was sent."
		return
	}

	// Session setup is a native metadata probe in an empty private directory, not
	// a user/project management feature. Client filesystem and terminal are absent.
	if p.Provider == "grok" {
		return
	} // Never trigger interactive login from discovery.
	setup, err := peer.Call(ctx, "session/new", map[string]any{"cwd": cwd, "mcpServers": []any{}})
	if err != nil {
		result.Detail = "Session capabilities could not be verified before the metadata deadline."
		return
	}
	parseACPInventory(result, setup)
	// T3 initializes the native session before requesting Cursor's extended
	// model catalog. The session already supplies usable modes/current effort;
	// a slow optional extension must not erase those verified capabilities.
	if p.Provider == "cursor" {
		if catalog, e := peer.Call(ctx, "cursor/list_available_models", map[string]any{}); e == nil {
			models := []Model{}
			for _, raw := range arr(catalog["models"]) {
				v := obj(raw)
				id := text(v, "value")
				if id == "" {
					continue
				}
				m := Model{ID: id, Label: text(v, "name"), Efforts: []NamedValue{}}
				if m.Label == "" {
					m.Label = id
				}
				if effort := effortConfig(arr(v["configOptions"])); effort != nil {
					m.Efforts = configChoices(effort)
					m.DefaultEffort = text(effort, "currentValue")
				}
				models = append(models, m)
			}
			if len(models) > 0 {
				result.Models = models
			}
		}
	}
	result.Detail = "Models and modes were read from the ACP interface; no prompt was sent."
}
func (d *rpcDriver) applyACPOptions(ctx context.Context, o AgentOptions) error {
	if o == (AgentOptions{}) {
		return nil
	}
	d.mu.Lock()
	setup, native := d.acpSetup, d.nativeSession
	d.mu.Unlock()
	options := acpConfigOptions(setup)
	if o.Model != "" {
		var response map[string]any
		var err error
		if config := modelConfig(options); config != nil {
			response, err = d.peer.Call(ctx, "session/set_config_option", map[string]any{"sessionId": native, "configId": text(config, "id"), "value": o.Model})
		} else {
			params := map[string]any{"sessionId": native, "modelId": o.Model}
			if d.profile.Provider == "grok" && o.Effort != "" {
				params["_meta"] = map[string]any{"reasoningEffort": o.Effort}
			}
			response, err = d.peer.Call(ctx, "session/set_model", params)
		}
		if err != nil {
			return err
		}
		if len(arr(response["configOptions"])) > 0 {
			setup["configOptions"] = response["configOptions"]
			options = acpConfigOptions(setup)
		}
	}
	if o.Effort != "" && d.profile.Provider != "grok" {
		config := effortConfig(options)
		if config == nil {
			return errors.New("session does not advertise a reasoning control for the selected model")
		}
		valid := false
		for _, v := range configChoices(config) {
			valid = valid || v.ID == o.Effort
		}
		if !valid {
			return errors.New("reasoning selection is no longer available")
		}
		if _, err := d.peer.Call(ctx, "session/set_config_option", map[string]any{"sessionId": native, "configId": text(config, "id"), "value": o.Effort}); err != nil {
			return err
		}
	}
	if o.Permission != "" && d.profile.Provider != "grok" {
		mode := o.Permission
		if d.profile.Provider == "cursor" && (mode == "approval-required" || mode == "full-access" || mode == "auto") {
			mode = "agent"
		}
		if config := modeConfig(options); config != nil {
			valid := false
			for _, v := range configChoices(config) {
				valid = valid || v.ID == mode
			}
			if !valid {
				return errors.New("mode is no longer available")
			}
			_, err := d.peer.Call(ctx, "session/set_config_option", map[string]any{"sessionId": native, "configId": text(config, "id"), "value": mode})
			if err != nil {
				return err
			}
		} else {
			valid := false
			for _, raw := range arr(obj(setup["modes"])["availableModes"]) {
				valid = valid || text(obj(raw), "id") == mode
			}
			if !valid {
				return errors.New("session does not advertise the selected mode")
			}
			if _, err := d.peer.Call(ctx, "session/set_mode", map[string]any{"sessionId": native, "modeId": mode}); err != nil {
				return err
			}
		}
	}

	return nil
}
