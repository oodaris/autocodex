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
	startPhase := fs.String("start-phase", "", "start phase (ideate, plan, implement, review, test)")
	useLatestArtifacts := fs.Bool("use-latest-artifacts", true, "append latest spec/plan references when starting after ideate")
	collaborationMode := fs.String("collaboration-mode", "", "codex collaboration mode override (passed via -c collaboration_mode=...)")
	preset := fs.String("preset", "", "codex collaboration preset override (passed via -c collaboration_mode_preset=...)")
	noCollab := fs.Bool("no-collaboration", false, "disable codex collaboration for this run")
	swarm := fs.Bool("swarm", false, "force bead-parallel coordinator with unlimited max_parallel (enables autonomy)")
	fs.Parse(args)

	taskPayload, err := resolveTaskInput(*task, *taskFile, *taskStdin, fs.Args(), os.Stdin)
	if err != nil {
		exitErr(err)
	}

	cfg, err := config.Load(*configPath)
	if err != nil {
		exitErr(err)
	}
	applyCodexOverrides(&cfg, strings.TrimSpace(*collaborationMode), strings.TrimSpace(*preset))
	if *noCollab {
		disableCollaboration(&cfg)
	}
	applySwarmOverrides(&cfg, *swarm)

	if err := applyStartPhase(&cfg, *startPhase); err != nil {
		exitErr(err)
	}
	if cfg.Autonomy.Enabled && strings.TrimSpace(*startPhase) != "" {
		fmt.Println("Warning: autonomy is enabled; it will regenerate spec/plan before the loop. Use resume or disable autonomy to reuse existing artifacts.")
	}
	taskPayload, err = appendArtifactHints(taskPayload, cfg, *startPhase, *useLatestArtifacts)
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
	startPhase := fs.String("start-phase", "", "start phase (ideate, plan, implement, review, test)")
	useLatestArtifacts := fs.Bool("use-latest-artifacts", true, "append latest spec/plan references when starting after ideate")
	collaborationMode := fs.String("collaboration-mode", "", "codex collaboration mode override (passed via -c collaboration_mode=...)")
	preset := fs.String("preset", "", "codex collaboration preset override (passed via -c collaboration_mode_preset=...)")
	noCollab := fs.Bool("no-collaboration", false, "disable codex collaboration for this run")
	swarm := fs.Bool("swarm", false, "force bead-parallel coordinator with unlimited max_parallel (enables autonomy)")
	fs.Parse(args)

	taskPayload, err := resolveTaskInput(*task, *taskFile, *taskStdin, fs.Args(), os.Stdin)
	if err != nil {
		exitErr(err)
	}

	cfg, err := config.Load(*configPath)
	if err != nil {
		exitErr(err)
	}
	applyCodexOverrides(&cfg, strings.TrimSpace(*collaborationMode), strings.TrimSpace(*preset))
	if *noCollab {
		disableCollaboration(&cfg)
	}
	applySwarmOverrides(&cfg, *swarm)
	if err := applyStartPhase(&cfg, *startPhase); err != nil {
		exitErr(err)
	}
	if cfg.Autonomy.Enabled && strings.TrimSpace(*startPhase) != "" {
		fmt.Println("Warning: autonomy is enabled; it will regenerate spec/plan before the loop. Use resume or disable autonomy to reuse existing artifacts.")
	}
	taskPayload, err = appendArtifactHints(taskPayload, cfg, *startPhase, *useLatestArtifacts)
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
	if err := runLoopWithTask(cfg, taskPayload); err != nil {
		exitErr(err)
	}
}

func runLoopWithTask(cfg config.Config, taskPayload string) error {
	logger := logging.NewLogger(cfg.Logging.Level, cfg.Logging.Format)
	store := state.NewStore(cfg.StateDir(), cfg.RunsDir(), cfg.MemoryDir(), cfg.LogsDir(), cfg.ArtifactsDir())
	loader := skills.Loader{Paths: cfg.Skills.Paths}
	runner := codex.Runner{
		CLIPath:           cfg.Codex.CLIPath,
		Model:             cfg.Codex.Model,
		ReasoningEffort:   cfg.Codex.ReasoningEffort,
		CollaborationMode: cfg.Codex.CollaborationMode,
		Preset:            cfg.Codex.Preset,
		ExtraArgs:         cfg.Codex.ExtraArgs,
		Mode:              cfg.Mode,
		ApprovalPolicy:    cfg.Codex.ApprovalPolicy,
		SandboxMode:       cfg.Codex.SandboxMode,
		JSONOutput:        cfg.Codex.JSONOutput,
		OutputLast:        cfg.Codex.OutputLast,
		PromptStdin:       cfg.Codex.PromptStdin,
		Timeout:           time.Duration(cfg.Codex.TimeoutSeconds) * time.Second,
		IdleTimeout:       time.Duration(cfg.Loop.StopConditions.MaxIdleSeconds) * time.Second,
		Env:               cfg.Codex.Env,
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
			return err
		}
		return nil
	}
	if _, err := orch.Run(ctx); err != nil {
		return err
	}
	return nil
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

	if cfg.Loop.Feedback.Mode == "" {
		cfg.Loop.Feedback.Mode = "on"
	}
	fmt.Println("Task appended to TODO.md")
	return payload, nil
}

