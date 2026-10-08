// Package managedfiles admits finite file work under an explicit host grant.
// Native subprocess writes also require a quota-limited writable mount; this
// package does not claim to sandbox an executable or replace filesystem quotas.
package managedfiles

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"

	"golang.org/x/sys/unix"
)

var ErrLimit = errors.New("managed file grant capacity exceeded")
var ErrGrant = errors.New("explicit managed file grant required")
var namePattern = regexp.MustCompile(`^[a-z][a-z0-9-]{0,63}$`)

// Grant is immutable deployment configuration, never user transcript data.
// Path must exist before consumers start. Scope identifies the user/workspace;
// the host owns directory allocation, mount isolation, cleanup and recovery.
type Grant struct {
	Owner        string `json:"owner"`
	Category     string `json:"category"`
	Schema       string `json:"schema"`
	Scope        string `json:"scope"`
	Path         string `json:"path"`
	MaxBytes     int64  `json:"max_bytes"`
	MaxFileBytes int64  `json:"max_file_bytes"`
	MaxEntries   int    `json:"max_entries"`
	MaxLeases    int    `json:"max_leases"`
	Trigger      string `json:"trigger"`
	Cleanup      string `json:"cleanup"`
	Recovery     string `json:"recovery"`
}
type Root struct {
	grant  Grant
	root   *os.Root
	mu     sync.Mutex
	active map[string]*Lease
	closed bool
}
type Lease struct {
	owner    *Root
	root     *os.Root
	name     string
	bytes    int64
	entries  int
	mu       sync.Mutex
	closed   bool
	writers  int
	reserved int64
}

func Open(g Grant) (*Root, error) {
	if len(g.Owner) > 256 || len(g.Scope) > 512 || len(g.Schema) > 256 || len(g.Path) > 4096 || len(g.Trigger) > 1024 || len(g.Cleanup) > 1024 || len(g.Recovery) > 1024 {
		return nil, ErrGrant
	}
	if g.Owner == "" || !namePattern.MatchString(g.Category) || g.Schema == "" || g.Scope == "" || g.Trigger == "" || g.Cleanup == "" || g.Recovery == "" || !filepath.IsAbs(g.Path) || filepath.Clean(g.Path) != g.Path || g.Path == "/" || g.MaxBytes <= 0 || g.MaxBytes > 1<<40 || g.MaxFileBytes <= 0 || g.MaxFileBytes > g.MaxBytes || g.MaxEntries < 1 || g.MaxEntries > 100000 || g.MaxLeases < 1 || g.MaxLeases > 128 {
		return nil, ErrGrant
	}
	root, err := openRoot(g.Path)
	if err != nil {
		return nil, err
	}
	r := &Root{grant: g, root: root, active: map[string]*Lease{}}
	if _, _, err = inventory(context.Background(), root, g.MaxBytes, g.MaxFileBytes, g.MaxEntries, nil); err != nil {
		root.Close()
		return nil, err
	}
	return r, nil
}
func (r *Root) Grant() Grant { return r.grant }
func (r *Root) Path() string { return r.grant.Path }
func (r *Root) Check(ctx context.Context) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.closed {
		return os.ErrClosed
	}
	_, _, err := inventory(ctx, r.root, r.grant.MaxBytes, r.grant.MaxFileBytes, r.grant.MaxEntries, nil)
	return err
}
func (r *Root) Close() error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.closed {
		return nil
	}
	if len(r.active) != 0 {
		return errors.New("managed file work is still active")
	}
	r.closed = true
	return r.root.Close()
}

