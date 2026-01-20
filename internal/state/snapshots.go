package state

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

type SnapshotSummary struct {
	ID          string    `json:"id"`
	RunID       string    `json:"run_id"`
	CreatedAt   time.Time `json:"created_at"`
	Reason      string    `json:"reason"`
	SizeBytes   int64     `json:"size_bytes"`
	ContentPath string    `json:"content_path"`
}

type SnapshotManifest struct {
	Events     int `json:"events"`
	Artifacts  int `json:"artifacts"`
	MemoryDocs int `json:"memory_docs"`
	Bytes      int `json:"bytes"`
}

type SnapshotDetail struct {
	Summary  SnapshotSummary  `json:"summary"`
	Manifest SnapshotManifest `json:"manifest"`
	Content  string           `json:"content"`
}

type SnapshotOptions struct {
	Reason       string
	Sources      []string
	MaxBytes     int
	MaxEvents    int
	MaxArtifacts int
	MemoryGlob   string
}

type snapshotRecord struct {
	Summary  SnapshotSummary  `json:"summary"`
	Manifest SnapshotManifest `json:"manifest"`
}

func (s *Store) CreateSnapshot(runID string, opts SnapshotOptions) (SnapshotDetail, error) {
	if runID == "" {
		return SnapshotDetail{}, errors.New("run id required")
	}
	if _, err := s.GetRun(runID); err != nil {
		return SnapshotDetail{}, err
	}

	if opts.MaxBytes < 0 {
		return SnapshotDetail{}, errors.New("max bytes must be >= 0")
	}
	if opts.MaxEvents < 0 || opts.MaxArtifacts < 0 {
		return SnapshotDetail{}, errors.New("max events/artifacts must be >= 0")
	}
	if len(opts.Sources) == 0 {
		opts.Sources = []string{"memory", "events", "artifacts"}
	}
	if opts.MemoryGlob == "" {
		opts.MemoryGlob = "*.md"
	}

	createdAt := time.Now().UTC()
	snapshotID := fmt.Sprintf("%s-%s", createdAt.Format("20060102T150405Z"), randSuffix(3))
	snapshotsDir := s.snapshotsDir(runID)
	if err := os.MkdirAll(snapshotsDir, 0o755); err != nil {
		return SnapshotDetail{}, fmt.Errorf("create snapshots dir: %w", err)
	}
	contentPath := s.snapshotContentPath(runID, snapshotID)
	recordPath := s.snapshotRecordPath(runID, snapshotID)

	var b strings.Builder
	appendWithLimit := newLimitWriter(&b, opts.MaxBytes)
	appendWithLimit(fmt.Sprintf("# autocodex Snapshot\nRun: %s\nCreated: %s\n", runID, createdAt.Format(time.RFC3339)))
	if opts.Reason != "" {
		appendWithLimit(fmt.Sprintf("Reason: %s\n", opts.Reason))
	}
	appendWithLimit("\n")

	manifest := SnapshotManifest{}

	if sourceEnabled(opts.Sources, "memory") {
		docs, err := s.ListMemoryDocs()
		if err != nil {
			return SnapshotDetail{}, err
		}
		for _, doc := range docs {
			matched, err := filepath.Match(opts.MemoryGlob, doc.Name)
			if err != nil || !matched {
				continue
			}
			detail, err := s.GetMemoryDoc(doc.Name)
			if err != nil {
				continue
			}
			manifest.MemoryDocs++
			if !appendWithLimit(fmt.Sprintf("## Memory: %s\n%s\n\n", detail.Name, detail.Content)) {
				break
			}
		}
	}

	if sourceEnabled(opts.Sources, "events") {
		events, err := s.ListEvents(runID)
		if err != nil {
			return SnapshotDetail{}, err
		}
		sort.Slice(events, func(i, j int) bool {
			if events[i].TS.Equal(events[j].TS) {
				return events[i].ID < events[j].ID
			}
			return events[i].TS.Before(events[j].TS)
		})
		if opts.MaxEvents > 0 && len(events) > opts.MaxEvents {
			events = events[len(events)-opts.MaxEvents:]
		}
		if len(events) > 0 {
			appendWithLimit("## Recent Events\n")
			for _, event := range events {
				manifest.Events++
				line := fmt.Sprintf("- %s [%s] %s %s\n", event.TS.Format(time.RFC3339), event.Type, event.Phase, event.Message)
				if !appendWithLimit(line) {
					break
				}
			}
			appendWithLimit("\n")
		}
	}

	if sourceEnabled(opts.Sources, "artifacts") {
		artifacts, err := s.ListArtifacts(runID)
		if err != nil {
			return SnapshotDetail{}, err
		}
		sort.Slice(artifacts, func(i, j int) bool {
			if artifacts[i].CreatedAt.Equal(artifacts[j].CreatedAt) {
				return artifacts[i].Name < artifacts[j].Name
			}
			return artifacts[i].CreatedAt.Before(artifacts[j].CreatedAt)
		})
		if opts.MaxArtifacts > 0 && len(artifacts) > opts.MaxArtifacts {
			artifacts = artifacts[len(artifacts)-opts.MaxArtifacts:]
		}
		if len(artifacts) > 0 {
			appendWithLimit("## Recent Artifacts\n")
			for _, artifact := range artifacts {
				manifest.Artifacts++
				line := fmt.Sprintf("- %s (%s, %d bytes)\n", artifact.Name, artifact.Type, artifact.SizeBytes)
				if !appendWithLimit(line) {
					break
				}
			}
			appendWithLimit("\n")
		}
		for _, artifact := range artifacts {
			if !isTextArtifact(artifact.Name) {
				continue
			}
			content, err := os.ReadFile(artifact.Path)
			if err != nil {
				continue
			}
			if !appendWithLimit(fmt.Sprintf("### Artifact: %s\n", artifact.Name)) {
				break
			}
			if !appendWithLimit("```\n") {
				break
			}
			if !appendWithLimit(string(content)) {
				appendWithLimit("\n```\n")
				break
			}
			appendWithLimit("\n```\n")
		}
	}

	content := b.String()
	if err := os.WriteFile(contentPath, []byte(content), 0o644); err != nil {
		return SnapshotDetail{}, fmt.Errorf("write snapshot content: %w", err)
	}

	summary := SnapshotSummary{
		ID:          snapshotID,
		RunID:       runID,
		CreatedAt:   createdAt,
		Reason:      opts.Reason,
		SizeBytes:   int64(len(content)),
		ContentPath: contentPath,
	}
	manifest.Bytes = len(content)
	record := snapshotRecord{Summary: summary, Manifest: manifest}
	data, err := json.MarshalIndent(record, "", "  ")
	if err != nil {
		return SnapshotDetail{}, fmt.Errorf("marshal snapshot record: %w", err)
	}
	if err := os.WriteFile(recordPath, data, 0o644); err != nil {
		return SnapshotDetail{}, fmt.Errorf("write snapshot record: %w", err)
	}
	return SnapshotDetail{
		Summary:  summary,
		Manifest: manifest,
		Content:  content,
	}, nil
}

