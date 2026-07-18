//go:build !windows

package command

import (
	"context"
	"testing"
	"time"
)

func TestCreateHiddenCmd_Unix_SysProcAttrIsNil(t *testing.T) {
	cmd := CreateHiddenCmd("echo")
	if cmd.SysProcAttr != nil {
		t.Fatal("expected SysProcAttr to be nil on Unix")
	}
}

func TestCreateHiddenCmdContext_Unix_SysProcAttrIsNil(t *testing.T) {
	ctx := context.Background()
	cmd := CreateHiddenCmdContext(ctx, "echo")
	if cmd.SysProcAttr != nil {
		t.Fatal("expected SysProcAttr to be nil on Unix")
	}
}

func TestCreateHiddenCmd_Unix_Runs(t *testing.T) {
	cmd := CreateHiddenCmd("echo", "hello")
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if string(out) != "hello\n" {
		t.Fatalf("expected 'hello\\n', got %q", out)
	}
}

func TestCreateHiddenCmdContext_Unix_Cancel(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	cmd := CreateHiddenCmdContext(ctx, "sleep", "10")
	err := cmd.Run()
	if err == nil {
		t.Fatal("expected error after cancelled context")
	}
}

func TestCreateHiddenCmdContext_Unix_Timeout(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	cmd := CreateHiddenCmdContext(ctx, "sleep", "10")
	err := cmd.Run()
	if err == nil {
		t.Fatal("expected error after timeout")
	}
}
