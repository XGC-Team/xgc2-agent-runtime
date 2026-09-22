//go:build unix

package agentruntime

import (
	"context"
	"os"
	"syscall"
	"time"
)

func lockSettings(ctx context.Context, f *os.File) error {
	ticker := time.NewTicker(20 * time.Millisecond)
	defer ticker.Stop()
	for {
		err := syscall.Flock(int(f.Fd()), syscall.LOCK_EX|syscall.LOCK_NB)
		if err == nil {
			return nil
		}
		if err != syscall.EWOULDBLOCK && err != syscall.EAGAIN {
			return err
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
		}
	}
}
func unlockSettings(f *os.File) { _ = syscall.Flock(int(f.Fd()), syscall.LOCK_UN) }