func (s *Store) ListSnapshots(runID string) ([]SnapshotSummary, error) {
	dir := s.snapshotsDir(runID)
	entries, err := os.ReadDir(dir)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return []SnapshotSummary{}, nil
		}
		return nil, fmt.Errorf("read snapshots dir: %w", err)
	}
	summaries := make([]SnapshotSummary, 0)
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
			continue
		}
		data, err := os.ReadFile(filepath.Join(dir, entry.Name()))
		if err != nil {
			continue
		}
		var record snapshotRecord
		if err := json.Unmarshal(data, &record); err != nil {
			continue
		}
		summaries = append(summaries, record.Summary)
	}
	sort.Slice(summaries, func(i, j int) bool {
		return summaries[i].CreatedAt.Before(summaries[j].CreatedAt)
	})
	return summaries, nil
}

func (s *Store) GetSnapshot(runID, snapshotID string) (SnapshotDetail, error) {
	if runID == "" || snapshotID == "" {
		return SnapshotDetail{}, errors.New("run id and snapshot id required")
	}
	recordPath := s.snapshotRecordPath(runID, snapshotID)
	data, err := os.ReadFile(recordPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return SnapshotDetail{}, fmt.Errorf("snapshot not found")
		}
		return SnapshotDetail{}, fmt.Errorf("read snapshot record: %w", err)
	}
	var record snapshotRecord
	if err := json.Unmarshal(data, &record); err != nil {
		return SnapshotDetail{}, fmt.Errorf("parse snapshot record: %w", err)
	}
	content, err := os.ReadFile(record.Summary.ContentPath)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return SnapshotDetail{}, fmt.Errorf("read snapshot content: %w", err)
	}
	return SnapshotDetail{
		Summary:  record.Summary,
		Manifest: record.Manifest,
		Content:  string(content),
	}, nil
}

func sourceEnabled(sources []string, name string) bool {
	for _, source := range sources {
		if source == name {
			return true
		}
	}
	return false
}

func newLimitWriter(builder *strings.Builder, maxBytes int) func(text string) bool {
	if maxBytes <= 0 {
		return func(text string) bool {
			if text == "" {
				return true
			}
			builder.WriteString(text)
			return true
		}
	}
	remaining := maxBytes
	return func(text string) bool {
		if text == "" {
			return true
		}
		if remaining <= 0 {
			return false
		}
		if len(text) > remaining {
			text = text[:remaining]
		}
		builder.WriteString(text)
		remaining -= len(text)
		return remaining > 0
	}
}
