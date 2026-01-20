package main

import (
	"strings"
	"testing"

	"github.com/oodaris/autocodex/internal/state"
)

func TestResumeMessage(t *testing.T) {
	tests := []struct {
		name     string
		run      state.Run
		task     string
		force    bool
		wantErr  bool
		contains string
	}{
		{
			name:     "running without force blocks",
			run:      state.Run{ID: "run1", Status: "running"},
			force:    false,
			wantErr:  true,
			contains: "still running",
		},
		{
			name:     "running with force allows",
			run:      state.Run{ID: "run2", Status: "running"},
			force:    true,
			wantErr:  false,
			contains: "still running",
		},
		{
			name:     "completed without task blocks",
			run:      state.Run{ID: "run3", Status: "completed"},
			force:    false,
			wantErr:  true,
			contains: "provide --task",
		},
		{
			name:     "completed without task and force allows",
			run:      state.Run{ID: "run4", Status: "completed"},
			force:    true,
			wantErr:  false,
			contains: "resuming without a new task",
		},
		{
			name:     "completed with task allows",
			run:      state.Run{ID: "run5", Status: "completed"},
			task:     "do more",
			force:    false,
			wantErr:  false,
			contains: "starting a new run",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			msg, err := resumeMessage(tt.run, tt.task, tt.force)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error")
				}
				if !strings.Contains(err.Error(), tt.contains) {
					t.Fatalf("expected error to contain %q, got %q", tt.contains, err.Error())
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if !strings.Contains(msg, tt.contains) {
				t.Fatalf("expected message to contain %q, got %q", tt.contains, msg)
			}
		})
	}
}
