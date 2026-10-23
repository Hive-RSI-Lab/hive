//go:build !windows

package proc

import (
	"syscall"
	"testing"
	"time"
)

// A detached process leads its own session, so whatever kills the starter's
// process group on exit (launchd, a terminal) leaves it running: the daemon's
// handover child depends on it.
func TestStartDetachedLeadsItsOwnSession(t *testing.T) {
	h, err := StartDetached([]string{"sleep", "5"}, nil, "", "", "")
	if err != nil {
		t.Fatal(err)
	}
	defer Kill(h)
	time.Sleep(100 * time.Millisecond)
	if sid, err := syscall.Getsid(h.PID); err != nil || sid != h.PID {
		t.Fatalf("session %d, %v: want %d", sid, err, h.PID)
	}
	if pg, _ := syscall.Getpgid(h.PID); pg == syscall.Getpgrp() {
		t.Fatalf("shares the test's process group %d", pg)
	}
}
