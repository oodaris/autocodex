package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/oodaris/autocodex/internal/config"
)

func TestDoctorChecksConfigMissing(t *testing.T) {
	result := checkConfigFile(filepath.Join(t.TempDir(), "missing.yaml"))
	if result.Status != "error" {
		t.Fatalf("expected error status, got %s", result.Status)
	}
}

func TestDoctorChecksConfigValid(t *testing.T) {
	tmp := t.TempDir()
	path := filepath.Join(tmp, "config.yaml")
	if err := os.WriteFile(path, []byte("version: v1\nmode: yolo\n"), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}
	result := checkConfigFile(path)
	if result.Status != "ok" {
		t.Fatalf("expected ok status, got %s", result.Status)
	}
}

func TestDoctorConfigValidation(t *testing.T) {
	cfg := config.Config{Version: "v1", Mode: "yolo"}
	cfg.ApplyDefaults()
	result := checkConfigValidation(cfg)
	if result.Status != "ok" {
		t.Fatalf("expected ok validation, got %s", result.Status)
	}

	bad := config.Config{Version: "v1", Mode: "bad"}
	bad.ApplyDefaults()
	result = checkConfigValidation(bad)
	if result.Status != "error" {
		t.Fatalf("expected error validation, got %s", result.Status)
	}
}

func TestDoctorCommandOptional(t *testing.T) {
	result := checkCommandOptional("definitely-not-a-command")
	if result.Status != "warn" {
		t.Fatalf("expected warn status, got %s", result.Status)
	}
}

func TestDoctorMemoryDirMissing(t *testing.T) {
	cfg := config.Config{Version: "v1", Mode: "yolo"}
	cfg.ApplyDefaults()
	cfg.Paths.MemoryDir = filepath.Join(t.TempDir(), "missing")
	result := checkMemoryDir(cfg, "")
	if result.Status != "warn" {
		t.Fatalf("expected warn status, got %s", result.Status)
	}
	if !strings.Contains(result.Details, "missing") {
		t.Fatalf("expected missing details, got %s", result.Details)
	}
}

func TestParseFirstSemver(t *testing.T) {
	v, ok := parseFirstSemver("codex-cli 0.101.0")
	if !ok {
		t.Fatalf("expected to parse version")
	}
	if v.Major != 0 || v.Minor != 101 || v.Patch != 0 {
		t.Fatalf("unexpected version: %+v", v)
	}

	v, ok = parseFirstSemver("0.102.0-alpha.7")
	if !ok {
		t.Fatalf("expected to parse prerelease version")
	}
	if v.Major != 0 || v.Minor != 102 || v.Patch != 0 {
		t.Fatalf("unexpected version: %+v", v)
	}
}

func TestSemverLess(t *testing.T) {
	if !(semver{Major: 0, Minor: 92, Patch: 9}).Less(semver{Major: 0, Minor: 93, Patch: 0}) {
		t.Fatalf("expected 0.92.9 < 0.93.0")
	}
	if (semver{Major: 0, Minor: 93, Patch: 0}).Less(semver{Major: 0, Minor: 93, Patch: 0}) {
		t.Fatalf("expected 0.93.0 !< 0.93.0")
	}
	if (semver{Major: 0, Minor: 93, Patch: 1}).Less(semver{Major: 0, Minor: 93, Patch: 0}) {
		t.Fatalf("expected 0.93.1 !< 0.93.0")
	}
}

func TestParseCodexFeatureList(t *testing.T) {
	raw := `shell_tool stable true
unified_exec stable true
multi_agent experimental false
`
	features := parseCodexFeatureList(raw)
	if !features["shell_tool"] {
		t.Fatalf("expected shell_tool=true")
	}
	if !features["unified_exec"] {
		t.Fatalf("expected unified_exec=true")
	}
	if features["multi_agent"] {
		t.Fatalf("expected multi_agent=false")
	}
}

func TestAssessBDVersionOutput(t *testing.T) {
	status, details := assessBDVersionOutput("bd version 0.55.4 (dev)")
	if status != "warn" {
		t.Fatalf("expected warn for old bd version, got %s", status)
	}
	if !strings.Contains(details, "requires >=") {
		t.Fatalf("expected minimum-version guidance, got %q", details)
	}

	status, details = assessBDVersionOutput("bd version 0.56.1 (48bfaaad)")
	if status != "ok" {
		t.Fatalf("expected ok for target bd version, got %s", status)
	}
	if !strings.Contains(details, "target >=") {
		t.Fatalf("expected target-version detail, got %q", details)
	}

	status, _ = assessBDVersionOutput("")
	if status != "warn" {
		t.Fatalf("expected warn for empty output, got %s", status)
	}
}

