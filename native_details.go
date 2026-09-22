package agentruntime

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"math"
	"sort"
	"strings"
	"unicode/utf8"
)

// Only these presentation fields cross the native boundary. RPC envelopes,
// authentication frames and raw reasoning are not journaled. Validated MCP
// raster images have a separate bounded budget so the operator sees the same image.
func codexItemDetails(item map[string]any) map[string]any {
	b := &detailBudget{left: 256 << 10}
	result := map[string]any{"type": text(item, "type")}
	copyText := func(key string, limit int) {
		if value, ok := item[key].(string); ok {
			result[key] = b.text(value, limit)
		}
	}
	switch text(item, "type") {
	case "agentMessage":
		phase := text(item, "phase")
		if phase == "commentary" || phase == "final_answer" {
			result["phase"] = phase
		}
	case "commandExecution":
		result["command"] = b.text(text(item, "command"), 16<<10)
		copyText("cwd", 4096)
		copyNumber(result, item, "exitCode", false)
		copyNumber(result, item, "durationMs", true)
		if actions, ok := item["commandActions"].([]any); ok {
			kept := []any{}
			for i, value := range actions {
				if i >= 64 {
					b.truncated = true
					break
				}
				action := obj(value)
				kind := text(action, "type")
				if kind != "read" && kind != "listFiles" && kind != "search" && kind != "unknown" {
					b.truncated = true
					continue
				}
				entry := map[string]any{"type": kind, "command": b.text(text(action, "command"), 16<<10)}
				for _, key := range []string{"name", "path", "query"} {
					if value, ok := action[key].(string); ok {
						entry[key] = b.text(value, 4096)
					}
				}
				kept = append(kept, entry)
			}
			result["commandActions"] = kept
		}
	case "fileChange":
		changes := []any{}
		for i, value := range arr(item["changes"]) {
			if i >= 64 {
				b.truncated = true
				break
			}
			change := obj(value)
			kind := obj(change["kind"])
			kindType := text(kind, "type")
			if kindType != "add" && kindType != "delete" && kindType != "update" {
				b.truncated = true
				continue
			}
			keptKind := map[string]any{"type": kindType}
			if move, ok := kind["move_path"].(string); ok {
				keptKind["movePath"] = b.text(move, 4096)
			}
			changes = append(changes, map[string]any{"path": b.text(text(change, "path"), 4096), "kind": keptKind, "diff": b.text(text(change, "diff"), 64<<10)})
		}
		result["changes"] = changes
	case "mcpToolCall":
		result["server"] = b.text(text(item, "server"), 4096)
		result["tool"] = b.text(text(item, "tool"), 4096)
		if value, ok := item["arguments"]; ok {
			result["arguments"] = b.jsonValue(value, 0)
		}
		if source := obj(item["result"]); source != nil {
			content := []any{}
			imageBytesLeft := 2 << 20
			imageCount := 0
			for i, value := range arr(source["content"]) {
				if i >= 64 {
					b.truncated = true
					break
				}
				block := obj(value)
				switch text(block, "type") {
				case "image":
					entry, size := nativeRasterImage(block, imageBytesLeft)
					if entry == nil || imageCount >= 8 {
						b.truncated = true
						continue
					}
					imageBytesLeft -= size
					imageCount++
					content = append(content, entry)
				case "text":
					content = append(content, map[string]any{"type": "text", "text": b.text(text(block, "text"), 16<<10)})
				case "resource_link":
					entry := map[string]any{"type": "resource_link", "uri": b.text(text(block, "uri"), 4096)}
					for _, key := range []string{"name", "mimeType", "description"} {
						if value, ok := block[key].(string); ok {
							entry[key] = b.text(value, 4096)
						}
					}
					content = append(content, entry)
				default:
					b.truncated = true
				}
			}
			kept := map[string]any{"content": content}
			if value, ok := source["structuredContent"]; ok {
				kept["structuredContent"] = b.jsonValue(value, 0)
			}
			result["result"] = kept
		}
		if source := obj(item["error"]); source != nil {
			result["error"] = map[string]any{"message": b.text(text(source, "message"), 4096)}
		}
		copyNumber(result, item, "durationMs", true)
	case "webSearch":
		result["query"] = b.text(text(item, "query"), 8192)
		if action := obj(item["action"]); action != nil {
			kept := map[string]any{"type": b.text(text(action, "type"), 64)}
			for _, key := range []string{"query", "url", "pattern"} {
				if value, ok := action[key].(string); ok {
					kept[key] = b.text(value, 8192)
				}
			}
			if values, ok := action["queries"].([]any); ok {
				queries := []string{}
				for i, value := range values {
					if i >= 32 {
						b.truncated = true
						break
					}
					if query, ok := value.(string); ok {
						queries = append(queries, b.text(query, 8192))
					}
				}
				kept["queries"] = queries
			}
			result["action"] = kept
		}
	default:
		return nil
	}
	if b.truncated {
		result["truncated"] = true
	}
	return result
}

