package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/oodaris/autocodex/internal/api"
	"github.com/oodaris/autocodex/internal/codex"
	"github.com/oodaris/autocodex/internal/config"
	"github.com/oodaris/autocodex/internal/hub"
	"github.com/oodaris/autocodex/internal/logging"
	"github.com/oodaris/autocodex/internal/orchestrator"
	"github.com/oodaris/autocodex/internal/plugins"
	"github.com/oodaris/autocodex/internal/skills"
	"github.com/oodaris/autocodex/internal/state"
	"github.com/oodaris/autocodex/internal/terminal"
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
	case "once":
		runOnce(os.Args[2:])
	case "resume":
		runResume(os.Args[2:])
	case "kill":
		runKill(os.Args[2:])
	case "snapshot":
		runSnapshot(os.Args[2:])
	case "status":
		runStatus(os.Args[2:])
	case "beads":
		runBeads(os.Args[2:])
	case "plugins":
		runPlugins(os.Args[2:])
	case "api":
		runAPI(os.Args[2:])
	case "config":
		runConfig(os.Args[2:])
	default:
		usage()
		os.Exit(1)
	}
}

func usage() {
	fmt.Println("Usage: autocodex <command> [args]")
	fmt.Println("Commands: init, run, once, resume, kill, snapshot, status, beads, plugins, api, config")
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

	fmt.Printf("Initialized autocodex. Config: %s\n", *configPath)
}

func runRun(args []string) {
	fs := flag.NewFlagSet("run", flag.ExitOnError)
	configPath := fs.String("config", config.ResolveConfigPath(), "Config file path")
	task := fs.String("task", "", "task text to append to TODO.md before run")
	taskFile := fs.String("task-file", "", "path to task file (appended to TODO.md before run)")
	fs.Parse(args)

	cfg, err := config.Load(*configPath)
	if err != nil {
		exitErr(err)
	}

	if err := applyTaskInput(&cfg, *task, *taskFile); err != nil {
		exitErr(err)
	}

	runLoop(cfg)
}

func runOnce(args []string) {
	fs := flag.NewFlagSet("once", flag.ExitOnError)
	configPath := fs.String("config", config.ResolveConfigPath(), "Config file path")
	task := fs.String("task", "", "task text to append to TODO.md before run")
	taskFile := fs.String("task-file", "", "path to task file (appended to TODO.md before run)")
	fs.Parse(args)

	cfg, err := config.Load(*configPath)
	if err != nil {
		exitErr(err)
	}
	if err := applyTaskInput(&cfg, *task, *taskFile); err != nil {
		exitErr(err)
	}
	cfg.Loop.Mode = "bounded"
	runLoop(cfg)
}

func runStatus(args []string) {
	fs := flag.NewFlagSet("status", flag.ExitOnError)
	configPath := fs.String("config", config.ResolveConfigPath(), "Config file path")
	runID := fs.String("run", "", "run id (optional)")
	jsonOut := fs.Bool("json", false, "output JSON")
	fs.Parse(args)

	cfg, err := config.Load(*configPath)
	if err != nil {
		exitErr(err)
	}
	store := state.NewStore(cfg.StateDir(), cfg.RunsDir(), cfg.MemoryDir(), cfg.LogsDir(), cfg.ArtifactsDir())
	statuses, err := collectRunStatuses(store, *runID)
	if err != nil {
		exitErr(err)
	}
	if len(statuses) == 0 {
		fmt.Println("No runs found")
		return
	}
	if *jsonOut {
		writeJSON(statuses)
		return
	}
	for _, status := range statuses {
		fmt.Printf(
			"%s\t%s\t%s\t%d\t%s\t%s\t%s\n",
			status.ID,
			status.Status,
			status.CurrentPhase,
			status.Iterations,
			emptyOr(status.LastAction),
			emptyOr(status.StopReason),
			emptyOr(status.LastError),
		)
	}
}

