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

	if sourceEnabled("snapshot") && strings.TrimSpace(cfg.SnapshotPath) != "" {
		content, err := os.ReadFile(cfg.SnapshotPath)
		if err == nil {
			meta.SnapshotPath = cfg.SnapshotPath
			if !appendWithLimit("## Resume Snapshot\n") {
				return b.String(), meta, nil
			}
			if !appendWithLimit(string(content)) {
				return b.String(), meta, nil
			}
			appendWithLimit("\n")
		}
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
