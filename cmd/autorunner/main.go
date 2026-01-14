package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/fatih/autocodex/internal/codex"
	"github.com/fatih/autocodex/internal/config"
	"github.com/fatih/autocodex/internal/logging"
	"github.com/fatih/autocodex/internal/orchestrator"
	"github.com/fatih/autocodex/internal/skills"
	"github.com/fatih/autocodex/internal/state"
)

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(1)
	}

	cmd := os.Args[1]
	switch cmd {
	case "init":
		runInit(os.Args[2:])
	case "run":
		runRun(os.Args[2:])
	case "status":
		runStatus(os.Args[2:])
	case "beads":
		runBeads(os.Args[2:])
	case "config":
		runConfig(os.Args[2:])
	default:
		usage()
		os.Exit(1)
	}
}

func usage() {
	fmt.Println("Usage: autorunner <command> [args]")
	fmt.Println("Commands: init, run, status, beads, config")
}

func runInit(args []string) {
	fs := flag.NewFlagSet("init", flag.ExitOnError)
	configPath := fs.String("config", config.ResolveConfigPath(), "Config file path")
	fs.Parse(args)

	if err := ensureConfig(*configPath); err != nil {
		exitErr(err)
	}

	cfg, err := config.Load(*configPath)
	if err != nil {
		exitErr(err)
	}

	store := state.NewStore(cfg.StateDir(), cfg.RunsDir(), cfg.MemoryDir(), cfg.LogsDir(), cfg.ArtifactsDir())
	if err := store.InitDirs(); err != nil {
		exitErr(err)
	}
	if err := store.EnsureMemoryDocs(); err != nil {
		exitErr(err)
	}

	fmt.Printf("Initialized Autocodex. Config: %s\n", *configPath)
}

func runRun(args []string) {
	fs := flag.NewFlagSet("run", flag.ExitOnError)
	configPath := fs.String("config", config.ResolveConfigPath(), "Config file path")
	fs.Parse(args)

	cfg, err := config.Load(*configPath)
	if err != nil {
		exitErr(err)
	}

	logger := logging.NewLogger(cfg.Logging.Level)
	store := state.NewStore(cfg.StateDir(), cfg.RunsDir(), cfg.MemoryDir(), cfg.LogsDir(), cfg.ArtifactsDir())
	loader := skills.Loader{Paths: cfg.Skills.Paths}
	runner := codex.Runner{
		CLIPath:        cfg.Codex.CLIPath,
		Model:          cfg.Codex.Model,
		ExtraArgs:      cfg.Codex.ExtraArgs,
		Mode:           cfg.Mode,
		ApprovalPolicy: cfg.Codex.ApprovalPolicy,
		SandboxMode:    cfg.Codex.SandboxMode,
		Timeout:        time.Duration(cfg.Codex.TimeoutSeconds) * time.Second,
		Env:            cfg.Codex.Env,
	}

	orch := orchestrator.Orchestrator{
		Config: cfg,
		Logger: logger,
		Store:  store,
		Skills: loader,
		Codex:  runner,
	}

	ctx := context.Background()
	if _, err := orch.Run(ctx); err != nil {
		exitErr(err)
	}
}

func runStatus(args []string) {
	fs := flag.NewFlagSet("status", flag.ExitOnError)
	configPath := fs.String("config", config.ResolveConfigPath(), "Config file path")
	fs.Parse(args)

	cfg, err := config.Load(*configPath)
	if err != nil {
		exitErr(err)
	}
	store := state.NewStore(cfg.StateDir(), cfg.RunsDir(), cfg.MemoryDir(), cfg.LogsDir(), cfg.ArtifactsDir())
	runs, err := store.ListRuns()
	if err != nil {
		exitErr(err)
	}
	if len(runs) == 0 {
		fmt.Println("No runs found")
		return
	}
	for _, run := range runs {
		fmt.Printf("%s\t%s\t%s\n", run.ID, run.Status, run.CurrentPhase)
	}
}

func runBeads(args []string) {
	fs := flag.NewFlagSet("beads", flag.ExitOnError)
	action := fs.String("action", "ready", "bd action (ready|show)")
	issue := fs.String("issue", "", "issue id for show")
	fs.Parse(args)

	cmdArgs := []string{*action}
	if *action == "show" && *issue != "" {
		cmdArgs = append(cmdArgs, *issue)
	}

	cmd := exec.Command("bd", cmdArgs...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		exitErr(fmt.Errorf("bd %s failed: %w", *action, err))
	}
}

func runConfig(args []string) {
	fs := flag.NewFlagSet("config", flag.ExitOnError)
	configPath := fs.String("config", config.ResolveConfigPath(), "Config file path")
	fs.Parse(args)
	fmt.Printf("Config path: %s\n", *configPath)
}

func ensureConfig(path string) error {
	if _, err := os.Stat(path); err == nil {
		return nil
	} else if !errors.Is(err, os.ErrNotExist) {
		return err
	}

	examplePath := filepath.Join("config.example.yaml")
	data, err := os.ReadFile(examplePath)
	if err != nil {
		return fmt.Errorf("read config.example.yaml: %w", err)
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return fmt.Errorf("write %s: %w", path, err)
	}
	return nil
}

func exitErr(err error) {
	fmt.Fprintf(os.Stderr, "error: %v\n", err)
	os.Exit(1)
}
