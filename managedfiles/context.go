package managedfiles

import "context"

type scratchKey struct{}

// WithScratch borrows the host's temporary file capability. The host closes it
// only after consumers have stopped; attaching a context grants no OS access.
func WithScratch(ctx context.Context, root *Root) context.Context {
	return context.WithValue(ctx, scratchKey{}, root)
}
func Scratch(ctx context.Context) (*Root, error) {
	r, _ := ctx.Value(scratchKey{}).(*Root)
	if r == nil {
		return nil, ErrGrant
	}
	return r, nil
}
func TempDir(ctx context.Context, purpose string, bytes int64, entries int) (*Lease, error) {
	r, err := Scratch(ctx)
	if err != nil {
		return nil, err
	}
	return r.TempDir(ctx, purpose, bytes, entries)
}

// CheckScratch verifies currently admitted native output before publication.
func CheckScratch(ctx context.Context, path string) error {
	r, err := Scratch(ctx)
	if err != nil {
		return err
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, lease := range r.active {
		if lease.Path() == path {
			return lease.Check(ctx)
		}
	}
	return ErrGrant
}