func nativeRasterImage(block map[string]any, budget int) (map[string]any, int) {
	mime := text(block, "mimeType")
	data := text(block, "data")
	if (mime != "image/jpeg" && mime != "image/png") || len(data) == 0 || len(data) > base64.StdEncoding.EncodedLen(budget) {
		return nil, 0
	}
	raw, err := base64.StdEncoding.Strict().DecodeString(data)
	if err != nil || len(raw) > budget {
		return nil, 0
	}
	config, format, err := image.DecodeConfig(bytes.NewReader(raw))
	if err != nil || mime != "image/"+format || config.Width < 1 || config.Height < 1 || config.Width > 8192 || config.Height > 8192 || config.Width*config.Height > 32<<20 {
		return nil, 0
	}
	return map[string]any{"type": "image", "mimeType": mime, "data": data, "width": config.Width, "height": config.Height}, len(raw)
}

type detailBudget struct {
	left      int
	nodes     int
	truncated bool
}

func (b *detailBudget) text(value string, limit int) string {
	if limit > b.left {
		limit = b.left
	}
	if len(value) > limit {
		value = value[:limit]
		for !utf8.ValidString(value) && len(value) > 0 {
			value = value[:len(value)-1]
		}
		b.truncated = true
	}
	b.left -= len(value)
	return value
}
func (b *detailBudget) jsonValue(value any, depth int) any {
	b.nodes++
	if depth > 6 || b.nodes > 1024 || b.left == 0 {
		b.truncated = true
		return nil
	}
	switch v := value.(type) {
	case nil, bool:
		return v
	case string:
		return b.text(v, 16<<10)
	case float64:
		if !math.IsNaN(v) && !math.IsInf(v, 0) {
			return v
		}
	case int:
		return v
	case json.Number:
		if n, err := v.Float64(); err == nil && !math.IsNaN(n) && !math.IsInf(n, 0) {
			return n
		}
	case []any:
		result := []any{}
		for i, entry := range v {
			if i >= 64 {
				b.truncated = true
				break
			}
			result = append(result, b.jsonValue(entry, depth+1))
		}
		return result
	case map[string]any:
		result := map[string]any{}
		keys := make([]string, 0, len(v))
		for key := range v {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		for _, key := range keys {
			entry := v[key]
			if len(result) >= 64 {
				b.truncated = true
				break
			}
			if len(key) > 256 || hiddenNativeField(key) {
				b.truncated = true
				continue
			}
			b.left -= min(b.left, len(key))
			result[key] = b.jsonValue(entry, depth+1)
		}
		return result
	}
	b.truncated = true
	return nil
}
func hiddenNativeField(key string) bool {
	key = strings.NewReplacer("-", "", "_", "", ".", "").Replace(strings.ToLower(key))
	switch key {
	case "auth", "authorization", "authentication", "accesstoken", "refreshtoken", "idtoken", "apikey", "password", "secret", "clientsecret", "cookie", "setcookie", "env", "environment", "headers", "meta", "proto", "constructor", "prototype":
		return true
	}
	return false
}
func copyNumber(to, from map[string]any, key string, nonnegative bool) {
	var value float64
	switch n := from[key].(type) {
	case float64:
		value = n
	case int:
		value = float64(n)
	default:
		return
	}
	if math.IsNaN(value) || math.IsInf(value, 0) || math.Trunc(value) != value || math.Abs(value) > 9007199254740991 || (nonnegative && value < 0) {
		return
	}
	to[key] = value
}
