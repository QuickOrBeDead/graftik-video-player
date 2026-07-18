//go:build !windows

package command

import (
	"context"
	"os/exec"
)

func CreateHiddenCmd(name string, args ...string) *exec.Cmd {
	// Unix systems don't have a "console window" to hide,
	// so no special SysProcAttr is needed.
	return exec.Command(name, args...)
}

func CreateHiddenCmdContext(ctx context.Context, name string, args ...string) *exec.Cmd {
	return exec.CommandContext(ctx, name, args...)
}
