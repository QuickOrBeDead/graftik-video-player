package command

import (
	"context"
	"testing"
)

func TestCreateHiddenCmd_NonNil(t *testing.T) {
	cmd := CreateHiddenCmd("echo")
	if cmd == nil {
		t.Fatal("expected non-nil *exec.Cmd")
	}
}

func TestCreateHiddenCmdContext_NonNil(t *testing.T) {
	ctx := context.Background()
	cmd := CreateHiddenCmdContext(ctx, "echo")
	if cmd == nil {
		t.Fatal("expected non-nil *exec.Cmd")
	}
}

func TestCreateHiddenCmd_Path(t *testing.T) {
	cmd := CreateHiddenCmd("echo", "hello")
	if cmd.Path == "" {
		t.Fatal("expected non-empty Path")
	}
}

func TestCreateHiddenCmdContext_Path(t *testing.T) {
	ctx := context.Background()
	cmd := CreateHiddenCmdContext(ctx, "echo", "hello")
	if cmd.Path == "" {
		t.Fatal("expected non-empty Path")
	}
}
