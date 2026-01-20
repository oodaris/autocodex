package orchestrator

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/oodaris/autocodex/internal/state"
)

func (o *Orchestrator) writeNamedArtifact(runID, name, output string) error {
	if output == "" {
		return nil
	}
	path := filepath.Join(o.Store.RunsDir, runID, "artifacts", name)
	return os.WriteFile(path, []byte(output), 0o644)
}

func (o *Orchestrator) writeArtifact(runID, phase, output string) error {
	return o.writeNamedArtifact(runID, fmt.Sprintf("%s.txt", phase), output)
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
