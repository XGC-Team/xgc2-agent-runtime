package managedfiles

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"golang.org/x/sys/unix"
)

func grant(t *testing.T) Grant {
	return Grant{Owner: "fixture", Category: "scratch", Schema: "scratch.v1", Scope: "user/workspace", Path: t.TempDir(), MaxBytes: 32, MaxFileBytes: 16, MaxEntries: 16, MaxLeases: 2, Trigger: "fixture", Cleanup: "close", Recovery: "report orphans"}
}
func TestGrantAdmissionAndWritersAreBoundedBeforePublication(t *testing.T) {
	g := grant(t)
	r, err := Open(g)
	if err != nil {
		t.Fatal(err)
	}
	defer r.Close()
	l, err := r.TempDir(context.Background(), "intake", 16, 4)
	if err != nil {
		t.Fatal(err)
	}
	w, err := l.Create("source.pdf", 16)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := l.Create("other.pdf", 1); !errors.Is(err, ErrLimit) {
		t.Fatalf("concurrent reserved writer: %v", err)
	}
	if _, err := w.Write(make([]byte, 17)); !errors.Is(err, ErrLimit) {
		t.Fatalf("oversized write: %v", err)
	}
	info, err := os.Stat(filepath.Join(l.Path(), "source.pdf"))
	if err != nil || info.Size() != 0 {
		t.Fatalf("partial oversize write: %v %v", info, err)
	}
	if err := l.Close(); err == nil {
		t.Fatal("released active writer")
	}
	if _, err := w.Write(make([]byte, 16)); err != nil {
		t.Fatal(err)
	}
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	if err := l.Check(context.Background()); err != nil {
		t.Fatal(err)
	}
	if err := l.Close(); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(l.Path()); !os.IsNotExist(err) {
		t.Fatalf("lease retained: %v", err)
	}
}
func TestGrantRejectsMissingRootSymlinksAndCapacityWithoutFallback(t *testing.T) {
	g := grant(t)
	missing := filepath.Join(g.Path, "missing")
	g.Path = missing
	if _, err := Open(g); !os.IsNotExist(err) {
		t.Fatalf("missing root: %v", err)
	}
	if _, err := os.Stat(missing); !os.IsNotExist(err) {
		t.Fatal("consumer created root")
	}
	g = grant(t)
	link := filepath.Join(t.TempDir(), "alias")
	if err := os.Symlink(g.Path, link); err != nil {
		t.Fatal(err)
	}
	g.Path = link
	if _, err := Open(g); err == nil {
		t.Fatal("symlink root accepted")
	}
	g = grant(t)
	r, err := Open(g)
	if err != nil {
		t.Fatal(err)
	}
	defer r.Close()
	l, err := r.TempDir(context.Background(), "parse", 24, 4)
	if err != nil {
		t.Fatal(err)
	}
	defer l.Close()
	if _, err := r.TempDir(context.Background(), "parse", 1<<62, 4); !errors.Is(err, ErrLimit) {
		t.Fatalf("oversized reservation: %v", err)
	}
	if _, err := r.TempDir(context.Background(), "parse", 9, 4); !errors.Is(err, ErrLimit) {
		t.Fatalf("reservation overflow: %v", err)
	}
	if err := os.Symlink(t.TempDir(), filepath.Join(l.Path(), "escape")); err != nil {
		t.Fatal(err)
	}
	if err := l.Check(context.Background()); !errors.Is(err, ErrGrant) {
		t.Fatalf("symlink output: %v", err)
	}
}
func TestInterruptedFilesAreCountedAndRetained(t *testing.T) {
	g := grant(t)
	orphan := filepath.Join(g.Path, "interrupted.pdf")
	if err := os.WriteFile(orphan, make([]byte, 16), 0600); err != nil {
		t.Fatal(err)
	}
	r, err := Open(g)
	if err != nil {
		t.Fatal(err)
	}
	defer r.Close()
	if _, err := r.TempDir(context.Background(), "intake", 17, 4); !errors.Is(err, ErrLimit) {
		t.Fatalf("orphan ignored: %v", err)
	}
	if info, err := os.Stat(orphan); err != nil || info.Size() != 16 {
		t.Fatal("orphan deleted")
	}
}

