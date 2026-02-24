package main

import (
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/oodaris/autocodex/internal/config"
)

func TestIsNonBlockingDoctorWarning(t *testing.T) {
	if !isNonBlockingDoctorWarning("memory") {
		t.Fatalf("expected memory warning to be non-blocking")
	}
	if isNonBlockingDoctorWarning("codex") {
		t.Fatalf("expected codex warning to be blocking")
	}
	if isNonBlockingDoctorWarning("bd-version") {
		t.Fatalf("expected bd-version warning to be blocking")
	}
	if isNonBlockingDoctorWarning("bd-dolt") {
		t.Fatalf("expected bd-dolt warning to be blocking")
	}
}

func TestRunHarnessLintMissingRolePackConfig(t *testing.T) {
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("get wd: %v", err)
	}
	tmp := t.TempDir()
	if err := os.Chdir(tmp); err != nil {
		t.Fatalf("chdir tmp: %v", err)
	}
	defer func() { _ = os.Chdir(cwd) }()

	cfg := config.Config{Version: "v1", Mode: "yolo"}
	cfg.ApplyDefaults()
	result := runHarnessLint(cfg, filepath.Join(tmp, "autocodex.yaml"))
	if result.Status != "error" {
		t.Fatalf("expected lint error when role pack is missing, got %s", result.Status)
	}
	if !strings.Contains(result.Details, ".codex/config.toml") {
		t.Fatalf("expected missing role pack config in details, got: %s", result.Details)
	}
}

func TestIsHarnessHelpArg(t *testing.T) {
	if !isHarnessHelpArg("-h") || !isHarnessHelpArg("--help") {
		t.Fatalf("expected harness help args to be true")
	}
	if isHarnessHelpArg("help") {
		t.Fatalf("expected bare help to be false for help-arg helper")
	}
}

func TestRunHarnessUsageOnHelp(t *testing.T) {
	out := captureStdout(t, func() {
		runHarness([]string{"--help"})
	})
	if !strings.Contains(out, "Usage: autocodex harness <subcommand> [args]") {
		t.Fatalf("expected harness usage output")
	}
	if !strings.Contains(out, "preflight, lint") {
		t.Fatalf("expected lint subcommand in usage output")
	}
}

func TestHarnessPreflightJSONOutputIsParseable(t *testing.T) {
	checks := []harnessCheck{
		{Name: "doctor.config", Status: "ok", Details: "validated"},
		{Name: "harness.lint", Status: "ok", Details: "ok"},
	}
	out := captureStdout(t, func() {
		if err := printHarnessPreflightChecks(checks, true); err != nil {
			t.Fatalf("print checks: %v", err)
		}
	})
	decoder := json.NewDecoder(strings.NewReader(out))
	var parsed []harnessCheck
	if err := decoder.Decode(&parsed); err != nil {
		t.Fatalf("expected parseable json output, got error: %v", err)
	}
	if len(parsed) != 2 {
		t.Fatalf("expected two checks, got %d", len(parsed))
	}
	var tail any
	if err := decoder.Decode(&tail); err != io.EOF {
		t.Fatalf("expected no trailing output after json, got: %v", err)
	}
}

func TestHarnessLintJSONOutputIsParseable(t *testing.T) {
	checks := []harnessCheck{
		{Name: "harness.lint", Status: "ok", Details: "lint ok"},
	}
	out := captureStdout(t, func() {
		if err := printHarnessPreflightChecks(checks, true); err != nil {
			t.Fatalf("print checks: %v", err)
		}
	})
	decoder := json.NewDecoder(strings.NewReader(out))
	var parsed []harnessCheck
	if err := decoder.Decode(&parsed); err != nil {
		t.Fatalf("expected parseable json output, got error: %v", err)
	}
	if len(parsed) != 1 || parsed[0].Name != "harness.lint" {
		t.Fatalf("unexpected parsed checks: %#v", parsed)
	}
	var tail any
	if err := decoder.Decode(&tail); err != io.EOF {
		t.Fatalf("expected no trailing output after json, got: %v", err)
	}
}
