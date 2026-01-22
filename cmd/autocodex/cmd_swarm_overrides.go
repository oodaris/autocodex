package main

import (
	"fmt"

	"github.com/oodaris/autocodex/internal/config"
)

func applySwarmOverrides(cfg *config.Config, swarm bool) {
	if !swarm {
		return
	}
	if !cfg.Autonomy.Enabled {
		fmt.Println("Warning: --swarm enabled autonomy for this run.")
		cfg.Autonomy.Enabled = true
	}
	cfg.Autonomy.Coordinator.Enabled = true
	cfg.Autonomy.Coordinator.Strategy = "bead"
	unlimited := 0
	cfg.Autonomy.Coordinator.MaxParallel = &unlimited
}
