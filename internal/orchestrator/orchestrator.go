package orchestrator

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sort"
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
	Codex  codex.Executor
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
	lock, err := o.Store.AcquireRunLock(run.ID)
	if err != nil {
		run.Status = "failed"
		finished := time.Now().UTC()
		run.FinishedAt = &finished
		_ = o.Store.SaveRun(run)
		return run, err
	}
	defer func() {
		_ = o.Store.ReleaseRunLock(run.ID)
	}()
	if lock != nil {
		_ = o.Store.TouchRunHeartbeat(run.ID, lock.PID)
	}

	heartbeatCtx, heartbeatCancel := context.WithCancel(ctx)
	defer heartbeatCancel()
	go o.heartbeatLoop(heartbeatCtx, run.ID, os.Getpid())

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
	_ = o.Store.SaveRunControl(state.RunControl{
		RunID:     run.ID,
		Status:    "running",
		UpdatedAt: time.Now().UTC(),
	})
	_ = o.Store.AppendEvent(state.RunEvent{
		ID:      eventID("run-start"),
		RunID:   run.ID,
		TS:      time.Now().UTC(),
		Type:    "run_start",
		Message: "run started",
		Meta:    map[string]string{"trace_id": traceID, "bead_id": beadID},
	})

	startTime := time.Now().UTC()
	lastProgress := startTime
	consecutiveFailures := 0
	loopMode := o.Config.Loop.Mode
	phases := o.Config.Loop.Phases

	for {
		for _, phase := range phases {
			if stop, reason, status, action := o.shouldStop(ctx, run, startTime, lastProgress, consecutiveFailures); stop {
				o.finalizeRun(run, status, &reason, nil, action)
				logger.Info("run stopped", "status", status, "reason", reason, "latency_ms", 0)
				_ = o.Store.AppendEvent(state.RunEvent{
					ID:      eventID("run-stopped"),
					RunID:   run.ID,
					TS:      time.Now().UTC(),
					Type:    "run_stopped",
					Message: reason,
					Meta:    map[string]string{"trace_id": traceID, "bead_id": beadID},
				})
				return run, nil
			}

			run.CurrentPhase = phase
			run.Iterations++
			if err := o.Store.SaveRun(run); err != nil {
				return run, err
			}
			_ = o.Store.TouchRunHeartbeat(run.ID, os.Getpid())

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

			feedbackText, feedbackMeta, err := o.gatherFeedback(run.ID)
			if err != nil {
				logger.Warn("feedback gather failed", "phase", phase, "error", err.Error())
			} else if feedbackMeta.RunID != "" {
				_ = o.Store.SaveRunFeedback(feedbackMeta)
			}

			prompt := o.buildPrompt(phase, run.ID, feedbackText)
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
				consecutiveFailures++
				errMsg := execErr.Error()
				controlStatus := "failed"
				if loopMode == "continuous" && o.Config.Loop.StopConditions.MaxConsecutiveFailures > 0 &&
					consecutiveFailures < o.Config.Loop.StopConditions.MaxConsecutiveFailures {
					controlStatus = "running"
				}
				_ = o.Store.SaveRunControl(state.RunControl{
					RunID:     run.ID,
					Status:    controlStatus,
					LastError: &errMsg,
					UpdatedAt: time.Now().UTC(),
				})
				if controlStatus == "running" {
					continue
				}
				run.Status = "failed"
				finished := time.Now().UTC()
				run.FinishedAt = &finished
				_ = o.Store.SaveRun(run)
				return run, execErr
			}
			consecutiveFailures = 0

			if err := o.writeArtifact(run.ID, phase, output); err != nil {
				logger.Warn("artifact write failed", "phase", phase, "error", err.Error())
			}
			phaseFinished := time.Now().UTC()
			if err := o.appendPhaseSummary(run.ID, phase, phaseStart, phaseFinished, output); err != nil {
				logger.Warn("phase summary append failed", "phase", phase, "error", err.Error())
			}
			if feedbackMeta.RunID != "" {
				feedbackMeta.LastOutputSummary = fmt.Sprintf("phase %s output bytes=%d", phase, len(output))
				feedbackMeta.UpdatedAt = time.Now().UTC()
				_ = o.Store.SaveRunFeedback(feedbackMeta)
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
			lastProgress = time.Now().UTC()
		}

		if loopMode != "continuous" {
			break
		}
	}

	o.finalizeRun(run, "completed", nil, nil, nil)
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

