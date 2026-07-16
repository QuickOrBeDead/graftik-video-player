//go:build !windows

package hls

import "os/exec"

func execCommand(name string, arg ...string) *exec.Cmd {
	return exec.Command(name, arg...)
}

func scriptExt() string {
	return ".sh"
}