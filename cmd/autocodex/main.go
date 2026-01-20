package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"io/fs"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/oodaris/autocodex/internal/api"
	"github.com/oodaris/autocodex/internal/autonomy"
	"github.com/oodaris/autocodex/internal/codex"
	"github.com/oodaris/autocodex/internal/config"
	"github.com/oodaris/autocodex/internal/hub"
	"github.com/oodaris/autocodex/internal/logging"
	"github.com/oodaris/autocodex/internal/orchestrator"
	"github.com/oodaris/autocodex/internal/plugins"
	"github.com/oodaris/autocodex/internal/skills"
	"github.com/oodaris/autocodex/internal/state"
	"github.com/oodaris/autocodex/internal/terminal"
	"github.com/oodaris/autocodex/web"
)

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(1)
	}

	cmd := os.Args[1]
	if isVersionArg(cmd) {
		printVersion()
		return
	}
	if !isCommand(cmd) && !strings.HasPrefix(cmd, "-") {
		task := strings.Join(os.Args[1:], " ")
		runRun([]string{"-task", task})
		return
	}

	switch cmd {
	case "bootstrap":
		runBootstrap(os.Args[2:])
	case "init":
		runInit(os.Args[2:])
	case "run":
		runRun(os.Args[2:])
	case "once":
		runOnce(os.Args[2:])
	case "resume":
		runResume(os.Args[2:])
	case "cleanup":
		runCleanup(os.Args[2:])
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
	case "ui":
		runUI(os.Args[2:])
	case "version":
		printVersion()
	case "config":
		runConfig(os.Args[2:])
	default:
		usage()
		os.Exit(1)
	}
}

func usage() {
	fmt.Println("Usage: autocodex <command> [args]")
	fmt.Println("Commands: bootstrap, init, run, once, resume, kill, snapshot, cleanup, status, beads, plugins, api, ui, version, config")
	fmt.Println("Shortcut: autocodex \"<task>\" (implicit run with --task)")
}

func isCommand(value string) bool {
	switch value {
	case "bootstrap", "init", "run", "once", "resume", "kill", "snapshot", "cleanup", "status", "beads", "plugins", "api", "ui", "version", "config":
		return true
	default:
		return false
	}
}

var version = "dev"

func isVersionArg(arg string) bool {
	switch arg {
	case "--version", "-v":
		return true
	default:
		return false
	}
}