func TestAssessBDDoltShowOutput(t *testing.T) {
	embeddedJSON := `{
  "backend": "dolt",
  "database": "beads",
  "host": "127.0.0.1",
  "mode": "embedded",
  "port": 3307,
  "user": "root"
}`
	status, details := assessBDDoltShowOutput(embeddedJSON)
	if status != "ok" {
		t.Fatalf("expected ok for embedded json output, got %s", status)
	}
	if !strings.Contains(details, "mode=embedded") || !strings.Contains(details, "embedded mode") {
		t.Fatalf("unexpected embedded details: %q", details)
	}

	serverJSONReachable := `{
  "backend": "dolt",
  "database": "beads",
  "host": "127.0.0.1",
  "port": 3307,
  "connection_ok": true
}`
	status, details = assessBDDoltShowOutput(serverJSONReachable)
	if status != "ok" {
		t.Fatalf("expected ok for connection_ok=true json output, got %s", status)
	}
	if !strings.Contains(details, "server reachable") {
		t.Fatalf("unexpected reachable connection details: %q", details)
	}

	serverJSONUnreachable := `{
  "backend": "dolt",
  "database": "beads",
  "host": "127.0.0.1",
  "port": 3307,
  "connection_ok": false
}`
	status, details = assessBDDoltShowOutput(serverJSONUnreachable)
	if status != "warn" {
		t.Fatalf("expected warn for unreachable server json output, got %s", status)
	}
	if !strings.Contains(details, "server not reachable") {
		t.Fatalf("unexpected server details: %q", details)
	}

	serverJSONUnknown := `{
  "backend": "dolt",
  "database": "beads",
  "mode": "server",
  "host": "127.0.0.1",
  "port": 3307
}`
	status, details = assessBDDoltShowOutput(serverJSONUnknown)
	if status != "warn" {
		t.Fatalf("expected warn for server mode with unknown reachability, got %s", status)
	}
	if !strings.Contains(details, "reachability unknown") {
		t.Fatalf("unexpected server unknown details: %q", details)
	}

	reachable := `Dolt Configuration
==================
  Mode:     server
  Database: beads
  Host:     127.0.0.1
  Port:     3307

  ✓ Server reachable`
	status, details = assessBDDoltShowOutput(reachable)
	if status != "ok" {
		t.Fatalf("expected ok for reachable output, got %s", status)
	}
	if !strings.Contains(details, "database=beads") || !strings.Contains(details, "mode=server") || !strings.Contains(details, "server reachable") {
		t.Fatalf("unexpected reachable details: %q", details)
	}

	unreachable := `Dolt Configuration
==================
  Mode:     server
  Database: beads
  Host:     127.0.0.1
  Port:     3307

  ✗ Server not reachable`
	status, details = assessBDDoltShowOutput(unreachable)
	if status != "warn" {
		t.Fatalf("expected warn for unreachable output, got %s", status)
	}
	if !strings.Contains(details, "server not reachable") {
		t.Fatalf("unexpected unreachable details: %q", details)
	}

	serverUnknown := `Dolt Configuration
==================
  Mode:     server
  Database: beads
  Host:     127.0.0.1
  Port:     3307`
	status, details = assessBDDoltShowOutput(serverUnknown)
	if status != "warn" {
		t.Fatalf("expected warn for text server mode with unknown reachability, got %s", status)
	}
	if !strings.Contains(details, "reachability unknown") {
		t.Fatalf("unexpected unknown reachability details: %q", details)
	}

	embedded := `Dolt Configuration
==================
  Mode:     embedded
  Database: beads`
	status, details = assessBDDoltShowOutput(embedded)
	if status != "ok" {
		t.Fatalf("expected ok for embedded text output, got %s", status)
	}
	if !strings.Contains(details, "embedded mode") {
		t.Fatalf("unexpected embedded text details: %q", details)
	}
}

func TestRunDoctorChecksUsesConfigPathForRepoSignals(t *testing.T) {
	root := t.TempDir()
	writeHarnessFixture(t, root)

	nested := filepath.Join(root, "docs", "nested")
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

	cfg := config.Config{Version: "v1", Mode: "yolo"}
	cfg.ApplyDefaults()
	results := runDoctorChecks(cfg, filepath.Join("..", "..", "autocodex.yaml"))

	if result := doctorCheckByName(t, results, "git"); result.Status != "ok" {
		t.Fatalf("expected git check to use config-derived repo root, got %s (%s)", result.Status, result.Details)
	}
	if result := doctorCheckByName(t, results, "memory"); result.Status != "warn" || !strings.Contains(result.Details, ".autocodex/memory") {
		t.Fatalf("expected memory check to resolve against repo root, got %s (%s)", result.Status, result.Details)
	}
}

func doctorCheckByName(t *testing.T, results []checkResult, name string) checkResult {
	t.Helper()
	for _, result := range results {
		if result.Name == name {
			return result
		}
	}
	t.Fatalf("missing doctor check %q", name)
	return checkResult{}
}
