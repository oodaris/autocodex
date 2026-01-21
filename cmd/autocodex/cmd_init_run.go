package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/oodaris/autocodex/internal/api"
	"github.com/oodaris/autocodex/internal/autonomy"
	"github.com/oodaris/autocodex/internal/codex"
	"github.com/oodaris/autocodex/internal/config"
	"github.com/oodaris/autocodex/internal/logging"
	"github.com/oodaris/autocodex/internal/orchestrator"
	"github.com/oodaris/autocodex/internal/skills"
	"github.com/oodaris/autocodex/internal/state"
)

func runInit(args []string) {
	fs := flag.NewFlagSet("init", flag.ExitOnError)
	configPath := fs.String("config", config.ResolveConfigPath(), "Config file path")
	initGit := fs.Bool("init-git", true, "initialize a git repo if missing")
	initBD := fs.Bool("init-bd", true, "initialize beads if missing")
	fs.Parse(args)

	if err := ensureConfig(*configPath); err != nil {
		exitErr(err)
	}

	cfg, err := config.Load(*configPath)
	if err != nil {
		exitErr(err)
	}

	if err := ensureRepoPrereqs(cfg, *initGit, *initBD); err != nil {
		exitErr(err)
	}

	store := state.NewStore(cfg.StateDir(), cfg.RunsDir(), cfg.MemoryDir(), cfg.LogsDir(), cfg.ArtifactsDir())
	if err := store.InitDirs(); err != nil {
		exitErr(err)
	}
	if err := store.EnsureMemoryDocs(); err != nil {
		exitErr(err)
	}

	fmt.Printf("Initialized autocodex. Config: %s\n", *configPath)
}

func runRun(args []string) {
	fs := flag.NewFlagSet("run", flag.ExitOnError)
	configPath := fs.String("config", config.ResolveConfigPath(), "Config file path")
	task := fs.String("task", "", "task text to append to TODO.md before run")
	taskFile := fs.String("task-file", "", "path to task file (appended to TODO.md before run)")
	taskStdin := fs.Bool("task-stdin", false, "read task text from stdin")
	fs.Parse(args)

	taskPayload, err := resolveTaskInput(*task, *taskFile, *taskStdin, fs.Args(), os.Stdin)
	if err != nil {
		exitErr(err)
	}

	cfg, err := config.Load(*configPath)
	if err != nil {
		exitErr(err)
	}

	taskPayload, err = applyTaskInput(&cfg, taskPayload)
	if err != nil {
		exitErr(err)
	}

	runLoop(cfg, taskPayload)
}

func runOnce(args []string) {
	fs := flag.NewFlagSet("once", flag.ExitOnError)
	configPath := fs.String("config", config.ResolveConfigPath(), "Config file path")
	task := fs.String("task", "", "task text to append to TODO.md before run")
	taskFile := fs.String("task-file", "", "path to task file (appended to TODO.md before run)")
	taskStdin := fs.Bool("task-stdin", false, "read task text from stdin")
	fs.Parse(args)

	taskPayload, err := resolveTaskInput(*task, *taskFile, *taskStdin, fs.Args(), os.Stdin)
	if err != nil {
		exitErr(err)
	}

	cfg, err := config.Load(*configPath)
	if err != nil {
		exitErr(err)
	}
	taskPayload, err = applyTaskInput(&cfg, taskPayload)
	if err != nil {
		exitErr(err)
	}
	cfg.Loop.Mode = "bounded"
	runLoop(cfg, taskPayload)
}

func runLoop(cfg config.Config, taskPayload string) {
	logger := logging.NewLogger(cfg.Logging.Level)
	store := state.NewStore(cfg.StateDir(), cfg.RunsDir(), cfg.MemoryDir(), cfg.LogsDir(), cfg.ArtifactsDir())
	loader := skills.Loader{Paths: cfg.Skills.Paths}
	runner := codex.Runner{
		CLIPath:         cfg.Codex.CLIPath,
		Model:           cfg.Codex.Model,
		ReasoningEffort: cfg.Codex.ReasoningEffort,
		ExtraArgs:       cfg.Codex.ExtraArgs,
		Mode:            cfg.Mode,
		ApprovalPolicy:  cfg.Codex.ApprovalPolicy,
		SandboxMode:     cfg.Codex.SandboxMode,
		JSONOutput:      cfg.Codex.JSONOutput,
		OutputLast:      cfg.Codex.OutputLast,
		PromptStdin:     cfg.Codex.PromptStdin,
		Timeout:         time.Duration(cfg.Codex.TimeoutSeconds) * time.Second,
		IdleTimeout:     time.Duration(cfg.Loop.StopConditions.MaxIdleSeconds) * time.Second,
		Env:             cfg.Codex.Env,
	}

	orch := orchestrator.Orchestrator{
		Config: cfg,
		Logger: logger,
		Store:  store,
		Skills: loader,
		Codex:  runner,
	}

	ctx := context.Background()
	if cfg.Loop.StopConditions.MaxHeartbeatSeconds > 0 {
		watchdog := &api.RunWatchdog{
			Store:               store,
			Logger:              logger,
			MaxHeartbeatSeconds: cfg.Loop.StopConditions.MaxHeartbeatSeconds,
		}
		watchdogCtx, cancelWatchdog := context.WithCancel(context.Background())
		defer cancelWatchdog()
		go watchdog.Start(watchdogCtx)
	}
	if cfg.Autonomy.Enabled {
		controller := autonomy.Controller{
			Config: cfg,
			Logger: logger,
			Store:  store,
			Skills: loader,
			Codex:  runner,
		}
		if _, err := controller.Run(ctx, autonomy.Input{Task: taskPayload}); err != nil {
			exitErr(err)
		}
		return
	}
	if _, err := orch.Run(ctx); err != nil {
		exitErr(err)
	}
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
		if errors.Is(err, os.ErrNotExist) {
			data = embeddedConfigExample
		} else {
			return fmt.Errorf("read config.example.yaml: %w", err)
		}
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return fmt.Errorf("write %s: %w", path, err)
	}
	return nil
}

