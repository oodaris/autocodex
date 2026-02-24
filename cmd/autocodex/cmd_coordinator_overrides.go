package main

import (
	"strings"

	"github.com/oodaris/autocodex/internal/config"
)

func applyCoordinatorOverrides(cfg *config.Config, beadScope string, allowAllReadyFallback bool, beadIDsCSV string, beadPrefix string) {
	if cfg == nil {
		return
	}
	beadScope = strings.TrimSpace(beadScope)
	if beadScope != "" {
		cfg.Autonomy.Coordinator.SelectionMode = beadScope
	}
	if allowAllReadyFallback {
		cfg.Autonomy.Coordinator.AllowAllReadyFallback = true
	}
	cfg.Autonomy.Coordinator.BeadIDs = parseCommaList(beadIDsCSV)
	cfg.Autonomy.Coordinator.BeadPrefix = strings.TrimSpace(beadPrefix)
}

func parseCommaList(input string) []string {
	if strings.TrimSpace(input) == "" {
		return []string{}
	}
	parts := strings.Split(input, ",")
	items := make([]string, 0, len(parts))
	for _, part := range parts {
		item := strings.TrimSpace(part)
		if item == "" {
			continue
		}
		items = append(items, item)
	}
	return items
}
