package orchestrator

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/oodaris/autocodex/internal/codex"
	"github.com/oodaris/autocodex/internal/config"
	"github.com/oodaris/autocodex/internal/skills"
	"github.com/oodaris/autocodex/internal/state"
)

type Orchestrator struct {
	Config config.Config
	Logger *slog.Logger
	Store  *state.Store
	Skills skills.Loader
	Codex  codex.Runner
}

func (o *Orchestrator) Run(ctx context.Context) (*state.Run, error) {
	if err := o.Store.InitDirs(); err != nil {
		return nil, err
	}
	if err := o.Store.EnsureMemoryDocs(); err != nil {
		return nil, err
	}

	run, err := o.Store.CreateRun()
	if err != nil {
		return nil, err
	}

	traceID := newTraceID()
	beadID := os.Getenv("AUTOCODEX_BEAD_ID")
	if beadID == "" {
		beadID = os.Getenv("BD_ISSUE")
	}
	tenantID := os.Getenv("AUTOCODEX_TENANT_ID")
	if tenantID == "" {
		tenantID = "local"
	}
	logger := o.Logger.With(
		"trace_id", traceID,
		"tenant_id", tenantID,
		"run_id", run.ID,
		"route", "orchestrator.run",
		"bead_id", beadID,
	)

	logger.Info("run started", "status", "started", "latency_ms", 0)
	_ = o.Store.AppendEvent(state.RunEvent{
		ID:      eventID("run-start"),
		RunID:   run.ID,
		TS:      time.Now().UTC(),
		Type:    "run_start",
		Message: "run started",
		Meta:    map[string]string{"trace_id": traceID, "bead_id": beadID},
	})

	for _, phase := range o.Config.Loop.Phases {
		run.CurrentPhase = phase
		run.Iterations++
		if err := o.Store.SaveRun(run); err != nil {
			return run, err
		}

		phaseStart := time.Now().UTC()
		logger.Info("phase start", "phase", phase, "status", "started", "latency_ms", 0)
		_ = o.Store.AppendEvent(state.RunEvent{
			ID:      eventID("phase-start"),
			RunID:   run.ID,
			TS:      time.Now().UTC(),
			Type:    "phase_start",
			Phase:   phase,
			Message: "phase started",
			Meta:    map[string]string{"trace_id": traceID, "bead_id": beadID},
		})

		prompt := o.buildPrompt(phase, run.ID)
		phaseCtx, cancel := context.WithTimeout(ctx, time.Duration(o.Config.Codex.TimeoutSeconds)*time.Second)
		output, execErr := o.Codex.Exec(phaseCtx, prompt)
		cancel()
		if execErr != nil {
			latency := time.Since(phaseStart).Milliseconds()
			logger.Error("phase failed", "phase", phase, "error", execErr.Error(), "status", "failed", "latency_ms", latency)
			_ = o.Store.AppendEvent(state.RunEvent{
				ID:      eventID("phase-failed"),
				RunID:   run.ID,
				TS:      time.Now().UTC(),
				Type:    "phase_failed",
				Phase:   phase,
				Message: execErr.Error(),
				Meta:    map[string]string{"trace_id": traceID, "bead_id": beadID},
			})
			run.Status = "failed"
			finished := time.Now().UTC()
			run.FinishedAt = &finished
			_ = o.Store.SaveRun(run)
			return run, execErr
		}

		if err := o.writeArtifact(run.ID, phase, output); err != nil {
			logger.Warn("artifact write failed", "phase", phase, "error", err.Error())
		}

		latency := time.Since(phaseStart).Milliseconds()
		logger.Info("phase complete", "phase", phase, "status", "completed", "latency_ms", latency)
		_ = o.Store.AppendEvent(state.RunEvent{
			ID:      eventID("phase-complete"),
			RunID:   run.ID,
			TS:      time.Now().UTC(),
			Type:    "phase_complete",
			Phase:   phase,
			Message: "phase completed",
			Meta:    map[string]string{"trace_id": traceID, "bead_id": beadID},
		})
	}

	run.Status = "completed"
	finished := time.Now().UTC()
	run.FinishedAt = &finished
	_ = o.Store.SaveRun(run)

	logger.Info("run completed", "status", "completed", "latency_ms", 0)
	_ = o.Store.AppendEvent(state.RunEvent{
		ID:      eventID("run-complete"),
		RunID:   run.ID,
		TS:      time.Now().UTC(),
		Type:    "run_complete",
		Message: "run completed",
		Meta:    map[string]string{"trace_id": traceID, "bead_id": beadID},
	})

	return run, nil
}

func (o *Orchestrator) buildPrompt(phase, runID string) string {
	skillName := skillForPhase(phase)
	skillContent := ""
	if skillName != "" {
		skill, err := o.Skills.LoadSkill(skillName)
		if err == nil {
			skillContent = skill.Content
		}
	}

	var b strings.Builder
	b.WriteString("Autocodex run ID: ")
	b.WriteString(runID)
	b.WriteString("\n")
	b.WriteString("Phase: ")
	b.WriteString(phase)
	b.WriteString("\n")
	if skillName != "" {
		b.WriteString("Requested skill: ")
		b.WriteString(skillName)
		b.WriteString("\n\n")
	}
	if skillContent != "" {
		b.WriteString("--- Skill Content ---\n")
		b.WriteString(skillContent)
		b.WriteString("\n--- End Skill Content ---\n")
	}
	return b.String()
}

func (o *Orchestrator) writeArtifact(runID, phase, output string) error {
	if output == "" {
		return nil
	}
	path := filepath.Join(o.Store.RunsDir, runID, "artifacts", fmt.Sprintf("%s.txt", phase))
	return os.WriteFile(path, []byte(output), 0o644)
}

func skillForPhase(phase string) string {
	switch phase {
	case "ideate":
		return "core-qna-synthesis"
	case "plan":
		return "core-holistic-planning-and-tracking"
	case "implement":
		return "eng-fullstack-engineer"
	case "review":
		return "eng-code-review-playbook"
	case "test":
		return "eng-smart-test-runner"
	default:
		return ""
	}
}

func newTraceID() string {
	buf := make([]byte, 8)
	if _, err := rand.Read(buf); err != nil {
		return fmt.Sprintf("trace-%d", time.Now().UnixNano())
	}
	return hex.EncodeToString(buf)
}

func eventID(prefix string) string {
	buf := make([]byte, 6)
	if _, err := rand.Read(buf); err != nil {
		return fmt.Sprintf("%s-%d", prefix, time.Now().UnixNano())
	}
	return fmt.Sprintf("%s-%s", prefix, hex.EncodeToString(buf))
}
