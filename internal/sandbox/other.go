//go:build !linux && !windows && !darwin

package sandbox

import (
	"fmt"
	"os"
)

func platform(Options) Sandbox { return nil }

func Exec(string) int    { fmt.Fprintln(os.Stderr, "sandbox-exec runs on Linux"); return 125 }
func Init(string) int    { return Exec("") }
func Inner([]string) int { return Exec("") }