func (o *Orchestrator) buildPrompt(phase, runID, feedback string) string {
	skillName := skillForPhase(phase)
	skillContent := ""
	if skillName != "" {
		skill, err := o.Skills.LoadSkill(skillName)
		if err == nil {
			skillContent = skill.Content
		}
	}

	var b strings.Builder
	b.WriteString("autocodex run ID: ")
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
	if feedback != "" {
		b.WriteString("\n--- Feedback Context ---\n")
		b.WriteString(feedback)
		b.WriteString("\n--- End Feedback ---\n")
	}
	if o.Config.Autonomy.Enabled && strings.TrimSpace(o.Config.Autonomy.ActionsSchema) != "" {
		if schemaContent, err := os.ReadFile(filepath.Clean(o.Config.Autonomy.ActionsSchema)); err == nil {
			b.WriteString("\n--- Autonomy Actions Schema ---\n")
			b.WriteString(string(schemaContent))
			b.WriteString("\n--- End Autonomy Actions Schema ---\n")
		}
		b.WriteString("\nAutonomy actions:\n")
		b.WriteString("- Only output an ACTIONS JSON block in the test phase.\n")
		b.WriteString("- When the phase is test, append the following markers:\n")
		b.WriteString("ACTIONS_JSON_START\n")
		b.WriteString("<JSON that conforms to the schema>\n")
		b.WriteString("ACTIONS_JSON_END\n")
		b.WriteString("- If tests or acceptance criteria fail, set gates.blocking = true and include a stop.reason.\n")
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

func (o *Orchestrator) shouldStop(
	ctx context.Context,
	run *state.Run,
	startTime time.Time,
	lastProgress time.Time,
	consecutiveFailures int,
) (bool, string, string, *string) {
	if ctx.Err() != nil {
		return true, "context_canceled", "canceled", nil
	}

	control, err := o.Store.GetRunControl(run.ID)
	if err == nil && control != nil && control.LastAction != nil {
		action := *control.LastAction
		switch action {
		case "stop", "cancel":
			return true, action, "canceled", &action
		case "kill":
			return true, action, "failed", &action
		}
	}

	if o.Config.Loop.Mode != "continuous" {
		return false, "", "", nil
	}

	if o.Config.Loop.MaxIterations > 0 && run.Iterations >= o.Config.Loop.MaxIterations {
		return true, "max_iterations", "completed", nil
	}
	if o.Config.Loop.StopConditions.MaxDurationSeconds > 0 {
		elapsed := time.Since(startTime)
		if elapsed >= time.Duration(o.Config.Loop.StopConditions.MaxDurationSeconds)*time.Second {
			return true, "max_duration", "completed", nil
		}
	}
	if o.Config.Loop.StopConditions.MaxIdleSeconds > 0 {
		idle := time.Since(lastProgress)
		if idle >= time.Duration(o.Config.Loop.StopConditions.MaxIdleSeconds)*time.Second {
			return true, "max_idle", "canceled", nil
		}
	}
	if o.Config.Loop.StopConditions.MaxConsecutiveFailures > 0 &&
		consecutiveFailures >= o.Config.Loop.StopConditions.MaxConsecutiveFailures {
		return true, "max_consecutive_failures", "failed", nil
	}
	return false, "", "", nil
}

func (o *Orchestrator) finalizeRun(
	run *state.Run,
	status string,
	stopReason *string,
	lastErr error,
	lastAction *string,
) {
	run.Status = status
	finished := time.Now().UTC()
	run.FinishedAt = &finished
	_ = o.Store.SaveRun(run)

	var errMsg *string
	if lastErr != nil {
		msg := lastErr.Error()
		errMsg = &msg
	}

	_ = o.Store.SaveRunControl(state.RunControl{
		RunID:      run.ID,
		Status:     status,
		StopReason: stopReason,
		LastError:  errMsg,
		LastAction: lastAction,
		UpdatedAt:  time.Now().UTC(),
	})

	if err := o.appendRunSummary(run, stopReason, lastErr, lastAction); err != nil && o.Logger != nil {
		o.Logger.Warn("memory summary append failed", "run_id", run.ID, "error", err.Error())
	}
}

func (o *Orchestrator) appendRunSummary(
	run *state.Run,
	stopReason *string,
	lastErr error,
	lastAction *string,
) error {
	if o.Store == nil || run == nil {
		return nil
	}
	var b strings.Builder
	finished := ""
	if run.FinishedAt != nil {
		finished = run.FinishedAt.UTC().Format(time.RFC3339)
	}
	b.WriteString(fmt.Sprintf("## Run %s — %s\n", run.ID, run.Status))
	if finished != "" {
		b.WriteString(fmt.Sprintf("- Finished: %s\n", finished))
	}
	if !run.StartedAt.IsZero() {
		b.WriteString(fmt.Sprintf("- Started: %s\n", run.StartedAt.UTC().Format(time.RFC3339)))
	}
	b.WriteString(fmt.Sprintf("- Iterations: %d\n", run.Iterations))
	if run.CurrentPhase != "" {
		b.WriteString(fmt.Sprintf("- Last phase: %s\n", run.CurrentPhase))
	}
	if stopReason != nil {
		b.WriteString(fmt.Sprintf("- Stop reason: %s\n", *stopReason))
	}
	if lastAction != nil {
		b.WriteString(fmt.Sprintf("- Last action: %s\n", *lastAction))
	}
	if lastErr != nil {
		b.WriteString(fmt.Sprintf("- Last error: %s\n", lastErr.Error()))
	}

	if artifacts, err := o.Store.ListArtifacts(run.ID); err == nil && len(artifacts) > 0 {
		sort.Slice(artifacts, func(i, j int) bool {
			return artifacts[i].CreatedAt.Before(artifacts[j].CreatedAt)
		})
		b.WriteString("- Artifacts:\n")
		for _, artifact := range artifacts {
			b.WriteString(fmt.Sprintf("  - %s (%s, %d bytes)\n", artifact.Name, artifact.Type, artifact.SizeBytes))
		}
	}

	if events, err := o.Store.ListEvents(run.ID); err == nil && len(events) > 0 {
		if len(events) > 5 {
			events = events[len(events)-5:]
		}
		b.WriteString("- Recent events:\n")
		for _, event := range events {
			b.WriteString(fmt.Sprintf("  - %s [%s] %s %s\n",
				event.TS.UTC().Format(time.RFC3339),
				event.Type,
				event.Phase,
				event.Message,
			))
		}
	}

	return o.Store.AppendMemoryDoc("PROGRESS.md", b.String())
}

func (o *Orchestrator) appendPhaseSummary(
	runID string,
	phase string,
	startedAt time.Time,
	finishedAt time.Time,
	output string,
) error {
	if o.Store == nil {
		return nil
	}
	var b strings.Builder
	b.WriteString(fmt.Sprintf("## Phase %s — %s\n", phase, runID))
	if !startedAt.IsZero() {
		b.WriteString(fmt.Sprintf("- Started: %s\n", startedAt.UTC().Format(time.RFC3339)))
	}
	if !finishedAt.IsZero() {
		b.WriteString(fmt.Sprintf("- Finished: %s\n", finishedAt.UTC().Format(time.RFC3339)))
	}
	b.WriteString(fmt.Sprintf("- Output bytes: %d\n", len(output)))
	if output != "" {
		b.WriteString(fmt.Sprintf("- Artifact: %s\n", fmt.Sprintf("%s.txt", phase)))
	}
	return o.Store.AppendMemoryDoc("PROGRESS.md", b.String())
}

func (o *Orchestrator) heartbeatLoop(ctx context.Context, runID string, pid int) {
	if o.Store == nil {
		return
	}
	interval := 30 * time.Second
	if o.Config.Loop.StopConditions.MaxHeartbeatSeconds > 0 {
		interval = time.Duration(o.Config.Loop.StopConditions.MaxHeartbeatSeconds/2) * time.Second
		if interval < 15*time.Second {
			interval = 15 * time.Second
		}
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		if err := o.Store.TouchRunHeartbeat(runID, pid); err != nil && o.Logger != nil {
			o.Logger.Warn("heartbeat update failed", "run_id", runID, "error", err.Error())
		}
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

func (o *Orchestrator) gatherFeedback(runID string) (string, state.RunFeedback, error) {
	cfg := o.Config.Loop.Feedback
	if cfg.Mode != "on" {
		return "", state.RunFeedback{}, nil
	}

	now := time.Now().UTC()
	remaining := cfg.MaxBytes
	limitEnabled := remaining > 0

	var b strings.Builder
	appendWithLimit := func(text string) bool {
		if text == "" {
			return true
		}
		if !limitEnabled {
			b.WriteString(text)
			return true
		}
		if remaining <= 0 {
			return false
		}
		if len(text) > remaining {
			text = text[:remaining]
		}
		b.WriteString(text)
		remaining -= len(text)
		return remaining > 0
	}

	meta := state.RunFeedback{
		RunID:     runID,
		UpdatedAt: now,
		Sources:   cfg.Sources,
	}

	sourceEnabled := func(name string) bool {
		for _, s := range cfg.Sources {
			if s == name {
				return true
			}
		}
		return false
	}

	if sourceEnabled("memory") {
		docs, err := o.Store.ListMemoryDocs()
		if err != nil {
			return "", meta, err
		}
		for _, doc := range docs {
			if cfg.MemoryGlob != "" {
				matched, err := filepath.Match(cfg.MemoryGlob, doc.Name)
				if err != nil || !matched {
					continue
				}
			}
			detail, err := o.Store.GetMemoryDoc(doc.Name)
			if err != nil {
				continue
			}
			meta.MemoryDocs = append(meta.MemoryDocs, detail.Name)
			if !appendWithLimit(fmt.Sprintf("## Memory: %s\n%s\n", detail.Name, detail.Content)) {
				break
			}
		}
	}

	if sourceEnabled("events") {
		events, err := o.Store.ListEvents(runID)
		if err != nil {
			return "", meta, err
		}
		if cfg.MaxEvents > 0 && len(events) > cfg.MaxEvents {
			events = events[len(events)-cfg.MaxEvents:]
		}
		if len(events) > 0 {
			if !appendWithLimit("## Recent Events\n") {
				return b.String(), meta, nil
			}
			for _, event := range events {
				meta.EventIDs = append(meta.EventIDs, event.ID)
				line := fmt.Sprintf("- %s [%s] %s %s\n", event.TS.Format(time.RFC3339), event.Type, event.Phase, event.Message)
				if !appendWithLimit(line) {
					break
				}
			}
		}
	}

	if sourceEnabled("artifacts") {
		artifacts, err := o.Store.ListArtifacts(runID)
		if err != nil {
			return "", meta, err
		}
		sort.Slice(artifacts, func(i, j int) bool {
			return artifacts[i].CreatedAt.Before(artifacts[j].CreatedAt)
		})
		if cfg.MaxArtifacts > 0 && len(artifacts) > cfg.MaxArtifacts {
			artifacts = artifacts[len(artifacts)-cfg.MaxArtifacts:]
		}
		if len(artifacts) > 0 {
			if !appendWithLimit("## Recent Artifacts\n") {
				return b.String(), meta, nil
			}
			for _, artifact := range artifacts {
				meta.ArtifactIDs = append(meta.ArtifactIDs, artifact.ID)
				line := fmt.Sprintf("- %s (%s, %d bytes)\n", artifact.Name, artifact.Type, artifact.SizeBytes)
				if !appendWithLimit(line) {
					break
				}
			}
		}
	}

	meta.Bytes = b.Len()
	meta.LastPromptSummary = feedbackSummary(meta)
	return b.String(), meta, nil
}

func feedbackSummary(meta state.RunFeedback) string {
	return fmt.Sprintf("memory_docs=%d events=%d artifacts=%d bytes=%d", len(meta.MemoryDocs), len(meta.EventIDs), len(meta.ArtifactIDs), meta.Bytes)
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
