//go:build !windows

package proc

import (
	"os/exec"
	"syscall"
)

func detach(cmd *exec.Cmd) { cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true} }

func startCmd(cmd *exec.Cmd) error { return cmd.Start() }

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
