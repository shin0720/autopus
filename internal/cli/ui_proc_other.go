//go:build !windows

package cli

import "os/exec"

func hideConsoleWindow(cmd *exec.Cmd) {}
