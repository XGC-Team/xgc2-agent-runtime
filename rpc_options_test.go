package nativeagent

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"testing"
	"time"
)

func TestACPWireUsesNativeNumericIDsAndExplicitEmptyParams(t *testing.T) {
	inputReader, inputWriter := io.Pipe()
	outputReader, outputWriter := io.Pipe()
	p := newPeer(inputWriter, outputReader, true, func(string, map[string]any) {}, nil)
	defer p.Close()
	defer inputReader.Close()
	defer outputReader.Close()
	defer outputWriter.Close()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := p.Call(ctx, "session/new", map[string]any{}); !errors.Is(err, context.Canceled) {
		t.Fatalf("cancelled request was written: %v", err)
	}
	captured := make(chan map[string]any, 1)
	go func() {
		var got map[string]any
		if json.NewDecoder(inputReader).Decode(&got) == nil {
			captured <- got
			_ = json.NewEncoder(outputWriter).Encode(map[string]any{"jsonrpc": "2.0", "id": got["id"], "result": map[string]any{}})
		}
	}()
	ctx, cancel = context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if _, err := p.Call(ctx, "cursor/list_available_models", map[string]any{}); err != nil {
		t.Fatal(err)
	}
	got := <-captured
	if got["id"] != float64(1) || got["jsonrpc"] != "2.0" || got["params"] == nil {
		t.Fatalf("native wire drift: %+v", got)
	}
}
