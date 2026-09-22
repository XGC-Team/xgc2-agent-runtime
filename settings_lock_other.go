//go:build !unix

package nativeagent

import (
	"context"
	"os"
)

func lockSettings(ctx context.Context, f *os.File) error { return ErrUnavailable }
func unlockSettings(f *os.File)                          {}
