//go:build windows

package hls

import "os/exec"

func execCommand(name string, arg ...string) *exec.Cmd {
	return exec.Command("powershell", "-ExecutionPolicy", "Bypass", "-File", name)
}

func scriptExt() string {
	return ".ps1"
}