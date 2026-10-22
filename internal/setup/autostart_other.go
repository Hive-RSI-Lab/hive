//go:build !windows

package setup

func windowsAutostart(string, string, bool) error { return nil }
func windowsAutostartEnabled() bool               { return false }
