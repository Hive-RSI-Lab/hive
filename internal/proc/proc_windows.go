//go:build windows

package proc

import (
	"os/exec"
	"strconv"
	"syscall"
	"time"

	"golang.org/x/sys/windows"
)

const (
	createNewProcessGroup  = 0x00000200
	createNoWindow         = 0x08000000
	createBreakawayFromJob = 0x01000000
)

func detach(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{CreationFlags: createNewProcessGroup | createNoWindow | createBreakawayFromJob, HideWindow: true}
}

func startCmd(cmd *exec.Cmd) error {
	err := cmd.Start()
	if err != nil && cmd.SysProcAttr != nil && cmd.SysProcAttr.CreationFlags&createBreakawayFromJob != 0 {
		// The job we run in may forbid breakaway; start without it.
		cmd.SysProcAttr.CreationFlags &^= createBreakawayFromJob
		cmd.Process = nil
		err = cmd.Start()
	}
	return err
}

func running(pid int) bool {
	h, err := windows.OpenProcess(windows.PROCESS_QUERY_LIMITED_INFORMATION, false, uint32(pid))
	if err != nil {
		return false
	}
	defer windows.CloseHandle(h)
	var code uint32
	if windows.GetExitCodeProcess(h, &code) != nil {
		return false
	}
	return code == 259 // STILL_ACTIVE
}

// StartTime returns the process creation time, or zero.
func StartTime(pid int) time.Time {
	h, err := windows.OpenProcess(windows.PROCESS_QUERY_LIMITED_INFORMATION, false, uint32(pid))
	if err != nil {
		return time.Time{}
	}
	defer windows.CloseHandle(h)
	var c, e, k, u windows.Filetime
	if windows.GetProcessTimes(h, &c, &e, &k, &u) != nil {
		return time.Time{}
	}
	return time.Unix(0, c.Nanoseconds()).UTC()
}

// Kill stops the process and its children.
func Kill(h Handle) error {
	if !Alive(h) {
		return nil
	}
	return exec.Command("taskkill", "/T", "/F", "/PID", strconv.Itoa(h.PID)).Run()
}

// Terminate asks politely; on Windows there is no SIGTERM for console-less
// processes, so it is the same as Kill.
func Terminate(h Handle) error { return Kill(h) }