// TempDir reserves the complete maximum output before creating anything. It
// never waits for capacity and never chooses another directory on exhaustion.
func (r *Root) TempDir(ctx context.Context, prefix string, bytes int64, entries int) (*Lease, error) {
	if r == nil {
		return nil, ErrGrant
	}
	if !namePattern.MatchString(prefix) || bytes < 1 || entries < 1 {
		return nil, ErrGrant
	}
	if bytes > r.grant.MaxBytes || entries > r.grant.MaxEntries {
		return nil, ErrLimit
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.closed {
		return nil, os.ErrClosed
	}
	if len(r.active) >= r.grant.MaxLeases {
		return nil, ErrLimit
	}
	skip := map[string]bool{}
	reservedBytes, reservedEntries := bytes, entries+1
	for name, lease := range r.active {
		skip[name] = true
		reservedBytes += lease.bytes
		reservedEntries += lease.entries + 1
	}
	used, count, err := inventory(ctx, r.root, r.grant.MaxBytes, r.grant.MaxFileBytes, r.grant.MaxEntries, skip)
	if err != nil {
		return nil, err
	}
	if used > r.grant.MaxBytes-reservedBytes || count > r.grant.MaxEntries-reservedEntries {
		return nil, ErrLimit
	}
	var id [8]byte
	if _, err = rand.Read(id[:]); err != nil {
		return nil, err
	}
	name := prefix + "-" + hex.EncodeToString(id[:])
	if err = r.root.Mkdir(name, 0700); err != nil {
		return nil, err
	}
	child, err := r.root.OpenRoot(name)
	if err != nil {
		_ = r.root.Remove(name)
		return nil, err
	}
	lease := &Lease{owner: r, root: child, name: name, bytes: bytes, entries: entries}
	r.active[name] = lease
	return lease, nil
}
func (l *Lease) Path() string { return filepath.Join(l.owner.grant.Path, l.name) }
func (l *Lease) Check(ctx context.Context) error {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.closed {
		return os.ErrClosed
	}
	_, _, err := inventory(ctx, l.root, l.bytes, l.owner.grant.MaxFileBytes, l.entries, nil)
	return err
}

// Create admits one direct child and bounds writes before bytes reach disk.
func (l *Lease) Create(name string, limit int64) (*File, error) {
	if name == "" || filepath.Base(name) != name || name == "." || name == ".." || strings.ContainsAny(name, "\\\x00") || limit < 1 || limit > l.bytes || limit > l.owner.grant.MaxFileBytes {
		return nil, ErrGrant
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.closed {
		return nil, os.ErrClosed
	}
	used, count, err := inventory(context.Background(), l.root, l.bytes, l.owner.grant.MaxFileBytes, l.entries, nil)
	if err != nil {
		return nil, err
	}
	if count >= l.entries || used > l.bytes-limit-l.reserved {
		return nil, ErrLimit
	}
	f, err := l.root.OpenFile(name, os.O_CREATE|os.O_EXCL|os.O_RDWR|unix.O_NOFOLLOW, 0600)
	if err != nil {
		return nil, err
	}
	l.writers++
	l.reserved += limit
	return &File{file: f, name: filepath.Join(l.Path(), name), remaining: limit, limit: limit, lease: l}, nil
}
func (l *Lease) WriteFile(name string, data []byte) error {
	limit := int64(len(data))
	if limit == 0 {
		limit = 1
	}
	w, err := l.Create(name, limit)
	if err != nil {
		return err
	}
	_, err = w.Write(data)
	closeErr := w.Close()
	if err != nil {
		return err
	}
	return closeErr
}
func (l *Lease) Close() error {
	l.owner.mu.Lock()
	defer l.owner.mu.Unlock()
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.closed {
		return nil
	}
	if l.writers != 0 {
		return errors.New("managed file writers are still active")
	}
	if err := l.owner.root.RemoveAll(l.name); err != nil {
		return err
	}
	if err := l.root.Close(); err != nil {
		return err
	}
	l.closed = true
	delete(l.owner.active, l.name)
	return nil
}

type File struct {
	file      *os.File
	name      string
	remaining int64
	limit     int64
	lease     *Lease
	mu        sync.Mutex
	closed    bool
}

func (w *File) Write(p []byte) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.closed {
		return 0, os.ErrClosed
	}
	if int64(len(p)) > w.remaining {
		return 0, ErrLimit
	}
	position, err := w.file.Seek(0, io.SeekCurrent)
	if err != nil {
		return 0, err
	}
	if position > w.limit-int64(len(p)) {
		return 0, ErrLimit
	}
	n, err := w.file.Write(p)
	w.remaining -= int64(n)
	return n, err
}
func (w *File) Close() error {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.closed {
		return nil
	}
	err := w.file.Close()
	w.lease.mu.Lock()
	w.lease.writers--
	w.lease.reserved -= w.limit
	w.lease.mu.Unlock()
	w.closed = true
	return err
}
func (w *File) Read(p []byte) (int, error)                   { return w.file.Read(p) }
func (w *File) ReadAt(p []byte, offset int64) (int, error)   { return w.file.ReadAt(p, offset) }
func (w *File) Seek(offset int64, whence int) (int64, error) { return w.file.Seek(offset, whence) }
func (w *File) Stat() (os.FileInfo, error)                   { return w.file.Stat() }
func (w *File) Sync() error                                  { return w.file.Sync() }
func (w *File) Name() string                                 { return w.name }
func (w *File) Chmod(mode os.FileMode) error {
	if mode != 0600 {
		return ErrGrant
	}
	return w.file.Chmod(mode)
}

func inventory(ctx context.Context, root *os.Root, maxBytes, maxFile int64, maxEntries int, skip map[string]bool) (int64, int, error) {
	var bytes int64
	entries := 0
	paths := []string{"."}
	for len(paths) > 0 {
		if err := ctx.Err(); err != nil {
			return 0, 0, err
		}
		path := paths[len(paths)-1]
		paths = paths[:len(paths)-1]
		dir, err := root.Open(path)
		if err != nil {
			return 0, 0, err
		}
		for {
			batch, readErr := dir.ReadDir(64)
			for _, entry := range batch {
				if err := ctx.Err(); err != nil {
					dir.Close()
					return 0, 0, err
				}
				if path == "." && skip[entry.Name()] {
					continue
				}
				entries++
				if entries > maxEntries {
					dir.Close()
					return 0, 0, ErrLimit
				}
				info, err := entry.Info()
				if err != nil {
					dir.Close()
					return 0, 0, err
				}
				if info.Mode()&os.ModeSymlink != 0 {
					dir.Close()
					return 0, 0, ErrGrant
				}
				name := filepath.Join(path, entry.Name())
				if info.IsDir() {
					paths = append(paths, name)
				} else if info.Mode().IsRegular() {
					if info.Size() > maxFile || bytes > maxBytes-info.Size() {
						dir.Close()
						return 0, 0, ErrLimit
					}
					bytes += info.Size()
				} else {
					dir.Close()
					return 0, 0, ErrGrant
				}
			}
			if readErr == io.EOF {
				break
			}
			if readErr != nil {
				dir.Close()
				return 0, 0, readErr
			}
		}
		if err := dir.Close(); err != nil {
			return 0, 0, err
		}
	}
	return bytes, entries, nil
}

func openRoot(path string) (*os.Root, error) {
	parts := strings.Split(strings.TrimPrefix(path, "/"), "/")
	if len(parts) > 128 {
		return nil, ErrGrant
	}
	fd, err := unix.Open("/", unix.O_RDONLY|unix.O_DIRECTORY|unix.O_CLOEXEC|unix.O_NOFOLLOW, 0)
	if err != nil {
		return nil, err
	}
	for _, part := range parts {
		next, e := unix.Openat(fd, part, unix.O_RDONLY|unix.O_DIRECTORY|unix.O_CLOEXEC|unix.O_NOFOLLOW, 0)
		unix.Close(fd)
		if e != nil {
			return nil, e
		}
		fd = next
	}
	defer unix.Close(fd)
	// The proc reference targets the already verified directory descriptor. The
	// returned os.Root owns a separate descriptor and confines subsequent work.
	return os.OpenRoot("/proc/self/fd/" + strconv.Itoa(fd))
}
