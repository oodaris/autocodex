package autonomy

import (
	"encoding/json"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/oodaris/autocodex/internal/config"
)

type fixAttemptTracker struct {
	mu      sync.Mutex
	attempt map[string]int
	scope   string
	path    string
	logger  *slog.Logger
}

func newFixAttemptTracker(cfg config.Config, logger *slog.Logger) *fixAttemptTracker {
	scope := strings.TrimSpace(cfg.Autonomy.Gate.MaxFixAttemptsScope)
	if scope == "" {
		scope = "global"
	}
	storePath := strings.TrimSpace(cfg.Autonomy.Gate.FixAttemptsStore)
	if storePath != "" && !filepath.IsAbs(storePath) {
		storePath = filepath.Join(cfg.StateDir(), storePath)
	}
	tracker := &fixAttemptTracker{
		attempt: map[string]int{},
		scope:   scope,
		path:    storePath,
		logger:  logger,
	}
	if tracker.scope == "global" {
		tracker.load()
	}
	return tracker
}

func (t *fixAttemptTracker) Increment(id string) int {
	id = sanitizeBeadID(id)
	if id == "" {
		return 0
	}
	t.mu.Lock()
	defer t.mu.Unlock()
	t.attempt[id]++
	if t.scope == "global" {
		t.persistLocked()
	}
	return t.attempt[id]
}

func (t *fixAttemptTracker) Reset(id string) {
	id = sanitizeBeadID(id)
	if id == "" {
		return
	}
	t.mu.Lock()
	defer t.mu.Unlock()
	delete(t.attempt, id)
	if t.scope == "global" {
		t.persistLocked()
	}
}

func (t *fixAttemptTracker) load() {
	if strings.TrimSpace(t.path) == "" {
		return
	}
	content, err := os.ReadFile(t.path)
	if err != nil {
		return
	}
	var payload map[string]int
	if err := json.Unmarshal(content, &payload); err != nil {
		if t.logger != nil {
			t.logger.Warn("autonomy fix attempts parse failed", "path", t.path, "error", err.Error())
		}
		return
	}
	for key, value := range payload {
		cleaned := sanitizeBeadID(key)
		if cleaned == "" || value <= 0 {
			continue
		}
		t.attempt[cleaned] = value
	}
}

func (t *fixAttemptTracker) persistLocked() {
	if strings.TrimSpace(t.path) == "" {
		return
	}
	if err := os.MkdirAll(filepath.Dir(t.path), 0o755); err != nil {
		if t.logger != nil {
			t.logger.Warn("autonomy fix attempts mkdir failed", "path", t.path, "error", err.Error())
		}
		return
	}
	content, err := json.MarshalIndent(t.attempt, "", "  ")
	if err != nil {
		if t.logger != nil {
			t.logger.Warn("autonomy fix attempts marshal failed", "path", t.path, "error", err.Error())
		}
		return
	}
	if err := os.WriteFile(t.path, append(content, '\n'), 0o644); err != nil {
		if t.logger != nil {
			t.logger.Warn("autonomy fix attempts write failed", "path", t.path, "error", err.Error())
		}
	}
}
