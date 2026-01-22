package autonomy

import (
	"context"
	"fmt"
	"path/filepath"
	"sync"
	"time"

	"github.com/oodaris/autocodex/internal/codex"
	"github.com/oodaris/autocodex/internal/config"
	"github.com/oodaris/autocodex/internal/orchestrator"
	"github.com/oodaris/autocodex/internal/state"
)

func (c *Controller) runCoordinator(ctx context.Context, specPath, planPath, tasksPath string) (*state.Run, error) {
	ready, err := listReadyBeads()
	if err != nil {
		return nil, err
	}
	readyTotal := len(ready)
	if readyTotal == 0 {
		if c.Logger != nil {
			c.Logger.Info("autonomy coordinator: no ready beads")
		}
		return nil, nil
	}

	maxBeads := c.Config.Autonomy.StopConditions.MaxBeads
	if maxBeads > 0 && len(ready) > maxBeads {
		ready = ready[:maxBeads]
	}
	readyCount := len(ready)

	strategy := c.Config.Autonomy.Coordinator.Strategy
	if strategy == "" {
		strategy = "bead"
	}
	var phases []string
	if strategy == "phase" {
		phases = c.Config.Loop.Phases
		if len(phases) == 0 {
			phases = []string{"ideate", "plan", "implement", "review", "test"}
		}
	}

	if c.Logger != nil {
		configuredMax, effectiveMax := resolveMaxParallel(c.Config.Autonomy.Coordinator, readyCount)
		fields := []any{
			"strategy", strategy,
			"ready_beads", readyCount,
			"ready_beads_total", readyTotal,
			"max_beads", maxBeads,
			"max_parallel", effectiveMax,
			"max_parallel_config", configuredMax,
			"fail_fast", c.Config.Autonomy.Coordinator.FailFast,
		}
		if len(phases) > 0 {
			fields = append(fields, "phase_count", len(phases))
		}
		c.Logger.Info(
			"autonomy coordinator starting",
			fields...,
		)
	}

	if strategy == "phase" {
		var lastRun *state.Run
		for idx, phase := range phases {
			applyActions := idx == len(phases)-1
			run, err := c.runParallelBeads(ctx, ready, phase, applyActions, specPath, planPath, tasksPath)
			if err != nil {
				return run, err
			}
			if run != nil {
				lastRun = run
			}
		}
		return lastRun, nil
	}

	return c.runParallelBeads(ctx, ready, "", true, specPath, planPath, tasksPath)
}

func (c *Controller) runParallelBeads(ctx context.Context, beads []ReadyBead, phase string, applyActions bool, specPath, planPath, tasksPath string) (*state.Run, error) {
	configuredMax, maxParallel := resolveMaxParallel(c.Config.Autonomy.Coordinator, len(beads))

	if c.Logger != nil && maxParallel > 1 {
		fields := []any{
			"strategy", c.Config.Autonomy.Coordinator.Strategy,
			"beads", len(beads),
			"max_parallel", maxParallel,
			"max_parallel_config", configuredMax,
		}
		if phase != "" {
			fields = append(fields, "phase", phase)
		}
		c.Logger.Info(
			"parallel agents launched",
			fields...,
		)
	}

	if phase != "" && c.Logger != nil {
		c.Logger.Warn(
			"autonomy coordinator running isolated phase (artifacts not shared across phases)",
			"phase", phase,
		)
	}
	if c.Config.Autonomy.RequireNext != nil && *c.Config.Autonomy.RequireNext && c.Logger != nil {
		c.Logger.Warn("autonomy coordinator ignores require_next in parallel mode")
	}

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	var lastRun *state.Run
	var lastRunMu sync.Mutex
	fixAttempts := newFixAttemptTracker()
	maxFixAttempts := c.Config.Autonomy.StopConditions.MaxFixAttempts

	beadCh := make(chan ReadyBead, len(beads))
	var errMu sync.Mutex
	var errList []error
	var wg sync.WaitGroup
	var bdMu sync.Mutex
	failFast := c.Config.Autonomy.Coordinator.FailFast

	worker := func() {
		defer wg.Done()
		for bead := range beadCh {
			if ctx.Err() != nil {
				return
			}
			run, err := c.runBeadOnce(ctx, bead, phase, applyActions, specPath, planPath, tasksPath, fixAttempts, maxFixAttempts, &bdMu)
			if err != nil {
				errMu.Lock()
				errList = append(errList, err)
				errMu.Unlock()
				if failFast {
					cancel()
					return
				}
				continue
			}
			if run != nil {
				lastRunMu.Lock()
				lastRun = run
				lastRunMu.Unlock()
			}
		}
	}

	wg.Add(maxParallel)
	for i := 0; i < maxParallel; i++ {
		go worker()
	}
	for _, bead := range beads {
		if ctx.Err() != nil {
			break
		}
		beadCh <- bead
	}
	close(beadCh)
	wg.Wait()

	if len(errList) > 0 {
		first := errList[0]
		return lastRun, fmt.Errorf("autonomy coordinator completed with %d bead errors; first: %w", len(errList), first)
	}
	return lastRun, nil
}

