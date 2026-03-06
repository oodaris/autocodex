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

func TestResolveHarnessConfigPathFallsBackToRepoRootExampleFromNestedDirectory(t *testing.T) {
	root := t.TempDir()
	writeHarnessFixture(t, root)
	if err := os.Remove(filepath.Join(root, "autocodex.yaml")); err != nil {
		t.Fatalf("remove autocodex.yaml: %v", err)
	}
	writeFile(t, filepath.Join(root, "config.example.yaml"), "version: v1\nmode: yolo\n")

	nested := filepath.Join(root, "nested", "deeper")
	if err := os.MkdirAll(nested, 0o755); err != nil {
		t.Fatalf("mkdir nested: %v", err)
	}

	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("get wd: %v", err)
	}
	if err := os.Chdir(nested); err != nil {
		t.Fatalf("chdir nested: %v", err)
	}
	defer func() { _ = os.Chdir(cwd) }()

	resolved := resolveHarnessConfigPath(config.ResolveConfigPath(), false)
	expected, err := filepath.Abs(filepath.Join(root, "config.example.yaml"))
	if err != nil {
		t.Fatalf("abs expected fallback path: %v", err)
	}
	if canonicalTestPath(t, resolved) != canonicalTestPath(t, expected) {
		t.Fatalf("expected repo-root config.example fallback %q, got %q", expected, resolved)
	}

	cfg, err := config.Load(resolved)
	if err != nil {
		t.Fatalf("load fallback config: %v", err)
	}
	result := runHarnessLint(cfg, resolved)
	if result.Status != "ok" {
		t.Fatalf("expected harness lint to pass with fallback config, got %s (%s)", result.Status, result.Details)
	}
}

func TestResolveHarnessConfigPathPrefersRepoRootAutocodexOverExample(t *testing.T) {
	root := t.TempDir()
	writeHarnessFixture(t, root)
	writeFile(t, filepath.Join(root, "config.example.yaml"), "version: v1\nmode: yolo\n")

	nested := filepath.Join(root, "nested", "deeper")
	if err := os.MkdirAll(nested, 0o755); err != nil {
		t.Fatalf("mkdir nested: %v", err)
	}

	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("get wd: %v", err)
	}
	if err := os.Chdir(nested); err != nil {
		t.Fatalf("chdir nested: %v", err)
	}
	defer func() { _ = os.Chdir(cwd) }()

	resolved := resolveHarnessConfigPath(config.ResolveConfigPath(), false)
	expected, err := filepath.Abs(filepath.Join(root, "autocodex.yaml"))
	if err != nil {
		t.Fatalf("abs expected autocodex path: %v", err)
	}
	if canonicalTestPath(t, resolved) != canonicalTestPath(t, expected) {
		t.Fatalf("expected repo-root autocodex.yaml %q, got %q", expected, resolved)
	}
}

func TestResolveHarnessConfigPathHonorsAutocodexConfigEnv(t *testing.T) {
	t.Setenv("AUTOCODEX_CONFIG", "custom.yaml")
	resolved := resolveHarnessConfigPath(config.ResolveConfigPath(), false)
	if resolved != "custom.yaml" {
		t.Fatalf("expected AUTOCODEX_CONFIG to win, got %q", resolved)
	}
}

func TestResolveHarnessConfigPathHonorsExplicitConfigFlag(t *testing.T) {
	resolved := resolveHarnessConfigPath("missing.yaml", true)
	if resolved != "missing.yaml" {
		t.Fatalf("expected explicit config flag to win, got %q", resolved)
	}
}

func TestResolveHarnessConfigPathDoesNotUseNestedExampleConfig(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, ".git"), "gitdir: .git\n")
	nested := filepath.Join(root, "nested", "deeper")
	if err := os.MkdirAll(nested, 0o755); err != nil {
		t.Fatalf("mkdir nested: %v", err)
	}
	writeFile(t, filepath.Join(nested, "config.example.yaml"), "version: v1\nmode: yolo\n")

	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("get wd: %v", err)
	}
	if err := os.Chdir(nested); err != nil {
		t.Fatalf("chdir nested: %v", err)
	}
	defer func() { _ = os.Chdir(cwd) }()

	resolved := resolveHarnessConfigPath(config.ResolveConfigPath(), false)
	if resolved != config.DefaultConfigFile {
		t.Fatalf("expected nested config.example.yaml to be ignored, got %q", resolved)
	}
}

func canonicalTestPath(t *testing.T, path string) string {
	t.Helper()
	canonical, err := filepath.EvalSymlinks(path)
	if err != nil {
		return filepath.Clean(path)
	}
	return canonical
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