func runSnapshot(args []string) {
	fs := flag.NewFlagSet("snapshot", flag.ExitOnError)
	configPath := fs.String("config", config.ResolveConfigPath(), "Config file path")
	runID := fs.String("run", "", "run id (optional, defaults to latest)")
	reason := fs.String("reason", "", "snapshot reason")
	sources := fs.String("sources", "", "comma-separated sources (memory,events,artifacts)")
	maxBytes := fs.Int("max-bytes", 0, "max snapshot bytes (0 = no limit)")
	maxEvents := fs.Int("max-events", 0, "max events included (0 = no limit)")
	maxArtifacts := fs.Int("max-artifacts", 0, "max artifacts included (0 = no limit)")
	memoryGlob := fs.String("memory-glob", "", "memory glob filter (default *.md)")
	jsonOut := fs.Bool("json", false, "output JSON")
	fs.Parse(args)

	cfg, err := config.Load(*configPath)
	if err != nil {
		exitErr(err)
	}
	store := state.NewStore(cfg.StateDir(), cfg.RunsDir(), cfg.MemoryDir(), cfg.LogsDir(), cfg.ArtifactsDir())
	if err := store.InitDirs(); err != nil {
		exitErr(err)
	}

	selectedRun := *runID
	if selectedRun == "" {
		latest, err := latestRunID(store)
		if err != nil {
			exitErr(err)
		}
		selectedRun = latest
	}

	sourceList := parseSources(*sources, cfg.Loop.Feedback.Sources)
	options := state.SnapshotOptions{
		Reason:       *reason,
		Sources:      sourceList,
		MaxBytes:     *maxBytes,
		MaxEvents:    chooseLimit(*maxEvents, cfg.Loop.Feedback.MaxEvents),
		MaxArtifacts: chooseLimit(*maxArtifacts, cfg.Loop.Feedback.MaxArtifacts),
		MemoryGlob:   defaultIfEmpty(*memoryGlob, cfg.Loop.Feedback.MemoryGlob),
	}
	snapshot, err := store.CreateSnapshot(selectedRun, options)
	if err != nil {
		exitErr(err)
	}
	if *jsonOut {
		writeJSON(snapshot)
		return
	}
	fmt.Printf(
		"Snapshot %s created for run %s (%d bytes)\nPath: %s\n",
		snapshot.Summary.ID,
		snapshot.Summary.RunID,
		snapshot.Summary.SizeBytes,
		snapshot.Summary.ContentPath,
	)
}

func runResume(args []string) {
	runControlAction(args, "resume")
}