func resolveMaxParallel(cfg config.AutonomyCoordinatorConfig, beadCount int) (int, int) {
	configured := 0
	if cfg.MaxParallel != nil {
		configured = *cfg.MaxParallel
	}
	effective := configured
	if effective <= 0 || effective > beadCount {
		effective = beadCount
	}
	if effective <= 0 {
		effective = 1
	}
	return configured, effective
}

func (c *Controller) runBeadOnce(
	ctx context.Context,
	bead ReadyBead,
	phase string,
	applyActions bool,
	specPath string,
	planPath string,
	tasksPath string,
	fixAttempts *fixAttemptTracker,
	maxFixAttempts int,
	bdMu *sync.Mutex,
) (*state.Run, error) {
	start := time.Now()
	if c.Logger != nil {
		c.Logger.Info(
			"autonomy bead start",
			"bead_id", bead.ID,
			"title", bead.Title,
			"phase", phase,
			"apply_actions", applyActions,
		)
	}
	store := c.Store
	if c.Config.Autonomy.Coordinator.Enabled && bead.ID != "" {
		store = state.NewStore(
			c.Config.StateDir(),
			c.Config.RunsDir(),
			memoryDirForBead(c.Config, bead.ID),
			c.Config.LogsDir(),
			c.Config.ArtifactsDir(),
		)
	}
	orch := orchestrator.Orchestrator{
		Config: c.Config,
		Logger: c.Logger,
		Store:  store,
		Skills: c.Skills,
		Codex:  runnerForBead(c.Config, c.Codex, bead.ID),
	}
	if phase != "" {
		orch.Config.Loop.Phases = []string{phase}
	}

	runCtx := orchestrator.WithBeadID(ctx, bead.ID)
	run, err := orch.Run(runCtx)
	if err != nil {
		if c.Config.Beads.AutoUpdate && bdAvailable() {
			withBDLock(bdMu, func() { _ = updateBeadStatus(bead.ID, "blocked") })
		}
		if c.Logger != nil {
			c.Logger.Error(
				"autonomy bead failed",
				"bead_id", bead.ID,
				"phase", phase,
				"error", err.Error(),
				"latency_ms", time.Since(start).Milliseconds(),
			)
		}
		if bead.ID != "" {
			err = fmt.Errorf("bead %s: %w", bead.ID, err)
		}
		return run, err
	}
	if c.Logger != nil {
		c.Logger.Info(
			"autonomy run summary",
			"run_id", run.ID,
			"bead_id", bead.ID,
			"title", bead.Title,
			"phase", phase,
			"apply_actions", applyActions,
			"spec_path", specPath,
			"plan_path", planPath,
			"tasks_path", tasksPath,
			"latency_ms", time.Since(start).Milliseconds(),
			"cleanup", fmt.Sprintf("autocodex cleanup --run %s", run.ID),
		)
	}
	if !applyActions {
		return run, nil
	}

	requireActions := c.Config.Autonomy.RequireActions != nil && *c.Config.Autonomy.RequireActions
	actions, actionsErr := c.actionsFromRun(run.ID)
	if actionsErr != nil && !requireActions && c.Logger != nil {
		c.Logger.Warn(
			"autonomy actions parse failed",
			"run_id", run.ID,
			"bead_id", bead.ID,
			"phase", phase,
			"error", actionsErr.Error(),
		)
	}

	stopReason, gateFailure, updatedCurrent, err := c.applyActions(bead.ID, actions)
	if err != nil {
		return run, err
	}

	if actionsErr != nil && requireActions {
		gateFailure = true
		if stopReason == "" {
			stopReason = fmt.Sprintf("invalid ACTIONS output: %v", actionsErr)
		}
		if c.Config.Beads.AutoUpdate && bdAvailable() {
			withBDLock(bdMu, func() { _ = updateBeadStatus(bead.ID, "blocked") })
			updatedCurrent = true
		}
	}

	if requireActions && actions == nil {
		gateFailure = true
		if stopReason == "" {
			stopReason = "missing required ACTIONS output"
		}
		if c.Config.Beads.AutoUpdate && bdAvailable() {
			withBDLock(bdMu, func() { _ = updateBeadStatus(bead.ID, "blocked") })
			updatedCurrent = true
		}
	}

	if gateFailure {
		if c.Config.Beads.AutoCreate && (actions == nil || len(actions.CreateBeads) == 0) {
			var fixErr error
			withBDLock(bdMu, func() { fixErr = c.createFixBead(bead.ID, stopReason) })
			if fixErr != nil && c.Logger != nil {
				c.Logger.Warn(
					"autonomy fix bead create failed",
					"bead_id", bead.ID,
					"phase", phase,
					"error", fixErr.Error(),
				)
			}
		}
		if fixAttempts != nil && maxFixAttempts > 0 && fixAttempts.Increment(bead.ID) >= maxFixAttempts {
			return run, fmt.Errorf("autonomy gate failed: max fix attempts reached (%d)", maxFixAttempts)
		}
		if c.Config.Autonomy.StopConditions.StopOnGateFailure != nil && *c.Config.Autonomy.StopConditions.StopOnGateFailure {
			return run, fmt.Errorf("autonomy gate failed: %s", stopReason)
		}
		return run, nil
	}

	if stopReason != "" {
		return run, fmt.Errorf("autonomy stop requested: %s", stopReason)
	}

	if c.Config.Beads.AutoUpdate && !updatedCurrent {
		if bdAvailable() {
			withBDLock(bdMu, func() { _ = updateBeadStatus(bead.ID, "done") })
		}
	}

	return run, nil
}

