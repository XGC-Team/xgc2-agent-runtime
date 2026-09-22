//go:build !unix

package agentruntime

import (
	"context"
	"os"
)

func lockSettings(ctx context.Context, f *os.File) error { return ErrUnavailable }
func unlockSettings(f *os.File)                          {}
