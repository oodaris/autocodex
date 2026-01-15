package autonomy

import (
	"context"
	"log/slog"
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
	Codex  codex.Runner
}

type Input struct {
	Task string
}

func (c *Controller) Run(ctx context.Context, input Input) (*state.Run, error) {
	if c.Logger != nil {
		c.Logger.Info("autonomy enabled", "mode", c.Config.Mode, "task_provided", input.Task != "")
	}
	task := strings.TrimSpace(input.Task)
	if task == "" {
		var err error
		task, err = latestTaskFromTodo(c.Config.MemoryDir())
		if err != nil {
			return nil, err
		}
	}
	specPath, planPath, slug, err := c.generateSpecAndPlan(ctx, task)
	if err != nil {
		return nil, err
	}
	tasksPath, tasksFile, err := c.generateTasksFile(ctx, task, slug, planPath)
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
	return orch.Run(ctx)
}
