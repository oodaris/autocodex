package main

import (
	"os"
	"testing"
)

func TestIsNonBlockingDoctorWarning(t *testing.T) {
	if !isNonBlockingDoctorWarning("memory") {
		t.Fatalf("expected memory warning to be non-blocking")
	}
	if isNonBlockingDoctorWarning("codex") {
		t.Fatalf("expected codex warning to be blocking")
	}
}

func TestRunHarnessLintMissingScript(t *testing.T) {
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("get wd: %v", err)
	}
	tmp := t.TempDir()
	if err := os.Chdir(tmp); err != nil {
		t.Fatalf("chdir tmp: %v", err)
	}
	defer func() { _ = os.Chdir(cwd) }()

	result := runHarnessLint()
	if result.Status != "error" {
		t.Fatalf("expected lint error when script missing, got %s", result.Status)
	}
}
