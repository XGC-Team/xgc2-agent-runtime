package agentruntime

import (
	"context"
	"errors"
	"github.com/XGC-Team/xgc2-agent-runtime/managedfiles"
)

// NativeFiles contains host-owned capabilities. Native state includes the
// provider's login, resume data and diagnostics; it is not Broker authority.
// Its writable mount and capacity are granted separately from scratch.
type NativeFiles struct {
	scratch *managedfiles.Root
	states  map[string]*managedfiles.Root
}

func NewNativeFiles(scratch *managedfiles.Root, states map[string]*managedfiles.Root) (*NativeFiles, error) {
	if scratch == nil || len(states) > len(providerKinds) {
		return nil, managedfiles.ErrGrant
	}
	f := &NativeFiles{scratch: scratch, states: map[string]*managedfiles.Root{}}
	paths := map[string]bool{scratch.Path(): true}
	for provider, root := range states {
		if root == nil {
			return nil, managedfiles.ErrGrant
		}
		if root.Grant().Owner != scratch.Grant().Owner || root.Grant().Scope != scratch.Grant().Scope {
			return nil, managedfiles.ErrGrant
		}
		if _, err := commandArgs(provider); err != nil || paths[root.Path()] {
			return nil, managedfiles.ErrGrant
		}
		paths[root.Path()] = true
		f.states[provider] = root
	}
	return f, nil
}

type nativeFilesKey struct{}

// WithNativeFiles binds explicit capabilities for a direct Driver consumer.
func WithNativeFiles(ctx context.Context, files *NativeFiles) context.Context {
	return context.WithValue(ctx, nativeFilesKey{}, files)
}
func (b *Broker) fileContext(ctx context.Context) context.Context {
	b.mu.Lock()
	files, _ := b.ctx.Value(nativeFilesKey{}).(*NativeFiles)
	b.mu.Unlock()
	return WithNativeFiles(ctx, files)
}
func nativeFiles(ctx context.Context) (*NativeFiles, error) {
	f, _ := ctx.Value(nativeFilesKey{}).(*NativeFiles)
	if f == nil {
		return nil, managedfiles.ErrGrant
	}
	return f, nil
}
func nativeScratch(ctx context.Context, purpose string, bytes int64, entries int) (*managedfiles.Lease, error) {
	f, err := nativeFiles(ctx)
	if err != nil {
		return nil, err
	}
	return f.scratch.TempDir(ctx, purpose, bytes, entries)
}
func nativeLaunch(ctx context.Context, p Profile) ([]string, *managedfiles.Lease, func() error, error) {
	f, err := nativeFiles(ctx)
	if err != nil {
		return nil, nil, nil, err
	}
	state := f.states[p.Provider]
	if state == nil {
		return nil, nil, nil, managedfiles.ErrGrant
	}
	if err = state.Check(ctx); err != nil {
		return nil, nil, nil, err
	}
	lease, err := f.scratch.TempDir(ctx, "native-temp", 32<<20, 256)
	if err != nil {
		return nil, nil, nil, err
	}
	home, temp := state.Path(), lease.Path()
	env := []string{"HOME=" + home, "USERPROFILE=" + home, "APPDATA=" + home, "LOCALAPPDATA=" + home, "XDG_CONFIG_HOME=" + home, "XDG_DATA_HOME=" + home, "XDG_STATE_HOME=" + home, "XDG_CACHE_HOME=" + temp, "TMPDIR=" + temp, "TMP=" + temp, "TEMP=" + temp}
	if p.Provider == "codex" {
		env = append(env, "CODEX_HOME="+home)
	}
	if p.Provider == "claude" {
		env = append(env, "CLAUDE_CONFIG_DIR="+home)
	}
	cleanup := func() error {
		// Release admission only after native exit; retain vendor state for resume.
		a := lease.Check(context.Background())
		b := state.Check(context.Background())
		c := lease.Close()
		return errors.Join(a, b, c)
	}
	return env, lease, cleanup, nil
}
