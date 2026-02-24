package main

import (
	"testing"

	"github.com/oodaris/autocodex/internal/config"
)

func TestApplyCoordinatorOverrides(t *testing.T) {
	cfg := config.Config{Version: "v1", Mode: "yolo"}
	cfg.ApplyDefaults()

	applyCoordinatorOverrides(&cfg, "all_ready", true, "a-1, a-2", "a-")

	if cfg.Autonomy.Coordinator.SelectionMode != "all_ready" {
		t.Fatalf("expected selection mode override, got %q", cfg.Autonomy.Coordinator.SelectionMode)
	}
	if !cfg.Autonomy.Coordinator.AllowAllReadyFallback {
		t.Fatalf("expected allow_all_ready_fallback override")
	}
	if len(cfg.Autonomy.Coordinator.BeadIDs) != 2 {
		t.Fatalf("expected bead ids override, got %#v", cfg.Autonomy.Coordinator.BeadIDs)
	}
	if cfg.Autonomy.Coordinator.BeadPrefix != "a-" {
		t.Fatalf("expected bead prefix override, got %q", cfg.Autonomy.Coordinator.BeadPrefix)
	}
}

func TestParseCommaList(t *testing.T) {
	items := parseCommaList(" a , , b ,, c ")
	if len(items) != 3 {
		t.Fatalf("expected 3 items, got %d", len(items))
	}
}
