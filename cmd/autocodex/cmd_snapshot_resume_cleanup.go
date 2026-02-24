package main

import (
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/oodaris/autocodex/internal/config"
	"github.com/oodaris/autocodex/internal/state"
)

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
	force := fs.Bool("force", false, "resume even if the run is still running or completed without a new task")
	list := fs.Bool("list", false, "list runs and exit")
	startPhase := fs.String("start-phase", "", "start phase (ideate, plan, implement, review, test)")
	useLatestArtifacts := fs.Bool("use-latest-artifacts", true, "append latest spec/plan references when starting after ideate")
	collaborationMode := fs.String("collaboration-mode", "", "codex collaboration mode override (passed via -c collaboration_mode=...)")
	preset := fs.String("preset", "", "codex collaboration preset override (passed via -c collaboration_mode_preset=...)")
	noCollab := fs.Bool("no-collaboration", false, "disable codex collaboration for this run")
	swarm := fs.Bool("swarm", false, "force bead-parallel coordinator with unlimited max_parallel (enables autonomy)")
	beadScope := fs.String("bead-scope", "", "coordinator bead selection mode override (run_scoped|all_ready)")
	allowAllReadyFallback := fs.Bool("allow-all-ready-fallback", false, "allow fallback to all ready beads when run_scoped selection has no matches")
	beadIDs := fs.String("bead", "", "optional comma-separated bead id selectors")
	beadPrefix := fs.String("bead-prefix", "", "optional bead id prefix selector")
	fs.Parse(args)

	extraArgs := fs.Args()
	if *runID == "" && fs.NArg() > 0 {
		*runID = strings.TrimSpace(fs.Arg(0))
		extraArgs = extraArgs[1:]
	}

	taskPayload, err := resolveTaskInput(*task, *taskFile, *taskStdin, extraArgs, os.Stdin)
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
	applyCoordinatorOverrides(&cfg, *beadScope, *allowAllReadyFallback, *beadIDs, *beadPrefix)
	if err := applyStartPhase(&cfg, *startPhase); err != nil {
		exitErr(err)
	}
	if cfg.Autonomy.Enabled && strings.TrimSpace(*startPhase) != "" {
		fmt.Println("Warning: autonomy is enabled; it will regenerate spec/plan before the loop. Use resume or disable autonomy to reuse existing artifacts.")
	}
	store := state.NewStore(cfg.StateDir(), cfg.RunsDir(), cfg.MemoryDir(), cfg.LogsDir(), cfg.ArtifactsDir())
	if err := store.InitDirs(); err != nil {
		exitErr(err)
	}
	if *runID == "" || *list {
		statuses, err := collectRunStatuses(store, "")
		if err != nil {
			exitErr(err)
		}
		if len(statuses) == 0 {
			exitErr(fmt.Errorf("no runs found"))
		}
		printRunTable(statuses)
		if *list {
			return
		}
		if !isInteractiveTerminal(os.Stdin) {
			exitErr(fmt.Errorf("run id is required; use --run or --list"))
		}
		selected, err := promptRunSelection(statuses, os.Stdin, os.Stdout)
		if err != nil {
			exitErr(err)
		}
		*runID = selected
	}
	run, err := store.GetRun(*runID)
	if err != nil {
		exitErr(err)
	}
	message, err := resumeMessage(run, taskPayload, *force)
	if err != nil {
		exitErr(err)
	}
	fmt.Println(message)

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
		taskPayload, err = appendArtifactHints(taskPayload, cfg, *startPhase, *useLatestArtifacts)
		if err != nil {
			exitErr(err)
		}
		taskPayload, err = applyTaskInput(&cfg, taskPayload)
		if err != nil {
			exitErr(err)
		}
	}

	if cfg.Loop.Feedback.Mode == "" {
		cfg.Loop.Feedback.Mode = "on"
	}
	cfg.Loop.Feedback.Sources = ensureSource(cfg.Loop.Feedback.Sources, "snapshot")
	cfg.Loop.Feedback.SnapshotPath = snapshot.Summary.ContentPath

	fmt.Printf("Resume snapshot %s created for run %s\nPath: %s\n", snapshot.Summary.ID, run.ID, snapshot.Summary.ContentPath)
	runLoop(cfg, taskPayload)
}

func resumeMessage(run state.Run, taskPayload string, force bool) (string, error) {
	hasTask := strings.TrimSpace(taskPayload) != ""
	if run.Status == "running" && !force {
		return "", fmt.Errorf("run %s is still running; use --force to resume anyway", run.ID)
	}
	if run.Status == "completed" && !force {
		return "", fmt.Errorf("run %s is completed; use --force to resume a completed run", run.ID)
	}
	if run.Status != "running" && !hasTask && !force {
		return "", fmt.Errorf("run %s is %s; provide --task/--task-file/--task-stdin or use --force to resume without a new task", run.ID, run.Status)
	}
	switch {
	case run.Status == "running":
		return fmt.Sprintf("Run %s is still running; resuming with a snapshot context due to --force.", run.ID), nil
	case run.Status == "completed" && !hasTask:
		return fmt.Sprintf("Run %s is completed; resuming without a new task due to --force.", run.ID), nil
	case run.Status == "completed" && hasTask:
		return fmt.Sprintf("Run %s is completed; starting a new run with snapshot context due to --force.", run.ID), nil
	case !hasTask:
		return fmt.Sprintf("Run %s is %s; resuming without a new task due to --force.", run.ID, run.Status), nil
	default:
		return fmt.Sprintf("Run %s is %s; starting a new run with snapshot context.", run.ID, run.Status), nil
	}
}

func runCleanup(args []string) {
	fs := flag.NewFlagSet("cleanup", flag.ExitOnError)
	configPath := fs.String("config", config.ResolveConfigPath(), "Config file path")
	runID := fs.String("run", "", "remove a single run by id")
	retentionDays := fs.Int("retention-days", 0, "remove completed runs older than N days (0 = use config)")
	dryRun := fs.Bool("dry-run", false, "list runs to be removed without deleting")
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

	if strings.TrimSpace(*runID) != "" {
		if _, err := store.GetRun(*runID); err != nil {
			exitErr(err)
		}
		if *dryRun {
			result := state.CleanupResult{Deleted: []string{*runID}}
			if *jsonOut {
				writeJSON(result)
				return
			}
			fmt.Printf("Would remove run %s.\n", *runID)
			return
		}
		if err := store.DeleteRun(*runID); err != nil {
			exitErr(err)
		}
		result := state.CleanupResult{Deleted: []string{*runID}}
		if *jsonOut {
			writeJSON(result)
			return
		}
		fmt.Printf("Removed run %s.\n", *runID)
		return
	}

	if *retentionDays == 0 {
		*retentionDays = cfg.Cleanup.RetentionDays
	}
	if *retentionDays <= 0 {
		exitErr(fmt.Errorf("retention days must be > 0"))
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
