//go:build windows

package testutil

import "os/exec"

func ExecCommand(name string, arg ...string) *exec.Cmd {
	return exec.Command("powershell", "-ExecutionPolicy", "Bypass", "-File", name)
}

func ScriptExt() string {
	return ".ps1"
}