func TestGrantedSymlinkMetadataNeverReadsOrRemovesTargets(t *testing.T) {
	g := grant(t)
	g.SymlinkMetadata, g.MaxBytes, g.MaxFileBytes = true, 4096, 1024
	outside := t.TempDir()
	sentinel := filepath.Join(outside, "retained")
	payload := []byte(strings.Repeat("outside", 1024))
	if err := os.WriteFile(sentinel, payload, 0600); err != nil {
		t.Fatal(err)
	}
	// Following either directory link would exceed the declared byte limit;
	// the cycle and dangling link have no traversable content at all.
	links := map[string]string{"file": sentinel, "directory": outside, "cycle": ".", "dangling": "missing"}
	var targetBytes int64
	for name, target := range links {
		if err := os.Symlink(target, filepath.Join(g.Path, name)); err != nil {
			t.Fatal(err)
		}
		targetBytes += int64(len(target))
	}
	r, err := Open(g)
	if err != nil {
		t.Fatal(err)
	}
	defer r.Close()
	used, count, err := inventory(context.Background(), r.root, g.MaxBytes, g.MaxFileBytes, g.MaxEntries, true, nil)
	if err != nil || used != targetBytes || count != len(links) {
		t.Fatalf("link metadata accounting: bytes=%d entries=%d error=%v", used, count, err)
	}
	l, err := r.TempDir(context.Background(), "native", 512, 4)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(sentinel, filepath.Join(l.Path(), "file")); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, filepath.Join(l.Path(), "directory")); err != nil {
		t.Fatal(err)
	}
	if err := l.Check(context.Background()); err != nil {
		t.Fatal(err)
	}
	if writer, err := l.Create("file", 16); err == nil {
		_ = writer.Close()
		t.Fatal("bounded writer followed an existing link")
	}
	if err := l.Close(); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(sentinel)
	if err != nil || string(data) != string(payload) {
		t.Fatal("lease cleanup followed a link or modified an outside target")
	}
	alias := filepath.Join(t.TempDir(), "root-alias")
	if err := os.Symlink(g.Path, alias); err != nil {
		t.Fatal(err)
	}
	g.Path = alias
	if _, err := Open(g); err == nil {
		t.Fatal("metadata permission admitted a linked root path")
	}
}

func TestSymlinkMetadataRetainsEveryFiniteAdmissionLimit(t *testing.T) {
	for _, test := range []struct {
		name  string
		links []string
		limit func(*Grant)
	}{
		{"single target bytes", []string{strings.Repeat("x", 17)}, func(g *Grant) {}},
		{"total target bytes", []string{strings.Repeat("x", 20), strings.Repeat("y", 20)}, func(g *Grant) { g.MaxFileBytes = 32 }},
		{"link entries", []string{"x", "y"}, func(g *Grant) { g.MaxEntries = 1 }},
	} {
		t.Run(test.name, func(t *testing.T) {
			g := grant(t)
			g.SymlinkMetadata = true
			test.limit(&g)
			for index, target := range test.links {
				if err := os.Symlink(target, filepath.Join(g.Path, strings.Repeat("link", index+1))); err != nil {
					t.Fatal(err)
				}
			}
			if _, err := Open(g); !errors.Is(err, ErrLimit) {
				t.Fatalf("metadata bypassed capacity: %v", err)
			}
		})
	}
	g := grant(t)
	g.SymlinkMetadata = true
	if err := unix.Mkfifo(filepath.Join(g.Path, "pipe"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := Open(g); !errors.Is(err, ErrGrant) {
		t.Fatalf("metadata permission admitted a special file: %v", err)
	}
}
