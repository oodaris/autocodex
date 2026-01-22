package main

import (
	"testing"

	"github.com/oodaris/autocodex/internal/config"
)

func TestApplySwarmOverridesEnabled(t *testing.T) {
	maxParallel := 3
	cfg := &config.Config{
		Autonomy: config.AutonomyConfig{
			Enabled: false,
			Coordinator: config.AutonomyCoordinatorConfig{
				Enabled:     false,
				Strategy:    "phase",
				MaxParallel: &maxParallel,
			},
		},
	}

	applySwarmOverrides(cfg, true)

	if !cfg.Autonomy.Enabled {
		t.Fatalf("expected autonomy enabled")
	}
	if !cfg.Autonomy.Coordinator.Enabled {
		t.Fatalf("expected coordinator enabled")
	}
	if cfg.Autonomy.Coordinator.Strategy != "bead" {
		t.Fatalf("expected strategy bead, got %q", cfg.Autonomy.Coordinator.Strategy)
	}
	if cfg.Autonomy.Coordinator.MaxParallel == nil {
		t.Fatalf("expected max_parallel to be set")
	}
	if got := *cfg.Autonomy.Coordinator.MaxParallel; got != 0 {
		t.Fatalf("expected max_parallel 0 (unlimited), got %d", got)
	}
}

func TestApplySwarmOverridesDisabled(t *testing.T) {
	maxParallel := 4
	cfg := &config.Config{
		Autonomy: config.AutonomyConfig{
			Enabled: true,
			Coordinator: config.AutonomyCoordinatorConfig{
				Enabled:     true,
				Strategy:    "bead",
				MaxParallel: &maxParallel,
			},
		},
	}

	applySwarmOverrides(cfg, false)

	if !cfg.Autonomy.Enabled {
		t.Fatalf("expected autonomy enabled")
	}
	if !cfg.Autonomy.Coordinator.Enabled {
		t.Fatalf("expected coordinator enabled")
	}
	if got := *cfg.Autonomy.Coordinator.MaxParallel; got != 4 {
		t.Fatalf("expected max_parallel unchanged, got %d", got)
	}
}
