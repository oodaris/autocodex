package autonomy

import (
	"context"
	"log/slog"

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

func (c *Controller) Run(ctx context.Context) (*state.Run, error) {
	if c.Logger != nil {
		c.Logger.Info("autonomy enabled", "mode", c.Config.Mode)
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
