//go:build !windows

package testutil

import "os/exec"

func ExecCommand(name string, arg ...string) *exec.Cmd {
	return exec.Command(name, arg...)
}

func ScriptExt() string {
	return ".sh"
}
