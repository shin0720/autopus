//go:build windows

package cli

import (
	"os/exec"
	"syscall"
)

// hideConsoleWindow prevents any console window from appearing for a subprocess.
// CREATE_NO_WINDOW blocks console window creation at the OS level, which also
// covers child processes spawned by batch scripts (.cmd/.bat).
func hideConsoleWindow(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{
		HideWindow:    true,
		CreationFlags: 0x08000000, // CREATE_NO_WINDOW
	}
}
