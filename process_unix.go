//go:build unix

package agentruntime

import (
	"os/exec"
	"syscall"
)

const supportedNativeHost = true

func isolateProcess(c *exec.Cmd) { c.SysProcAttr = &syscall.SysProcAttr{Setpgid: true} }
func terminateProcess(c *exec.Cmd, force bool) error {
	if c.Process == nil {
		return nil
	}
	signal := syscall.SIGTERM
	if force {
		signal = syscall.SIGKILL
	}
	return syscall.Kill(-c.Process.Pid, signal)
}
