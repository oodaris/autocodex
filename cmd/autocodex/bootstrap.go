package main

import (
	"flag"
	"fmt"

	"github.com/oodaris/autocodex/internal/config"
	"github.com/oodaris/autocodex/internal/state"
)

func runBootstrap(args []string) {
	fs := flag.NewFlagSet("bootstrap", flag.ExitOnError)
	configPath := fs.String("config", config.ResolveConfigPath(), "Config file path")
	force := fs.Bool("force", false, "overwrite existing templates, schemas, and skills")
	fs.Parse(args)

	if err := bootstrapRepo(*configPath, *force); err != nil {
		exitErr(err)
	}
	fmt.Printf("Bootstrap complete. Config: %s\n", *configPath)
}

func bootstrapRepo(configPath string, force bool) error {
	if err := ensureConfig(configPath); err != nil {
		return err
	}
	cfg, err := config.Load(configPath)
	if err != nil {
		return err
	}

	store := state.NewStore(cfg.StateDir(), cfg.RunsDir(), cfg.MemoryDir(), cfg.LogsDir(), cfg.ArtifactsDir())
	if err := store.InitDirs(); err != nil {
		return err
	}
	if err := store.EnsureMemoryDocs(); err != nil {
		return err
	}

	skillsRoot := "skills"
	if err := bootstrapSkills(skillsRoot, force); err != nil {
		return err
	}
	if !skillsPathConfigured(cfg.Skills.Paths, skillsRoot) {
		fmt.Printf("Warning: skills.paths does not include %q; add it to use the bootstrap skill pack.\n", skillsRoot)
	}

	if cfg.Autonomy.Enabled {
		if err := bootstrapAutonomyAssets(cfg, force); err != nil {
			return err
		}
	}
	return nil
}
