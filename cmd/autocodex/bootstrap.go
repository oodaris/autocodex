package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/oodaris/autocodex/internal/config"
	"github.com/oodaris/autocodex/internal/state"
)

func runBootstrap(args []string) {
	fs := flag.NewFlagSet("bootstrap", flag.ExitOnError)
	configPath := fs.String("config", config.ResolveConfigPath(), "Config file path")
	profile := fs.String("profile", "", "bootstrap profile: max_capability|balanced|max_throughput (default: profile from config or max_capability)")
	force := fs.Bool("force", false, "overwrite existing templates, schemas, and skills")
	initGit := fs.Bool("init-git", true, "initialize a git repo if missing")
	initBD := fs.Bool("init-bd", true, "initialize beads if missing")
	fs.Parse(args)

	if err := bootstrapRepo(*configPath, strings.TrimSpace(*profile), *force, *initGit, *initBD); err != nil {
		exitErr(err)
	}
	fmt.Printf("Bootstrap complete. Config: %s\n", *configPath)
}

func bootstrapRepo(configPath string, requestedProfile string, force bool, initGit bool, initBD bool) error {
	_, configExistsErr := os.Stat(configPath)
	configExists := configExistsErr == nil
	if configExistsErr != nil && !os.IsNotExist(configExistsErr) {
		return configExistsErr
	}
	if err := ensureConfig(configPath); err != nil {
		return err
	}
	cfg, err := config.Load(configPath)
	if err != nil {
		return err
	}
	selectedProfile, explicitProfile, err := resolveBootstrapProfile(requestedProfile, cfg.Profile)
	if err != nil {
		return err
	}
	if explicitProfile || !configExists {
		if err := applyBootstrapProfile(&cfg, selectedProfile); err != nil {
			return err
		}
		if err := writeBootstrapConfig(configPath, cfg); err != nil {
			return err
		}
	}

	if err := ensureRepoPrereqs(cfg, initGit, initBD); err != nil {
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
