//go:build !windows

package playback

import "os/exec"

func quietProcess(cmd *exec.Cmd) {}
