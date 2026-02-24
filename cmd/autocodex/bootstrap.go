package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/oodaris/autocodex/internal/config"
	"github.com/oodaris/autocodex/internal/state"
)

var (
	bootstrapRepoRunner        = bootstrapRepo
	bootstrapReadyChecksRunner = runBootstrapReadyChecks
	bootstrapSmokeTaskRunner   = runBootstrapSmokeTask
)

func runBootstrap(args []string) {
	fs := flag.NewFlagSet("bootstrap", flag.ExitOnError)
	configPath := fs.String("config", config.ResolveConfigPath(), "Config file path")
	profile := fs.String("profile", "", "bootstrap profile: max_capability|balanced|max_throughput (default: profile from config or max_capability)")
	ready := fs.Bool("ready", false, "run strict readiness checks after bootstrap (harness preflight + harness lint)")
	smokeTask := fs.String("smoke-task", "", "optional task to run after bootstrap using current config/profile")
	force := fs.Bool("force", false, "overwrite existing templates, schemas, and skills")
	initGit := fs.Bool("init-git", true, "initialize a git repo if missing")
	initBD := fs.Bool("init-bd", true, "initialize beads if missing")
	fs.Parse(args)

	if err := runBootstrapWorkflow(
		*configPath,
		strings.TrimSpace(*profile),
		*force,
		*initGit,
		*initBD,
		*ready,
		strings.TrimSpace(*smokeTask),
	); err != nil {
		exitErr(err)
	}
	fmt.Printf("Bootstrap complete. Config: %s\n", *configPath)
}

func runBootstrapWorkflow(
	configPath string,
	requestedProfile string,
	force bool,
	initGit bool,
	initBD bool,
	ready bool,
	smokeTask string,
) error {
	if err := bootstrapRepoRunner(configPath, requestedProfile, force, initGit, initBD); err != nil {
		return err
	}
	if ready {
		if err := bootstrapReadyChecksRunner(configPath); err != nil {
			return err
		}
	}
	if strings.TrimSpace(smokeTask) != "" {
		if err := bootstrapSmokeTaskRunner(configPath, smokeTask); err != nil {
			return err
		}
	}
	return nil
}

func runBootstrapReadyChecks(configPath string) error {
	fmt.Println("Running bootstrap readiness checks...")

	cfg, err := config.Load(configPath)
	if err != nil {
		return err
	}

	checks, hasFailure := runHarnessPreflightChecks(cfg, configPath, true)
	if err := printHarnessPreflightChecks(checks, false); err != nil {
		return err
	}
	if hasFailure {
		return fmt.Errorf("harness preflight found issues")
	}

	// Keep an explicit standalone lint pass so bootstrap --ready mirrors the
	// documented sequence and captures a separate lint result.
	lintResult := runHarnessLint(cfg)
	lintResult.Name = "harness.lint-standalone"
	if err := printHarnessPreflightChecks([]harnessCheck{lintResult}, false); err != nil {
		return err
	}
	if lintResult.Status == "error" {
		return fmt.Errorf("harness lint failed: %s", lintResult.Details)
	}

	fmt.Println("Bootstrap readiness checks passed.")
	return nil
}

func runBootstrapSmokeTask(configPath, task string) error {
	task = strings.TrimSpace(task)
	if task == "" {
		return nil
	}

	fmt.Printf("Running bootstrap smoke task: %s\n", task)

	cfg, err := config.Load(configPath)
	if err != nil {
		return err
	}

	taskPayload, err := applyTaskInput(&cfg, task)
	if err != nil {
		return err
	}
	if err := runLoopWithTask(cfg, taskPayload); err != nil {
		return err
	}

	fmt.Println("Bootstrap smoke task completed.")
	return nil
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
