package autonomy

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"strings"

	"github.com/oodaris/autocodex/internal/codex"
	"github.com/oodaris/autocodex/internal/config"
	"github.com/oodaris/autocodex/internal/orchestrator"
	"github.com/oodaris/autocodex/internal/skills"
	"github.com/oodaris/autocodex/internal/state"
)

type Controller struct {
	Config config.Config
	Logger *slog.Logger
	Store  *state.Store
	Skills skills.Loader
	Codex  codex.Executor
}

type Input struct {
	Task string
}

func (c *Controller) Run(ctx context.Context, input Input) (*state.Run, error) {
	if c.Logger != nil {
		c.Logger.Info("autonomy enabled", "mode", c.Config.Mode, "task_provided", input.Task != "")
	}
	requireActions := c.Config.Autonomy.RequireActions != nil && *c.Config.Autonomy.RequireActions
	requireNext := c.Config.Autonomy.RequireNext != nil && *c.Config.Autonomy.RequireNext
	requireBD := c.Config.Autonomy.RequireBD != nil && *c.Config.Autonomy.RequireBD
	if requireBD && !bdAvailable() {
		return nil, fmt.Errorf("bd is required for autonomy runs (autonomy.require_bd: true)")
	}
	task := strings.TrimSpace(input.Task)
	if task == "" {
		var err error
		task, err = latestTaskFromTodo(c.Config.MemoryDir())
		if err != nil {
			return nil, err
		}
	}
	runTag := newArtifactTag()
	specPath, planPath, slug, err := c.generateSpecAndPlan(ctx, task, runTag)
	if err != nil {
		return nil, err
	}
	tasksPath, tasksFile, err := c.generateTasksFile(ctx, task, slug, planPath, runTag)
	if err != nil {
		return nil, err
	}
	if err := c.createBeads(tasksFile); err != nil {
		return nil, err
	}
	if c.Logger != nil {
		c.Logger.Info("autonomy artifacts ready", "spec_path", specPath, "plan_path", planPath, "tasks_path", tasksPath)
	}

	orch := orchestrator.Orchestrator{
		Config: c.Config,
		Logger: c.Logger,
		Store:  c.Store,
		Skills: c.Skills,
		Codex:  c.Codex,
	}

	var lastRun *state.Run
	var nextHint string
	var fixAttempts int
	maxBeads := c.Config.Autonomy.StopConditions.MaxBeads
	maxFixAttempts := c.Config.Autonomy.StopConditions.MaxFixAttempts

	for processed := 0; ; processed++ {
		if maxBeads > 0 && processed >= maxBeads {
			if c.Logger != nil {
				c.Logger.Warn("autonomy stop: max beads reached", "max_beads", maxBeads)
			}
			return lastRun, nil
		}
		bead, err := c.selectBead(nextHint)
		nextHint = ""
		if err != nil {
			return lastRun, err
		}
		if bead == nil {
			if c.Logger != nil {
				c.Logger.Info("autonomy complete: no ready beads")
			}
			return lastRun, nil
		}

		restoreEnv := setBeadEnv(bead.ID)
		run, err := orch.Run(ctx)
		restoreEnv()
		if err != nil {
			if c.Config.Beads.AutoUpdate {
				_ = updateBeadStatus(bead.ID, "blocked")
			}
			return run, err
		}
		lastRun = run

		actions, actionsErr := c.actionsFromRun(run.ID)
		if actionsErr != nil && !requireActions && c.Logger != nil {
			c.Logger.Warn("autonomy actions parse failed", "run_id", run.ID, "error", actionsErr.Error())
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
				_ = updateBeadStatus(bead.ID, "blocked")
				updatedCurrent = true
			}
		}

		if requireActions && actions == nil {
			gateFailure = true
			if stopReason == "" {
				stopReason = "missing required ACTIONS output"
			}
			if c.Config.Beads.AutoUpdate && bdAvailable() {
				_ = updateBeadStatus(bead.ID, "blocked")
				updatedCurrent = true
			}
		}

		if actions != nil && actions.Next.Type == "bead" {
			nextHint = sanitizeBeadID(actions.Next.ID)
		}

		if !gateFailure && requireNext && actions != nil && actions.Next.Type != "bead" {
			ready, err := listReadyBeads()
			if err != nil {
				return run, err
			}
			if len(ready) > 1 {
				gateFailure = true
				if stopReason == "" {
					stopReason = "explicit next bead required when multiple beads are ready"
				}
				if c.Config.Beads.AutoUpdate && bdAvailable() {
					_ = updateBeadStatus(bead.ID, "blocked")
					updatedCurrent = true
				}
			}
		}

		if gateFailure {
			fixAttempts++
			reason := stopReason
			if strings.TrimSpace(reason) == "" {
				reason = "gate failure"
			}
			if c.Config.Beads.AutoCreate && (actions == nil || len(actions.CreateBeads) == 0) {
				if err := c.createFixBead(bead.ID, reason); err != nil && c.Logger != nil {
					c.Logger.Warn("autonomy fix bead create failed", "error", err.Error())
				}
			}
			if maxFixAttempts > 0 && fixAttempts >= maxFixAttempts {
				return run, fmt.Errorf("autonomy gate failed: max fix attempts reached (%d)", maxFixAttempts)
			}
			if c.Config.Autonomy.StopConditions.StopOnGateFailure != nil && *c.Config.Autonomy.StopConditions.StopOnGateFailure {
				return run, fmt.Errorf("autonomy gate failed: %s", reason)
			}
			continue
		}
		fixAttempts = 0

		if stopReason != "" {
			if c.Logger != nil {
				c.Logger.Warn("autonomy stop requested", "reason", stopReason)
			}
			return run, nil
		}

		if c.Config.Beads.AutoUpdate && !updatedCurrent && !gateFailure {
			_ = updateBeadStatus(bead.ID, "done")
		}
	}
}

func setBeadEnv(beadID string) func() {
	prevBead := os.Getenv("AUTOCODEX_BEAD_ID")
	prevBD := os.Getenv("BD_ISSUE")
	_ = os.Setenv("AUTOCODEX_BEAD_ID", beadID)
	_ = os.Setenv("BD_ISSUE", beadID)
	return func() {
		if prevBead == "" {
			_ = os.Unsetenv("AUTOCODEX_BEAD_ID")
		} else {
			_ = os.Setenv("AUTOCODEX_BEAD_ID", prevBead)
		}
		if prevBD == "" {
			_ = os.Unsetenv("BD_ISSUE")
		} else {
			_ = os.Setenv("BD_ISSUE", prevBD)
		}
	}
}