func printVersion() {
	fmt.Println(version)
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

func runStatus(args []string) {
	fs := flag.NewFlagSet("status", flag.ExitOnError)
	configPath := fs.String("config", config.ResolveConfigPath(), "Config file path")
	runID := fs.String("run", "", "run id (optional)")
	latest := fs.Bool("latest", false, "show latest run only")
	jsonOut := fs.Bool("json", false, "output JSON")
	fs.Parse(args)

	cfg, err := config.Load(*configPath)
	if err != nil {
		exitErr(err)
	}
	store := state.NewStore(cfg.StateDir(), cfg.RunsDir(), cfg.MemoryDir(), cfg.LogsDir(), cfg.ArtifactsDir())
	if finalized, err := store.FinalizeStaleRuns(cfg.Loop.StopConditions.MaxHeartbeatSeconds, "stale_after"); err == nil {
		_ = finalized
	}
	statuses, err := selectRunStatuses(store, *runID, *latest)
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

func selectRunStatuses(store *state.Store, runID string, latest bool) ([]RunStatus, error) {
	if latest && strings.TrimSpace(runID) != "" {
		return nil, fmt.Errorf("--latest cannot be used with --run")
	}
	statuses, err := collectRunStatuses(store, runID)
	if err != nil {
		return nil, err
	}
	if latest {
		return filterLatest(statuses), nil
	}
	return statuses, nil
}

func filterLatest(statuses []RunStatus) []RunStatus {
	if len(statuses) == 0 {
		return statuses
	}
	return statuses[len(statuses)-1:]
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

	if *runID == "" && fs.NArg() > 0 {
		*runID = strings.TrimSpace(fs.Arg(0))
	}

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
	fs := flag.NewFlagSet("resume", flag.ExitOnError)
	configPath := fs.String("config", config.ResolveConfigPath(), "Config file path")
	runID := fs.String("run", "", "run id to resume from")
	task := fs.String("task", "", "optional task text to append to TODO.md before resume")
	taskFile := fs.String("task-file", "", "path to task file (appended to TODO.md before resume)")
	taskStdin := fs.Bool("task-stdin", false, "read task text from stdin")
	force := fs.Bool("force", false, "resume even if the run is still running")
	fs.Parse(args)

	if *runID == "" && fs.NArg() > 0 {
		*runID = strings.TrimSpace(fs.Arg(0))
	}
	if *runID == "" {
		exitErr(fmt.Errorf("run id is required"))
	}

	taskPayload, err := resolveTaskInput(*task, *taskFile, *taskStdin, fs.Args(), os.Stdin)
	if err != nil {
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
	run, err := store.GetRun(*runID)
	if err != nil {
		exitErr(err)
	}
	if run.Status == "running" && !*force {
		exitErr(fmt.Errorf("run %s is still running; use --force to resume anyway", run.ID))
	}
	if run.Status != "running" {
		fmt.Printf("Run %s is %s; starting a new run with snapshot context.\n", run.ID, run.Status)
	} else {
		fmt.Printf("Run %s is still running; resuming with a snapshot context due to --force.\n", run.ID)
	}

	sourceList := cfg.Loop.Feedback.Sources
	options := state.SnapshotOptions{
		Reason:       "resume",
		Sources:      parseSources("", sourceList),
		MaxBytes:     cfg.Loop.Feedback.MaxBytes,
		MaxEvents:    cfg.Loop.Feedback.MaxEvents,
		MaxArtifacts: cfg.Loop.Feedback.MaxArtifacts,
		MemoryGlob:   cfg.Loop.Feedback.MemoryGlob,
	}
	snapshot, err := store.CreateSnapshot(run.ID, options)
	if err != nil {
		exitErr(err)
	}

	if strings.TrimSpace(taskPayload) != "" {
		taskPayload, err = applyTaskInput(&cfg, taskPayload)
		if err != nil {
			exitErr(err)
		}
	}

	if cfg.Loop.Feedback.Mode == "" || cfg.Loop.Feedback.Mode == "off" {
		cfg.Loop.Feedback.Mode = "on"
	}
	cfg.Loop.Feedback.Sources = ensureSource(cfg.Loop.Feedback.Sources, "snapshot")
	cfg.Loop.Feedback.SnapshotPath = snapshot.Summary.ContentPath

	fmt.Printf("Resume snapshot %s created for run %s\nPath: %s\n", snapshot.Summary.ID, run.ID, snapshot.Summary.ContentPath)
	runLoop(cfg, taskPayload)
}

func runKill(args []string) {
	runControlAction(args, "kill")
}

func runCleanup(args []string) {
	fs := flag.NewFlagSet("cleanup", flag.ExitOnError)
	configPath := fs.String("config", config.ResolveConfigPath(), "Config file path")
	retentionDays := fs.Int("retention-days", 0, "remove completed runs older than N days (0 = use config)")
	dryRun := fs.Bool("dry-run", false, "list runs to be removed without deleting")
	jsonOut := fs.Bool("json", false, "output JSON")
	fs.Parse(args)

	cfg, err := config.Load(*configPath)
	if err != nil {
		exitErr(err)
	}
	if *retentionDays == 0 {
		*retentionDays = cfg.Cleanup.RetentionDays
	}
	if *retentionDays <= 0 {
		exitErr(fmt.Errorf("retention days must be > 0"))
	}
	store := state.NewStore(cfg.StateDir(), cfg.RunsDir(), cfg.MemoryDir(), cfg.LogsDir(), cfg.ArtifactsDir())
	if err := store.InitDirs(); err != nil {
		exitErr(err)
	}

	result, err := store.CleanupRuns(state.CleanupOptions{
		OlderThan: time.Duration(*retentionDays) * 24 * time.Hour,
		DryRun:    *dryRun,
	})
	if err != nil {
		exitErr(err)
	}
	if *jsonOut {
		writeJSON(result)
		return
	}
	action := "Removed"
	if *dryRun {
		action = "Would remove"
	}
	fmt.Printf("%s %d run(s). Skipped %d run(s).\n", action, len(result.Deleted), len(result.Skipped))
	if len(result.Deleted) > 0 {
		fmt.Printf("Runs: %s\n", strings.Join(result.Deleted, ", "))
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
	flagSet := flag.NewFlagSet("api", flag.ExitOnError)
	action := flagSet.String("action", "serve", "Action: serve")
	configPath := flagSet.String("config", config.ResolveConfigPath(), "Config file path")
	flagSet.Parse(args)

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
	var uiFS fs.FS
	if cfg.UI.Enabled {
		uiFS, err = web.DistFS()
		if err != nil {
			exitErr(fmt.Errorf("load embedded ui: %w", err))
		}
	}

	serveAPI(cfg, *configPath, uiFS)
}

func runUI(args []string) {
	flagSet := flag.NewFlagSet("ui", flag.ExitOnError)
	configPath := flagSet.String("config", config.ResolveConfigPath(), "Config file path")
	flagSet.Parse(args)

	cfg, err := config.Load(*configPath)
	if err != nil {
		exitErr(err)
	}
	cfg.API.Enabled = true
	cfg.UI.Enabled = true

	uiFS, err := web.DistFS()
	if err != nil {
		exitErr(fmt.Errorf("load embedded ui: %w", err))
	}

	serveAPI(cfg, *configPath, uiFS)
}

func serveAPI(cfg config.Config, configPath string, uiFS fs.FS) {
	logger := logging.NewLogger(cfg.Logging.Level)
	store := state.NewStore(cfg.StateDir(), cfg.RunsDir(), cfg.MemoryDir(), cfg.LogsDir(), cfg.ArtifactsDir())
	if err := store.InitDirs(); err != nil {
		exitErr(err)
	}

	rootDir, err := filepath.Abs(filepath.Dir(configPath))
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
			ConfigPath: configPath,
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
		UIFS:     uiFS,
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
	baseURL := fmt.Sprintf("http://%s:%d", hostForLog(cfg.API.Host), cfg.API.Port)
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		if isAddrInUse(err) {
			logger.Error("api bind failed", "addr", addr, "error", err.Error(), "hint", "stop the existing process or change api.port in autocodex.yaml")
		}
		exitErr(err)
	}
	logger.Info("api server starting", "route", "/", "status", "starting", "latency_ms", 0, "addr", addr, "url", baseURL, "config", configPath)
	if uiFS != nil {
		logger.Info("ui embedded", "route", "/", "status", "enabled", "latency_ms", 0, "addr", addr, "url", baseURL)
	}
	logger.Info("api endpoints", "health", baseURL+"/health", "runs", baseURL+"/runs", "memory", baseURL+"/memory")
	if uiFS != nil {
		logger.Info("ui ready", "url", baseURL, "hub", baseURL+"/hub")
	}
	if err := http.Serve(listener, server.Handler()); err != nil {
		exitErr(err)
	}
}

func hostForLog(host string) string {
	trimmed := strings.TrimSpace(host)
	switch trimmed {
	case "", "0.0.0.0", "::", "[::]":
		return "127.0.0.1"
	default:
		return trimmed
	}
}

func isAddrInUse(err error) bool {
	if err == nil {
		return false
	}
	lower := strings.ToLower(err.Error())
	return strings.Contains(lower, "address already in use")
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

	if *runID == "" && fs.NArg() > 0 {
		*runID = strings.TrimSpace(fs.Arg(0))
	}

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
	if action == "kill" {
		lock, _ := store.GetRunLock(run.ID)
		if lock != nil && lock.PID > 0 && state.IsProcessAlive(lock.PID) {
			_ = state.TerminateProcess(lock.PID)
		} else {
			stopReason := reason
			if strings.TrimSpace(stopReason) == "" {
				stopReason = "killed"
			}
			if err := store.FinalizeRun(run.ID, "failed", stopReason, "kill"); err != nil {
				return nil, err
			}
			return store.GetRunControl(run.ID)
		}
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

func ensureSource(sources []string, name string) []string {
	for _, item := range sources {
		if item == name {
			return sources
		}
	}
	return append(sources, name)
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