func applyStartPhase(cfg *config.Config, startPhase string) error {
	if cfg == nil || strings.TrimSpace(startPhase) == "" {
		return nil
	}
	startPhase = strings.ToLower(strings.TrimSpace(startPhase))
	phases := cfg.Loop.Phases
	if len(phases) == 0 {
		return fmt.Errorf("loop phases are empty; cannot start at %s", startPhase)
	}
	index := -1
	for i, phase := range phases {
		if strings.ToLower(strings.TrimSpace(phase)) == startPhase {
			index = i
			break
		}
	}
	if index == -1 {
		return fmt.Errorf("start phase %q not found in loop phases", startPhase)
	}
	cfg.Loop.Phases = phases[index:]
	return nil
}

func appendArtifactHints(payload string, cfg config.Config, startPhase string, useLatest bool) (string, error) {
	if !useLatest || strings.TrimSpace(startPhase) == "" {
		return payload, nil
	}
	startPhase = strings.ToLower(strings.TrimSpace(startPhase))
	startIdx := phaseIndex(startPhase)
	if startIdx < 0 {
		return payload, nil
	}
	specPath, planPath, tasksPath := latestAutonomyArtifacts(cfg)
	if startIdx >= phaseIndex("plan") && specPath != "" {
		payload = appendHint(payload, fmt.Sprintf("Use existing spec: %s", specPath))
	}
	if startIdx >= phaseIndex("implement") && planPath != "" {
		payload = appendHint(payload, fmt.Sprintf("Use existing plan: %s", planPath))
	}
	if startIdx >= phaseIndex("implement") && tasksPath != "" {
		payload = appendHint(payload, fmt.Sprintf("Use existing tasks file: %s", tasksPath))
	}
	return payload, nil
}

func appendHint(payload, hint string) string {
	hint = strings.TrimSpace(hint)
	if hint == "" {
		return payload
	}
	if strings.TrimSpace(payload) == "" {
		return hint
	}
	return payload + "\n\n" + hint
}

func phaseIndex(name string) int {
	switch strings.ToLower(strings.TrimSpace(name)) {
	case "ideate":
		return 0
	case "plan":
		return 1
	case "implement":
		return 2
	case "review":
		return 3
	case "test":
		return 4
	default:
		return -1
	}
}

func latestAutonomyArtifacts(cfg config.Config) (string, string, string) {
	store := state.NewStore(cfg.StateDir(), cfg.RunsDir(), cfg.MemoryDir(), cfg.LogsDir(), cfg.ArtifactsDir())
	if err := store.InitDirs(); err == nil {
		if art, err := store.LoadLatestAutonomyArtifacts(); err == nil && art != nil {
			return art.SpecPath, art.PlanPath, art.TasksPath
		}
	}
	specDir := filepath.Dir(cfg.Autonomy.SpecTemplate)
	planDir := filepath.Dir(cfg.Autonomy.PlanTemplate)
	spec := latestFile(specDir, func(name string) bool {
		return strings.HasSuffix(name, ".md") && !strings.EqualFold(name, "TEMPLATE.md")
	})
	plan := latestFile(planDir, func(name string) bool {
		return strings.HasSuffix(name, ".md") && !strings.EqualFold(name, "TEMPLATE.md")
	})
	return spec, plan, ""
}

func latestFile(dir string, accept func(string) bool) string {
	if dir == "" {
		return ""
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return ""
	}
	var latest string
	var latestMod int64
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if accept != nil && !accept(name) {
			continue
		}
		info, err := entry.Info()
		if err != nil {
			continue
		}
		mod := info.ModTime().Unix()
		if latest == "" || mod > latestMod {
			latest = filepath.Join(dir, name)
			latestMod = mod
		}
	}
	return latest
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
