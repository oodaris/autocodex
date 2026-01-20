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
	result := checkMemoryDir(cfg)
	if result.Status != "warn" {
		t.Fatalf("expected warn status, got %s", result.Status)
	}
	if !strings.Contains(result.Details, "missing") {
		t.Fatalf("expected missing details, got %s", result.Details)
	}
}
