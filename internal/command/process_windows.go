//go:build windows

package command

import (
	"context"
	"os/exec"
	"syscall"
)

func CreateHiddenCmd(name string, args ...string) *exec.Cmd {
	cmd := exec.Command(name, args...)
	// Windows-specific flag to suppress the console window
	cmd.SysProcAttr = &syscall.SysProcAttr{
		HideWindow: true,
	}
	return cmd
}

func CreateHiddenCmdContext(ctx context.Context, name string, args ...string) *exec.Cmd {
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.SysProcAttr = &syscall.SysProcAttr{
		HideWindow: true,
	}
	return cmd
}