func resolveInput(input, inputFile string) (json.RawMessage, error) {
	if inputFile != "" {
		data, err := os.ReadFile(inputFile)
		if err != nil {
			return nil, fmt.Errorf("read input file: %w", err)
		}
		return json.RawMessage(data), nil
	}
	if strings.TrimSpace(input) == "" {
		return json.RawMessage([]byte("{}")), nil
	}
	return json.RawMessage([]byte(input)), nil
}

func applyTaskInput(cfg *config.Config, payload string) (string, error) {
	if cfg == nil {
		return "", nil
	}
	if strings.TrimSpace(payload) == "" {
		return "", nil
	}

	store := state.NewStore(cfg.StateDir(), cfg.RunsDir(), cfg.MemoryDir(), cfg.LogsDir(), cfg.ArtifactsDir())
	if err := store.InitDirs(); err != nil {
		return "", err
	}
	if err := store.EnsureMemoryDocs(); err != nil {
		return "", err
	}
	if err := appendTaskToTodo(cfg.MemoryDir(), payload); err != nil {
		return "", err
	}

	if cfg.Loop.Feedback.Mode == "" || cfg.Loop.Feedback.Mode == "off" {
		cfg.Loop.Feedback.Mode = "on"
	}
	fmt.Println("Task appended to TODO.md")
	return payload, nil
}

func appendTaskToTodo(memoryDir, payload string) error {
	if strings.TrimSpace(memoryDir) == "" {
		return fmt.Errorf("memory dir is empty")
	}
	path := filepath.Join(memoryDir, "TODO.md")
	now := time.Now().UTC().Format(time.RFC3339)
	entry := fmt.Sprintf("\n\n## Task (%s)\n%s\n", now, strings.TrimSpace(payload))
	f, err := os.OpenFile(path, os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return fmt.Errorf("open TODO.md: %w", err)
	}
	defer f.Close()
	if _, err := f.WriteString(entry); err != nil {
		return fmt.Errorf("append TODO.md: %w", err)
	}
	return nil
}

func resolveTaskPayload(task, taskFile string) (string, error) {
	if strings.TrimSpace(task) == "" && strings.TrimSpace(taskFile) == "" {
		return "", nil
	}
	if strings.TrimSpace(task) != "" && strings.TrimSpace(taskFile) != "" {
		return "", fmt.Errorf("only one of --task or --task-file may be set")
	}
	if strings.TrimSpace(taskFile) != "" {
		data, err := os.ReadFile(taskFile)
		if err != nil {
			return "", fmt.Errorf("read task file: %w", err)
		}
		return strings.TrimSpace(string(data)), nil
	}
	return strings.TrimSpace(task), nil
}

func resolveTaskInput(task, taskFile string, taskStdin bool, args []string, stdin io.Reader) (string, error) {
	if taskStdin {
		if strings.TrimSpace(task) != "" || strings.TrimSpace(taskFile) != "" || len(args) > 0 {
			return "", fmt.Errorf("--task-stdin cannot be used with --task, --task-file, or positional args")
		}
		return readTaskFromStdin(stdin)
	}
	if strings.TrimSpace(task) == "" && strings.TrimSpace(taskFile) == "" && len(args) > 0 {
		task = strings.Join(args, " ")
	}
	return resolveTaskPayload(task, taskFile)
}

func readTaskFromStdin(stdin io.Reader) (string, error) {
	if stdin == nil {
		return "", fmt.Errorf("stdin is nil")
	}
	data, err := io.ReadAll(stdin)
	if err != nil {
		return "", fmt.Errorf("read stdin: %w", err)
	}
	payload := strings.TrimSpace(string(data))
	if payload == "" {
		return "", fmt.Errorf("stdin task is empty")
	}
	return payload, nil
}
