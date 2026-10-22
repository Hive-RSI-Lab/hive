//go:build !windows

package proc

import (
	"os/exec"
	"syscall"
)

var detachAttempts = []uint32{0}

func detach(cmd *exec.Cmd, _ uint32) { cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true} }

func running(pid int) bool {
	err := syscall.Kill(pid, 0)
	return err == nil || err == syscall.EPERM
}

// Terminate sends SIGTERM to the process group.
func Terminate(h Handle) error {
	if !Alive(h) {
		return nil
	}
	return syscall.Kill(-h.PID, syscall.SIGTERM)
}

// Kill sends SIGKILL to the process group.
func Kill(h Handle) error {
	if !Alive(h) {
		return nil
	}
	return syscall.Kill(-h.PID, syscall.SIGKILL)
}
