package main

import "github.com/oodaris/autocodex/internal/config"

func applyCodexOverrides(cfg *config.Config, collaborationMode, preset string) {
	if cfg == nil {
		return
	}
	if collaborationMode != "" {
		cfg.Codex.CollaborationMode = collaborationMode
	}
	if preset != "" {
		cfg.Codex.Preset = preset
	}
}

func disableCollaboration(cfg *config.Config) {
	if cfg == nil {
		return
	}
	disabled := false
	cfg.Codex.CollaborationOn = &disabled
	cfg.Codex.CollaborationMode = ""
	cfg.Codex.Preset = ""
}
