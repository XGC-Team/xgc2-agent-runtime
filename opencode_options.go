package nativeagent

import (
	"context"
	"encoding/json"
	"regexp"
	"sort"
	"strings"
)

// Adapted from T3 Code opencodeRuntime.parseModelsCliOutput at the fixed ref.
// Only native variant names enrich models already advertised over ACP. Unlike
// T3's presentation fallback, absent variants never synthesize reasoning levels.
var openCodeModelLine = regexp.MustCompile(`^[^\s{}]+/[^\s{}]+$`)

func openCodeVariants(output string) map[string][]NamedValue {
	result := map[string][]NamedValue{}
	slug := ""
	lines := []string{}
	flush := func() {
		if slug == "" {
			return
		}
		var model map[string]any
		if json.Unmarshal([]byte(strings.Join(lines, "\n")), &model) != nil {
			return
		}
		variants := obj(model["variants"])
		keys := []string{}
		for key := range variants {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		for _, key := range keys {
			result[slug] = append(result[slug], NamedValue{key, key})
		}
	}
	for _, line := range strings.Split(output, "\n") {
		trimmed := strings.TrimSpace(line)
		if openCodeModelLine.MatchString(trimmed) {
			flush()
			slug = trimmed
			lines = nil
		} else if slug != "" {
			lines = append(lines, line)
		}
	}
	flush()
	return result
}
func inspectOpenCodeVariants(ctx context.Context, p Profile, result *ProviderSetting) {
	output, err := cliOutput(ctx, p, "models", "--verbose")
	if err != nil {
		return
	}
	variants := openCodeVariants(string(output))
	for i := range result.Models {
		if values := variants[result.Models[i].ID]; len(values) > 0 {
			result.Models[i].Efforts = values
		}
	}
}
