package terminal

import (
	"context"
	"os"
	"runtime"
	"testing"
	"time"
)

func TestTerminalSessionLifecycle(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("pty not supported on windows")
	}
	manager := NewManager(nil)
	shell := os.Getenv("SHELL")
	if shell == "" {
		shell = "sh"
	}
	session, err := manager.Create(context.Background(), SessionConfig{
		Command: shell,
		Args:    []string{"-c", "sleep 0.2"},
	})
	if err != nil {
		t.Fatalf("create session: %v", err)
	}
	if session.PTY() == nil {
		t.Fatalf("expected pty")
	}
	if !session.TryAttach() {
		t.Fatalf("expected attach to succeed")
	}
	session.Detach()

	if err := manager.Close(session.ID); err != nil {
		t.Fatalf("close session: %v", err)
	}
	time.Sleep(50 * time.Millisecond)

	summary := session.Summary()
	if summary.Status != StatusClosed {
		// allow a short window for process to exit
		time.Sleep(100 * time.Millisecond)
		summary = session.Summary()
	}
	if summary.Status != StatusClosed {
		t.Fatalf("expected closed status")
	}
}
