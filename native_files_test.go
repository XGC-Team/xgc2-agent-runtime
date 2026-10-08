package agentruntime

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"github.com/XGC-Team/xgc2-agent-runtime/managedfiles"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func testNativeFiles(t *testing.T) *NativeFiles {
	t.Helper()
	open := func(category string) *managedfiles.Root {
		r, err := managedfiles.Open(managedfiles.Grant{Owner: "protocol-fixture", Category: category, Schema: "native-files.v1", Scope: "fixture-user/workspace", Path: t.TempDir(), MaxBytes: 256 << 20, MaxFileBytes: 64 << 20, MaxEntries: 4096, MaxLeases: 16, Trigger: "isolated fixture launch", Cleanup: "lease close after native exit", Recovery: "keep interrupted work until explicit cleanup"})
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() {
			if err := r.Close(); err != nil {
				t.Error(err)
			}
		})
		return r
	}
	states := map[string]*managedfiles.Root{}
	for _, provider := range providerKinds {
		states[provider] = open("native-" + provider)
	}
	f, err := NewNativeFiles(open("native-scratch"), states)
	if err != nil {
		t.Fatal(err)
	}
	return f
}
func testNativeContext(t *testing.T) context.Context {
	return WithNativeFiles(context.Background(), testNativeFiles(t))
}

func TestNativeProbeRequiresFilesAndUsesOnlyGrantedHomeAndScratch(t *testing.T) {
	ambient := t.TempDir()
	t.Setenv("HOME", ambient)
	t.Setenv("TMPDIR", ambient)
	path := filepath.Join(t.TempDir(), "probe")
	program := []byte("#!/bin/sh\nprintf '%s\\n%s\\n%s\\n' \"$HOME\" \"$TMPDIR\" \"$PWD\"\nprintf native > \"$HOME/resume-data\"\nprintf scratch > \"$TMPDIR/temporary\"\n")
	if err := os.WriteFile(path, program, 0700); err != nil {
		t.Fatal(err)
	}
	digest := sha256.Sum256(program)
	p := Profile{Provider: "codex", Executable: path, SHA256: hex.EncodeToString(digest[:])}
	if _, err := cliOutput(context.Background(), p, "--version"); !errors.Is(err, managedfiles.ErrGrant) {
		t.Fatalf("missing grant: %v", err)
	}
	f := testNativeFiles(t)
	data, err := cliOutput(WithNativeFiles(context.Background(), f), p, "--version")
	if err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(strings.TrimSpace(string(data)), "\n")
	if len(lines) != 3 || lines[0] != f.states["codex"].Path() || lines[1] != lines[2] || !strings.HasPrefix(lines[1], f.scratch.Path()+"/") {
		t.Fatalf("native files escaped grants: %s", data)
	}
	if _, err := os.Stat(filepath.Join(lines[0], "resume-data")); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(lines[1]); !os.IsNotExist(err) {
		t.Fatalf("scratch retained: %v", err)
	}
	entries, err := os.ReadDir(ambient)
	if err != nil || len(entries) != 0 {
		t.Fatalf("ambient writes: %v %v", entries, err)
	}
}
