//go:build !unix

package agentruntime

import "os/exec"

// Native subprocess supervision is currently accepted only on Unix hosts.
const supportedNativeHost = false

func isolateProcess(c *exec.Cmd) {}
func terminateProcess(c *exec.Cmd, force bool) error {
	if c.Process == nil {
		return nil
	}
	return c.Process.Kill()
}
