package orchestrator

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strconv"
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
	if o.Logger == nil {
		o.Logger = slog.New(slog.NewJSONHandler(io.Discard, &slog.HandlerOptions{}))
	}

	run, err := o.Store.CreateRun()
	if err != nil {
		return nil, err
	}
	run.Model = o.Config.Codex.Model
	run.Reasoning = o.Config.Codex.ReasoningEffort
	run.CollabMode = o.Config.Codex.CollaborationMode
	run.CollabPreset = o.Config.Codex.Preset
	run.CodexCLI = o.Config.Codex.CLIPath
	run.Mode = o.Config.Mode
	if err := o.Store.SaveRun(run); err != nil {
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
	beadID := beadIDFromContext(ctx)
	if beadID == "" {
		beadID = os.Getenv("AUTOCODEX_BEAD_ID")
	}
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
	runLogger := logger.With("stage", "run")
	idempotencyKey := run.ID
	if beadID != "" {
		idempotencyKey = fmt.Sprintf("%s:%s", run.ID, beadID)
	}
	buildEventMeta := func(queueState, admissionDecision, phase, turnID, itemID string, attempt, retryBackoffMS int) map[string]string {
		meta := map[string]string{
			"trace_id":           traceID,
			"bead_id":            beadID,
			"run_id":             run.ID,
			"idempotency_key":    idempotencyKey,
			"attempt":            strconv.Itoa(attempt),
			"admission_decision": admissionDecision,
			"queue_state":        queueState,
			"retry_backoff_ms":   strconv.Itoa(retryBackoffMS),
			"thread_id":          run.ID,
			"turn_id":            turnID,
			"item_id":            itemID,
		}
		if phase != "" {
			meta["phase"] = phase
		}
		return meta
	}

	runLogger.Info("run started", "status", "started", "latency_ms", 0)
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
		Meta: func() map[string]string {
			meta := buildEventMeta("running", "admit", "run", "run-start", "run-start", 0, 0)
			meta["model"] = o.Config.Codex.Model
			meta["reasoning"] = o.Config.Codex.ReasoningEffort
			meta["collab_mode"] = o.Config.Codex.CollaborationMode
			meta["collab_preset"] = o.Config.Codex.Preset
			meta["codex_cli"] = o.Config.Codex.CLIPath
			meta["autocodex_mode"] = o.Config.Mode
			return meta
		}(),
	})
	_ = o.Store.AppendEvent(state.RunEvent{
		ID:      eventID("thread-started"),
		RunID:   run.ID,
		TS:      time.Now().UTC(),
		Type:    "thread_started",
		Message: "thread started",
		Meta:    buildEventMeta("running", "admit", "run", "run-start", "run-start", 0, 0),
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
				runLogger.Info("run stopped", "status", status, "reason", reason, "latency_ms", 0)
				stopTurnID := fmt.Sprintf("%s-turn-%d", run.ID, run.Iterations)
				stopItemID := fmt.Sprintf("%s-item-stop", stopTurnID)
				_ = o.Store.AppendEvent(state.RunEvent{
					ID:      eventID("run-stopped"),
					RunID:   run.ID,
					TS:      time.Now().UTC(),
					Type:    "run_stopped",
					Message: reason,
					Meta:    buildEventMeta(status, "admit", "run", stopTurnID, stopItemID, run.Iterations, 0),
				})
				threadEventType := "thread_failed"
				threadMessage := "thread failed"
				if status == "completed" {
					threadEventType = "thread_completed"
					threadMessage = "thread completed"
				}
				_ = o.Store.AppendEvent(state.RunEvent{
					ID:      eventID(threadEventType),
					RunID:   run.ID,
					TS:      time.Now().UTC(),
					Type:    threadEventType,
					Message: threadMessage,
					Meta:    buildEventMeta(status, "admit", "run", stopTurnID, stopItemID, run.Iterations, 0),
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
			turnID := fmt.Sprintf("%s-turn-%d", run.ID, run.Iterations)
			itemID := fmt.Sprintf("%s-item-%s", turnID, phase)
			eventMeta := buildEventMeta("running", "admit", phase, turnID, itemID, run.Iterations, 0)
			phaseLogger := logger.With("stage", phase)
			phaseLogger.Info("phase start", "phase", phase, "status", "started", "latency_ms", 0)
			_ = o.Store.AppendEvent(state.RunEvent{
				ID:      eventID("phase-start"),
				RunID:   run.ID,
				TS:      time.Now().UTC(),
				Type:    "phase_start",
				Phase:   phase,
				Message: "phase started",
				Meta:    eventMeta,
			})
			_ = o.Store.AppendEvent(state.RunEvent{
				ID:      eventID("turn-started"),
				RunID:   run.ID,
				TS:      time.Now().UTC(),
				Type:    "turn_started",
				Phase:   phase,
				Message: "turn started",
				Meta:    eventMeta,
			})
			_ = o.Store.AppendEvent(state.RunEvent{
				ID:      eventID("item-started"),
				RunID:   run.ID,
				TS:      time.Now().UTC(),
				Type:    "item_started",
				Phase:   phase,
				Message: "item started",
				Meta:    eventMeta,
			})

			feedbackText, feedbackMeta, err := o.gatherFeedback(run.ID)
			if err != nil {
				phaseLogger.Warn("feedback gather failed", "phase", phase, "error", err.Error())
			} else if feedbackMeta.RunID != "" {
				_ = o.Store.SaveRunFeedback(feedbackMeta)
			}

			prompt := o.buildPrompt(phase, run.ID, feedbackText)
			promptBytes := len(prompt)
			feedbackBytes := 0
			feedbackSummaryText := ""
			if feedbackMeta.RunID != "" {
				feedbackBytes = feedbackMeta.Bytes
				feedbackSummaryText = feedbackSummary(feedbackMeta)
			}
			metrics := fmt.Sprintf(
				"prompt_bytes=%d\nfeedback_bytes=%d\nfeedback_summary=%s\n",
				promptBytes,
				feedbackBytes,
				feedbackSummaryText,
			)
			if err := o.writeNamedArtifact(run.ID, fmt.Sprintf("%s-prompt-metrics.txt", phase), metrics); err != nil {
				phaseLogger.Warn("prompt metrics write failed", "phase", phase, "error", err.Error())
			}
			phaseLogger.Info(
				"phase prompt metrics",
				"phase", phase,
				"prompt_bytes", promptBytes,
				"feedback_bytes", feedbackBytes,
			)
			if phase == "review" && o.Config.Loop.PromptGuardrails.ReviewMaxBytes > 0 &&
				promptBytes > o.Config.Loop.PromptGuardrails.ReviewMaxBytes {
				skipMsg := fmt.Sprintf(
					"review skipped: prompt_bytes=%d exceeds review_max_bytes=%d\n",
					promptBytes,
					o.Config.Loop.PromptGuardrails.ReviewMaxBytes,
				)
				if err := o.writeNamedArtifact(run.ID, "review-skipped.txt", skipMsg); err != nil {
					phaseLogger.Warn("review skipped artifact write failed", "phase", phase, "error", err.Error())
				}
				phaseFinished := time.Now().UTC()
				if err := o.appendPhaseSummary(run.ID, phase, phaseStart, phaseFinished, skipMsg); err != nil {
					phaseLogger.Warn("phase summary append failed", "phase", phase, "error", err.Error())
				}
				if feedbackMeta.RunID != "" {
					feedbackMeta.LastOutputSummary = fmt.Sprintf("phase %s output bytes=%d", phase, len(skipMsg))
					feedbackMeta.UpdatedAt = time.Now().UTC()
					_ = o.Store.SaveRunFeedback(feedbackMeta)
				}
				latency := time.Since(phaseStart).Milliseconds()
				phaseLogger.Info("phase complete", "phase", phase, "status", "skipped", "latency_ms", latency)
				_ = o.Store.AppendEvent(state.RunEvent{
					ID:      eventID("phase-complete"),
					RunID:   run.ID,
					TS:      time.Now().UTC(),
					Type:    "phase_complete",
					Phase:   phase,
					Message: "phase skipped",
					Meta:    eventMeta,
				})
				_ = o.Store.AppendEvent(state.RunEvent{
					ID:      eventID("item-completed"),
					RunID:   run.ID,
					TS:      time.Now().UTC(),
					Type:    "item_completed",
					Phase:   phase,
					Message: "item completed (skipped)",
					Meta:    eventMeta,
				})
				_ = o.Store.AppendEvent(state.RunEvent{
					ID:      eventID("turn-completed"),
					RunID:   run.ID,
					TS:      time.Now().UTC(),
					Type:    "turn_completed",
					Phase:   phase,
					Message: "turn completed (skipped)",
					Meta:    eventMeta,
				})
				lastProgress = time.Now().UTC()
				continue
			}
			phaseCtx, cancel := context.WithTimeout(ctx, time.Duration(o.Config.Codex.TimeoutSeconds)*time.Second)
			idleSeconds := o.Config.Loop.StopConditions.MaxIdleSeconds
			if override, ok := o.Config.Loop.PhaseIdleSecs[phase]; ok && override > 0 {
				idleSeconds = override
			}
			if idleSeconds > 0 {
				phaseCtx = codex.WithIdleTimeout(phaseCtx, time.Duration(idleSeconds)*time.Second)
			}
			outputPath := ""
			if o.Config.Codex.OutputLast {
				outputPath = filepath.Join(o.Store.RunsDir, run.ID, "artifacts", fmt.Sprintf("%s-final.txt", phase))
				phaseCtx = codex.WithOutputPath(phaseCtx, outputPath)
			}
			artifactsDir := filepath.Join(o.Store.RunsDir, run.ID, "artifacts")
			_ = os.MkdirAll(artifactsDir, 0o755)
			stdoutPath := filepath.Join(artifactsDir, fmt.Sprintf("%s-stdout.txt", phase))
			stderrPath := filepath.Join(artifactsDir, fmt.Sprintf("%s-stderr.txt", phase))
			var stdoutFile *os.File
			var stderrFile *os.File
			streamStderr := false
			if file, err := os.Create(stdoutPath); err == nil {
				stdoutFile = file
			} else {
				phaseLogger.Warn("stdout stream open failed", "phase", phase, "error", err.Error())
			}
			if file, err := os.Create(stderrPath); err == nil {
				stderrFile = file
				streamStderr = true
			} else {
				phaseLogger.Warn("stderr stream open failed", "phase", phase, "error", err.Error())
			}
			if stdoutFile != nil || stderrFile != nil {
				phaseCtx = codex.WithOutputSinks(phaseCtx, stdoutFile, stderrFile)
			}
			phaseCtx = codex.WithPIDReporter(phaseCtx, func(pid int) {
				if err := o.Store.SetRunChildPID(run.ID, pid); err != nil && phaseLogger != nil {
					phaseLogger.Warn("child pid update failed", "phase", phase, "error", err.Error())
				}
			})
			result, execErr := o.Codex.Exec(phaseCtx, prompt)
			cancel()
			if stdoutFile != nil {
				_ = stdoutFile.Close()
			}
			if stderrFile != nil {
				_ = stderrFile.Close()
			}
			if err := o.Store.ClearRunChildPID(run.ID); err != nil && phaseLogger != nil {
				phaseLogger.Warn("child pid clear failed", "phase", phase, "error", err.Error())
			}
			if execErr != nil && strings.Contains(execErr.Error(), "requires follow-up") && strings.TrimSpace(result.Stdout) != "" {
				phaseLogger.Warn("phase requested follow-up, using available output", "phase", phase)
				execErr = nil
			}
			if strings.TrimSpace(result.Stderr) != "" && !streamStderr {
				if err := o.writeNamedArtifact(run.ID, fmt.Sprintf("%s-stderr.txt", phase), result.Stderr); err != nil {
					phaseLogger.Warn("stderr artifact write failed", "phase", phase, "error", err.Error())
				}
			}
			rawOutput := result.Stdout
			finalOutput := rawOutput
			if outputPath != "" {
				if data, err := os.ReadFile(outputPath); err == nil && strings.TrimSpace(string(data)) != "" {
					finalOutput = string(data)
				}
			}
			if o.Config.Codex.JSONOutput && strings.TrimSpace(rawOutput) != "" {
				if err := o.writeNamedArtifact(run.ID, fmt.Sprintf("%s-jsonl.txt", phase), rawOutput); err != nil {
					phaseLogger.Warn("jsonl artifact write failed", "phase", phase, "error", err.Error())
				}
			}
			if execErr != nil {
				latency := time.Since(phaseStart).Milliseconds()
				phaseLogger.Error("phase failed", "phase", phase, "error", execErr.Error(), "status", "failed", "latency_ms", latency)
				_ = o.Store.AppendEvent(state.RunEvent{
					ID:      eventID("phase-failed"),
					RunID:   run.ID,
					TS:      time.Now().UTC(),
					Type:    "phase_failed",
					Phase:   phase,
					Message: execErr.Error(),
					Meta:    eventMeta,
				})
				_ = o.Store.AppendEvent(state.RunEvent{
					ID:      eventID("item-completed-failed"),
					RunID:   run.ID,
					TS:      time.Now().UTC(),
					Type:    "item_completed",
					Phase:   phase,
					Message: "item completed (failed)",
					Meta:    eventMeta,
				})
				_ = o.Store.AppendEvent(state.RunEvent{
					ID:      eventID("turn-completed-failed"),
					RunID:   run.ID,
					TS:      time.Now().UTC(),
					Type:    "turn_completed",
					Phase:   phase,
					Message: "turn completed (failed)",
					Meta:    eventMeta,
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
				run.StopReason = strPtr("phase_failed")
				run.LastError = &errMsg
				run.LastAction = nil
				finished := time.Now().UTC()
				run.FinishedAt = &finished
				run.LastActionAt = nil
				_ = o.Store.SaveRun(run)
				_ = o.Store.AppendEvent(state.RunEvent{
					ID:      eventID("thread-failed"),
					RunID:   run.ID,
					TS:      time.Now().UTC(),
					Type:    "thread_failed",
					Message: "thread failed",
					Meta:    buildEventMeta("failed", "admit", phase, turnID, itemID, run.Iterations, 0),
				})
				return run, execErr
			}
			consecutiveFailures = 0

			if err := o.writeArtifact(run.ID, phase, finalOutput); err != nil {
				phaseLogger.Warn("artifact write failed", "phase", phase, "error", err.Error())
			}
			phaseFinished := time.Now().UTC()
			if err := o.appendPhaseSummary(run.ID, phase, phaseStart, phaseFinished, finalOutput); err != nil {
				phaseLogger.Warn("phase summary append failed", "phase", phase, "error", err.Error())
			}
			if feedbackMeta.RunID != "" {
				feedbackMeta.LastOutputSummary = fmt.Sprintf("phase %s output bytes=%d", phase, len(finalOutput))
				feedbackMeta.UpdatedAt = time.Now().UTC()
				_ = o.Store.SaveRunFeedback(feedbackMeta)
			}

			latency := time.Since(phaseStart).Milliseconds()
			phaseLogger.Info("phase complete", "phase", phase, "status", "completed", "latency_ms", latency)
			_ = o.Store.AppendEvent(state.RunEvent{
				ID:      eventID("phase-complete"),
				RunID:   run.ID,
				TS:      time.Now().UTC(),
				Type:    "phase_complete",
				Phase:   phase,
				Message: "phase completed",
				Meta:    eventMeta,
			})
			_ = o.Store.AppendEvent(state.RunEvent{
				ID:      eventID("item-completed"),
				RunID:   run.ID,
				TS:      time.Now().UTC(),
				Type:    "item_completed",
				Phase:   phase,
				Message: "item completed",
				Meta:    eventMeta,
			})
			_ = o.Store.AppendEvent(state.RunEvent{
				ID:      eventID("turn-completed"),
				RunID:   run.ID,
				TS:      time.Now().UTC(),
				Type:    "turn_completed",
				Phase:   phase,
				Message: "turn completed",
				Meta:    eventMeta,
			})
			lastProgress = time.Now().UTC()
		}

		if loopMode != "continuous" {
			break
		}
	}

	o.finalizeRun(run, "completed", nil, nil, nil)
	runLogger.Info("run completed", "status", "completed", "latency_ms", 0)
	completionTurnID := fmt.Sprintf("%s-turn-%d", run.ID, run.Iterations)
	completionItemID := fmt.Sprintf("%s-item-run-complete", completionTurnID)
	completionMeta := buildEventMeta("completed", "admit", "run", completionTurnID, completionItemID, run.Iterations, 0)
	_ = o.Store.AppendEvent(state.RunEvent{
		ID:      eventID("thread-completed"),
		RunID:   run.ID,
		TS:      time.Now().UTC(),
		Type:    "thread_completed",
		Message: "thread completed",
		Meta:    completionMeta,
	})
	_ = o.Store.AppendEvent(state.RunEvent{
		ID:      eventID("run-complete"),
		RunID:   run.ID,
		TS:      time.Now().UTC(),
		Type:    "run_complete",
		Message: "run completed",
		Meta:    completionMeta,
	})

	return run, nil
}

func strPtr(value string) *string {
	return &value
}
