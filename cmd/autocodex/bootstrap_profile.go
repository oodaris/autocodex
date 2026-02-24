package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/oodaris/autocodex/internal/config"
	"gopkg.in/yaml.v3"
)

const (
	profileMaxCapability = "max_capability"
	profileBalanced      = "balanced"
	profileMaxThroughput = "max_throughput"
)

func resolveBootstrapProfile(flagValue string, configValue string) (string, bool, error) {
	explicit := strings.TrimSpace(flagValue) != ""
	profile := strings.ToLower(strings.TrimSpace(flagValue))
	if profile == "" {
		profile = strings.ToLower(strings.TrimSpace(configValue))
	}
	if profile == "" {
		profile = profileMaxCapability
	}
	if !isSupportedBootstrapProfile(profile) {
		return "", explicit, fmt.Errorf("unsupported profile %q (expected: %s|%s|%s)", profile, profileMaxCapability, profileBalanced, profileMaxThroughput)
	}
	return profile, explicit, nil
}

func isSupportedBootstrapProfile(profile string) bool {
	switch strings.ToLower(strings.TrimSpace(profile)) {
	case profileMaxCapability, profileBalanced, profileMaxThroughput:
		return true
	default:
		return false
	}
}

func applyBootstrapProfile(cfg *config.Config, profile string) error {
	if cfg == nil {
		return fmt.Errorf("config is nil")
	}
	profile = strings.ToLower(strings.TrimSpace(profile))
	if !isSupportedBootstrapProfile(profile) {
		return fmt.Errorf("unsupported profile %q", profile)
	}

	cfg.Profile = profile
	cfg.Autonomy.Enabled = true
	cfg.Autonomy.Coordinator.Enabled = true
	cfg.Autonomy.Coordinator.Strategy = "bead"
	cfg.Autonomy.Harness.Enabled = true
	cfg.Autonomy.Harness.ImpactMode = "normal"

	switch profile {
	case profileMaxCapability:
		cfg.Codex.ReasoningEffort = "xhigh"
		setCollaborationEnabled(cfg, true)
		setCoordinatorMaxParallel(cfg, 4)
		cfg.Autonomy.StopConditions.MaxFixAttempts = 3
	case profileBalanced:
		cfg.Codex.ReasoningEffort = "high"
		setCollaborationEnabled(cfg, true)
		setCoordinatorMaxParallel(cfg, 2)
		cfg.Autonomy.StopConditions.MaxFixAttempts = 3
	case profileMaxThroughput:
		cfg.Codex.ReasoningEffort = "medium"
		setCollaborationEnabled(cfg, false)
		setCoordinatorMaxParallel(cfg, 2)
		cfg.Autonomy.StopConditions.MaxFixAttempts = 2
	}
	return nil
}

func setCollaborationEnabled(cfg *config.Config, enabled bool) {
	if cfg == nil {
		return
	}
	cfg.Codex.CollaborationOn = boolPtr(enabled)
	if enabled {
		cfg.Codex.CollaborationMode = "auto"
		cfg.Codex.Preset = "default"
		return
	}
	cfg.Codex.CollaborationMode = ""
	cfg.Codex.Preset = ""
}

func setCoordinatorMaxParallel(cfg *config.Config, value int) {
	if cfg == nil {
		return
	}
	parallel := value
	cfg.Autonomy.Coordinator.MaxParallel = &parallel
}

func boolPtr(value bool) *bool {
	v := value
	return &v
}

func writeBootstrapConfig(path string, cfg config.Config) error {
	data, err := yaml.Marshal(&cfg)
	if err != nil {
		return fmt.Errorf("marshal config: %w", err)
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return fmt.Errorf("write config %s: %w", path, err)
	}
	return nil
}