func runnerForBead(cfg config.Config, exec codex.Executor, beadID string) codex.Executor {
	if runner, ok := exec.(codex.Runner); ok {
		cloned := runner
		cloned.Env = cloneEnvMap(runner.Env)
		if cloned.Env == nil {
			cloned.Env = map[string]string{}
		}
		if beadID != "" {
			cloned.Env["AUTOCODEX_BEAD_ID"] = beadID
			cloned.Env["BD_ISSUE"] = beadID
		}
		return cloned
	}
	env := cloneEnvMap(cfg.Codex.Env)
	if env == nil {
		env = map[string]string{}
	}
	if beadID != "" {
		env["AUTOCODEX_BEAD_ID"] = beadID
		env["BD_ISSUE"] = beadID
	}
	return codex.Runner{
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
		Env:               env,
	}
}

func cloneEnvMap(input map[string]string) map[string]string {
	if input == nil {
		return nil
	}
	clone := make(map[string]string, len(input))
	for key, value := range input {
		clone[key] = value
	}
	return clone
}

func withBDLock(mu *sync.Mutex, fn func()) {
	if mu == nil {
		fn()
		return
	}
	mu.Lock()
	defer mu.Unlock()
	fn()
}

type fixAttemptTracker struct {
	mu       sync.Mutex
	attempts map[string]int
}

func newFixAttemptTracker() *fixAttemptTracker {
	return &fixAttemptTracker{attempts: map[string]int{}}
}

func (t *fixAttemptTracker) Increment(id string) int {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.attempts[id]++
	return t.attempts[id]
}

func memoryDirForBead(cfg config.Config, beadID string) string {
	safeID := sanitizeBeadID(beadID)
	if safeID == "" {
		safeID = "unknown"
	}
	return filepath.Join(cfg.MemoryDir(), "beads", safeID)
}