func runKill(args []string) {
	runControlAction(args, "kill")
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

func runPlugins(args []string) {
	fs := flag.NewFlagSet("plugins", flag.ExitOnError)
	action := fs.String("action", "list", "Action: list|run")
	name := fs.String("name", "", "Plugin name (run)")
	capability := fs.String("capability", "", "Capability name (run)")
	input := fs.String("input", "", "JSON input string (run)")
	inputFile := fs.String("input-file", "", "Path to JSON input file (run)")
	configPath := fs.String("config", config.ResolveConfigPath(), "Config file path")
	fs.Parse(args)

	cfg, err := config.Load(*configPath)
	if err != nil {
		exitErr(err)
	}

	logger := logging.NewLogger(cfg.Logging.Level)
	traceID := fmt.Sprintf("trace-%d", time.Now().UnixNano())
	tenantID := os.Getenv("AUTOCODEX_TENANT_ID")
	if tenantID == "" {
		tenantID = "local"
	}
	logger = logger.With("trace_id", traceID, "tenant_id", tenantID, "route", "plugins."+*action)

	pluginsList, err := plugins.Discover(cfg.Plugins.Paths)
	if err != nil {
		logger.Error("plugin discovery failed", "status", "failed", "latency_ms", 0, "error", err.Error())
		exitErr(err)
	}

	switch *action {
	case "list":
		payload := make([]map[string]interface{}, 0, len(pluginsList))
		for _, p := range pluginsList {
			caps := make([]string, 0, len(p.Manifest.Capabilities))
			for _, cap := range p.Manifest.Capabilities {
				caps = append(caps, cap.Name)
			}
			payload = append(payload, map[string]interface{}{
				"name":         p.Manifest.Name,
				"version":      p.Manifest.Version,
				"transport":    p.Manifest.Transport,
				"capabilities": caps,
			})
		}
		writeJSON(payload)
	case "run":
		if *name == "" || *capability == "" {
			exitErr(fmt.Errorf("name and capability are required for run"))
		}
		plugin, err := findPlugin(pluginsList, *name)
		if err != nil {
			exitErr(err)
		}
		inputBytes, err := resolveInput(*input, *inputFile)
		if err != nil {
			exitErr(err)
		}
		host := plugins.Host{Timeout: time.Duration(cfg.Plugins.TimeoutSeconds) * time.Second}
		start := time.Now()
		output, err := host.Call(context.Background(), plugin, *capability, inputBytes)
		latency := time.Since(start).Milliseconds()
		if err != nil {
			logger.Error("plugin call failed", "status", "failed", "latency_ms", latency, "error", err.Error())
			writeJSON(map[string]interface{}{
				"output": nil,
				"error": map[string]string{
					"message": err.Error(),
				},
			})
			os.Exit(1)
		}
		logger.Info("plugin call completed", "status", "completed", "latency_ms", latency)
		writeJSON(map[string]interface{}{
			"output": json.RawMessage(output),
			"error":  nil,
		})
	default:
		exitErr(fmt.Errorf("unknown action: %s", *action))
	}
}

func runAPI(args []string) {
	fs := flag.NewFlagSet("api", flag.ExitOnError)
	action := fs.String("action", "serve", "Action: serve")
	configPath := fs.String("config", config.ResolveConfigPath(), "Config file path")
	fs.Parse(args)

	if *action != "serve" {
		exitErr(fmt.Errorf("unknown action: %s", *action))
	}

	cfg, err := config.Load(*configPath)
	if err != nil {
		exitErr(err)
	}
	if !cfg.API.Enabled {
		exitErr(errors.New("api is disabled in config"))
	}

	logger := logging.NewLogger(cfg.Logging.Level)
	store := state.NewStore(cfg.StateDir(), cfg.RunsDir(), cfg.MemoryDir(), cfg.LogsDir(), cfg.ArtifactsDir())
	if err := store.InitDirs(); err != nil {
		exitErr(err)
	}

	rootDir, err := filepath.Abs(filepath.Dir(*configPath))
	if err != nil {
		rootDir = ""
	}
	if cfg.Hub.Enabled && len(cfg.Hub.Workspaces) == 0 && rootDir != "" {
		wsID := filepath.Base(rootDir)
		if wsID == "" || wsID == "." || wsID == string(filepath.Separator) {
			wsID = "local"
		}
		cfg.Hub.Workspaces = []config.WorkspaceConfig{{
			ID:         wsID,
			Name:       wsID,
			Root:       rootDir,
			ConfigPath: *configPath,
		}}
	}
	var hubManager *hub.Manager
	if cfg.Hub.Enabled {
		hubManager, err = hub.NewManager(cfg, logger)
		if err != nil {
			exitErr(err)
		}
	}

	authConfig := api.NewAuthConfig(cfg.Auth)
	if authConfig.Enabled && len(authConfig.Tokens) == 0 {
		exitErr(errors.New("auth enabled but no tokens resolved"))
	}

	server := &api.Server{
		Store:    store,
		Logger:   logger,
		Hub:      hubManager,
		Terminal: terminal.NewManager(logger),
		Auth:     authConfig,
		Config:   cfg,
		RootDir:  rootDir,
	}

	if cfg.Loop.StopConditions.MaxHeartbeatSeconds > 0 {
		watchdog := &api.RunWatchdog{
			Store:               store,
			Logger:              logger,
			MaxHeartbeatSeconds: cfg.Loop.StopConditions.MaxHeartbeatSeconds,
		}
		go watchdog.Start(context.Background())
	}
	addr := net.JoinHostPort(cfg.API.Host, fmt.Sprintf("%d", cfg.API.Port))
	logger.Info("api server starting", "route", "/", "status", "starting", "latency_ms", 0, "addr", addr)
	if err := http.ListenAndServe(addr, server.Handler()); err != nil {
		exitErr(err)
	}
}

type RunStatus struct {
	ID           string             `json:"id"`
	Status       string             `json:"status"`
	CurrentPhase string             `json:"current_phase"`
	Iterations   int                `json:"iterations"`
	StartedAt    time.Time          `json:"started_at"`
	FinishedAt   *time.Time         `json:"finished_at"`
	LastAction   *string            `json:"last_action,omitempty"`
	LastActionAt *time.Time         `json:"last_action_at,omitempty"`
	StopReason   *string            `json:"stop_reason,omitempty"`
	LastError    *string            `json:"last_error,omitempty"`
	Feedback     *state.RunFeedback `json:"feedback,omitempty"`
	Control      *state.RunControl  `json:"control,omitempty"`
}

func runControlAction(args []string, action string) {
	fs := flag.NewFlagSet(action, flag.ExitOnError)
	configPath := fs.String("config", config.ResolveConfigPath(), "Config file path")
	runID := fs.String("run", "", "run id")
	reason := fs.String("reason", "", "reason (optional)")
	jsonOut := fs.Bool("json", false, "output JSON")
	fs.Parse(args)

	if *runID == "" {
		exitErr(fmt.Errorf("run id is required"))
	}
	cfg, err := config.Load(*configPath)
	if err != nil {
		exitErr(err)
	}
	store := state.NewStore(cfg.StateDir(), cfg.RunsDir(), cfg.MemoryDir(), cfg.LogsDir(), cfg.ArtifactsDir())
	if err := store.InitDirs(); err != nil {
		exitErr(err)
	}
	control, err := requestRunAction(store, *runID, action, *reason)
	if err != nil {
		exitErr(err)
	}
	if *jsonOut {
		writeJSON(control)
		return
	}
	fmt.Printf("Requested %s for run %s\n", action, *runID)
}

func requestRunAction(store *state.Store, runID, action, reason string) (*state.RunControl, error) {
	run, err := store.GetRun(runID)
	if err != nil {
		return nil, err
	}
	now := time.Now().UTC()
	control := state.RunControl{
		RunID:        run.ID,
		Status:       run.Status,
		LastAction:   &action,
		LastActionAt: &now,
		UpdatedAt:    now,
	}
	if reason != "" && action != "resume" {
		control.StopReason = &reason
	}
	if err := store.SaveRunControl(control); err != nil {
		return nil, err
	}
	return &control, nil
}

func collectRunStatuses(store *state.Store, runID string) ([]RunStatus, error) {
	var runs []state.Run
	if runID != "" {
		run, err := store.GetRun(runID)
		if err != nil {
			return nil, err
		}
		runs = []state.Run{run}
	} else {
		list, err := store.ListRuns()
		if err != nil {
			return nil, err
		}
		runs = list
	}

	statuses := make([]RunStatus, 0, len(runs))
	for _, run := range runs {
		control, err := store.GetRunControl(run.ID)
		if err != nil {
			return nil, err
		}
		feedback, err := store.GetRunFeedback(run.ID)
		if err != nil {
			return nil, err
		}
		statuses = append(statuses, RunStatus{
			ID:           run.ID,
			Status:       run.Status,
			CurrentPhase: run.CurrentPhase,
			Iterations:   run.Iterations,
			StartedAt:    run.StartedAt,
			FinishedAt:   run.FinishedAt,
			LastAction:   actionValue(control),
			LastActionAt: actionAtValue(control),
			StopReason:   stopReasonValue(control),
			LastError:    lastErrorValue(control),
			Feedback:     feedback,
			Control:      control,
		})
	}
	return statuses, nil
}

func actionValue(control *state.RunControl) *string {
	if control == nil {
		return nil
	}
	return control.LastAction
}

func actionAtValue(control *state.RunControl) *time.Time {
	if control == nil {
		return nil
	}
	return control.LastActionAt
}

func stopReasonValue(control *state.RunControl) *string {
	if control == nil {
		return nil
	}
	return control.StopReason
}

func lastErrorValue(control *state.RunControl) *string {
	if control == nil {
		return nil
	}
	return control.LastError
}

func emptyOr(value *string) string {
	if value == nil || *value == "" {
		return "—"
	}
	return *value
}

func runConfig(args []string) {
	fs := flag.NewFlagSet("config", flag.ExitOnError)
	configPath := fs.String("config", config.ResolveConfigPath(), "Config file path")
	fs.Parse(args)
	fmt.Printf("Config path: %s\n", *configPath)
}

func runLoop(cfg config.Config) {
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
		Timeout:         time.Duration(cfg.Codex.TimeoutSeconds) * time.Second,
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
		return fmt.Errorf("read config.example.yaml: %w", err)
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

func applyTaskInput(cfg *config.Config, task, taskFile string) error {
	if cfg == nil {
		return nil
	}
	if strings.TrimSpace(task) == "" && strings.TrimSpace(taskFile) == "" {
		return nil
	}
	if strings.TrimSpace(task) != "" && strings.TrimSpace(taskFile) != "" {
		return fmt.Errorf("only one of --task or --task-file may be set")
	}
	payload := task
	if strings.TrimSpace(taskFile) != "" {
		data, err := os.ReadFile(taskFile)
		if err != nil {
			return fmt.Errorf("read task file: %w", err)
		}
		payload = string(data)
	}
	payload = strings.TrimSpace(payload)
	if payload == "" {
		return nil
	}

	store := state.NewStore(cfg.StateDir(), cfg.RunsDir(), cfg.MemoryDir(), cfg.LogsDir(), cfg.ArtifactsDir())
	if err := store.InitDirs(); err != nil {
		return err
	}
	if err := store.EnsureMemoryDocs(); err != nil {
		return err
	}
	if err := appendTaskToTodo(cfg.MemoryDir(), payload); err != nil {
		return err
	}

	if cfg.Loop.Feedback.Mode == "" || cfg.Loop.Feedback.Mode == "off" {
		cfg.Loop.Feedback.Mode = "on"
	}
	fmt.Println("Task appended to TODO.md")
	return nil
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

func findPlugin(pluginsList []plugins.Plugin, name string) (plugins.Plugin, error) {
	for _, p := range pluginsList {
		if p.Manifest.Name == name {
			return p, nil
		}
	}
	return plugins.Plugin{}, fmt.Errorf("plugin not found: %s", name)
}

func writeJSON(v interface{}) {
	enc := json.NewEncoder(os.Stdout)
	enc.SetEscapeHTML(false)
	_ = enc.Encode(v)
}

func parseSources(input string, fallback []string) []string {
	if strings.TrimSpace(input) == "" {
		if len(fallback) == 0 {
			return []string{"memory", "events", "artifacts"}
		}
		return append([]string{}, fallback...)
	}
	parts := strings.Split(input, ",")
	sources := make([]string, 0, len(parts))
	for _, part := range parts {
		item := strings.TrimSpace(part)
		if item == "" {
			continue
		}
		sources = append(sources, item)
	}
	if len(sources) == 0 {
		return append([]string{}, fallback...)
	}
	return sources
}

func latestRunID(store *state.Store) (string, error) {
	runs, err := store.ListRuns()
	if err != nil {
		return "", err
	}
	if len(runs) == 0 {
		return "", fmt.Errorf("no runs found")
	}
	return runs[len(runs)-1].ID, nil
}

func chooseLimit(input, fallback int) int {
	if input != 0 {
		return input
	}
	return fallback
}

func defaultIfEmpty(value, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
}

func exitErr(err error) {
	fmt.Fprintf(os.Stderr, "error: %v\n", err)
	os.Exit(1)
}
