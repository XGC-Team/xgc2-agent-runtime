package managedfiles

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
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
