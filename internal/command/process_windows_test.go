//go:build windows

package command

import (
	"context"
	"testing"
	"time"
)

func TestCreateHiddenCmd_Windows_HideWindow(t *testing.T) {
	cmd := CreateHiddenCmd("cmd.exe", "/c", "echo", "hello")
	if cmd.SysProcAttr == nil {
		t.Fatal("expected SysProcAttr to be non-nil on Windows")
	}
	if !cmd.SysProcAttr.HideWindow {
		t.Fatal("expected HideWindow to be true")
	}
}

func TestCreateHiddenCmdContext_Windows_HideWindow(t *testing.T) {
	ctx := context.Background()
	cmd := CreateHiddenCmdContext(ctx, "cmd.exe", "/c", "echo", "hello")
	if cmd.SysProcAttr == nil {
		t.Fatal("expected SysProcAttr to be non-nil on Windows")
	}
	if !cmd.SysProcAttr.HideWindow {
		t.Fatal("expected HideWindow to be true")
	}
}

func TestCreateHiddenCmd_Windows_Runs(t *testing.T) {
	cmd := CreateHiddenCmd("cmd.exe", "/c", "echo", "hello")
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if string(out) != "hello\r\n" {
		t.Fatalf("expected 'hello\\r\\n', got %q", out)
	}
}

func TestCreateHiddenCmdContext_Windows_Cancel(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	cmd := CreateHiddenCmdContext(ctx, "cmd.exe", "/c", "timeout", "/t", "10")
	err := cmd.Run()
	if err == nil {
		t.Fatal("expected error after cancelled context")
	}
}

func TestCreateHiddenCmdContext_Windows_Timeout(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	cmd := CreateHiddenCmdContext(ctx, "cmd.exe", "/c", "timeout", "/t", "10")
	err := cmd.Run()
	if err == nil {
		t.Fatal("expected error after timeout")
	}
}
