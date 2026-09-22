package agentruntime

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"image"
	"image/color"
	"image/png"
	"os"
	"testing"
)

func nativeImageFixture(t *testing.T) (map[string]any, string) {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, 16, 8))
	img.Set(2, 2, color.RGBA{R: 255, A: 255})
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatal(err)
	}
	data := base64.StdEncoding.EncodeToString(buf.Bytes())
	return map[string]any{"type": "image", "mimeType": "image/png", "data": data}, data
}

func TestNativeMCPImageSurvivesJournalWithoutChangingBytes(t *testing.T) {
	block, data := nativeImageFixture(t)
	details := codexItemDetails(map[string]any{"type": "mcpToolCall", "server": "fixture", "tool": "view", "result": map[string]any{"content": []any{block}}})
	file, err := os.CreateTemp(t.TempDir(), "image-*.jsonl")
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	session := newSession(Session{ID: "s_image", Provider: "codex"}, file)
	if err := session.appendLocked(Event{Kind: "item.snapshot", Role: "tool", ItemID: "view-1", Details: details}); err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(file.Name())
	if err != nil {
		t.Fatal(err)
	}
	var event Event
	if err := json.Unmarshal(raw, &event); err != nil {
		t.Fatal(err)
	}
	kept := obj(arr(obj(event.Details["result"])["content"])[0])
	if kept["data"] != data || kept["width"] != float64(16) || kept["height"] != float64(8) {
		t.Fatal("image changed across journal")
	}
}

func TestNativeMCPImageRejectsInvalidAndBoundsTotal(t *testing.T) {
	block, _ := nativeImageFixture(t)
	for _, bad := range []map[string]any{
		{"type": "image", "mimeType": "image/svg+xml", "data": block["data"]},
		{"type": "image", "mimeType": "image/jpeg", "data": block["data"]},
		{"type": "image", "mimeType": "image/png", "data": "invalid"},
	} {
		if kept, _ := nativeRasterImage(bad, 2<<20); kept != nil {
			t.Fatal("accepted invalid image")
		}
	}
	if kept, _ := nativeRasterImage(block, 1); kept != nil {
		t.Fatal("accepted over-budget image")
	}
	blocks := []any{}
	for i := 0; i < 9; i++ {
		blocks = append(blocks, block)
	}
	details := codexItemDetails(map[string]any{"type": "mcpToolCall", "server": "fixture", "tool": "view", "result": map[string]any{"content": blocks}})
	if len(arr(obj(details["result"])["content"])) != 8 || details["truncated"] != true {
		t.Fatal("image count was not bounded")
	}
}
