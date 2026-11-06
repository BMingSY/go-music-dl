//go:build windows

package playback

import (
	"os/exec"
	"syscall"
)

func quietProcess(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true, CreationFlags: 0x08000000}
}
