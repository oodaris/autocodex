package autonomy

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"sort"
	"strings"
)

type ReadyBead struct {
	ID       string `json:"id"`
	Title    string `json:"title"`
	Status   string `json:"status"`
	Priority int    `json:"priority"`
}

var ErrNoRunScopedReadyBeads = errors.New("no run-scoped ready beads")

func (c *Controller) selectBead(preferredID string, scopeIDs map[string]bool) (*ReadyBead, error) {
	ready, _, err := listReadyBeadsWithScope(
		scopeIDs,
		c.Config.Autonomy.Coordinator.SelectionMode,
		c.Config.Autonomy.Coordinator.AllowAllReadyFallback,
	)
	if err != nil {
		return nil, err
	}
	if len(ready) == 0 {
		if c.Logger != nil {
			c.Logger.Info("no ready beads")
		}
		return nil, nil
	}
	bead := ready[0]
	preferredID = sanitizeBeadID(preferredID)
	if preferredID != "" {
		for _, candidate := range ready {
			if candidate.ID == preferredID {
				bead = candidate
				break
			}
		}
	}
	if c.Config.Beads.AutoUpdate {
		if err := markBeadInProgress(bead.ID); err != nil {
			return nil, err
		}
	}
	if c.Logger != nil {
		c.Logger.Info("bead selected", "bead_id", bead.ID, "title", bead.Title)
	}
	return &bead, nil
}

func listReadyBeads() ([]ReadyBead, error) {
	output, err := runBD("ready", "--json")
	if err != nil {
		return nil, err
	}
	var beads []ReadyBead
	if err := json.Unmarshal([]byte(output), &beads); err != nil {
		return nil, fmt.Errorf("parse bd ready output: %w", err)
	}
	sort.SliceStable(beads, func(i, j int) bool {
		if beads[i].Priority == beads[j].Priority {
			return beads[i].ID < beads[j].ID
		}
		return beads[i].Priority < beads[j].Priority
	})
	return beads, nil
}

func listReadyBeadsWithScope(scopeIDs map[string]bool, selectionMode string, allowFallback bool) ([]ReadyBead, int, error) {
	allReady, err := listReadyBeads()
	if err != nil {
		return nil, 0, err
	}
	if len(allReady) == 0 {
		return nil, 0, nil
	}
	if strings.TrimSpace(selectionMode) == "" {
		selectionMode = "run_scoped"
	}
	if selectionMode == "all_ready" || len(scopeIDs) == 0 {
		return allReady, len(allReady), nil
	}
	scoped := make([]ReadyBead, 0, len(allReady))
	for _, bead := range allReady {
		if scopeIDs[sanitizeBeadID(bead.ID)] {
			scoped = append(scoped, bead)
		}
	}
	if len(scoped) > 0 {
		return scoped, len(allReady), nil
	}
	if allowFallback {
		return allReady, len(allReady), nil
	}
	return nil, len(allReady), fmt.Errorf(
		"%w: selection_mode=run_scoped and no ready beads match current run scope (ready_total=%d); set autonomy.coordinator.allow_all_ready_fallback=true or use --bead-scope all_ready to override",
		ErrNoRunScopedReadyBeads,
		len(allReady),
	)
}

func buildBeadScopeFromTasks(tasks []Task) map[string]bool {
	if len(tasks) == 0 {
		return nil
	}
	scope := make(map[string]bool, len(tasks))
	for _, task := range tasks {
		id := sanitizeBeadID(task.ID)
		if id == "" {
			continue
		}
		scope[id] = true
	}
	if len(scope) == 0 {
		return nil
	}
	return scope
}

func buildBeadScopeFromTasksPath(tasksPath string) (map[string]bool, error) {
	tasksPath = strings.TrimSpace(tasksPath)
	if tasksPath == "" {
		return nil, nil
	}
	content, err := os.ReadFile(tasksPath)
	if err != nil {
		return nil, fmt.Errorf("read tasks file for bead scope: %w", err)
	}
	var tasksFile TasksFile
	if err := json.Unmarshal(content, &tasksFile); err != nil {
		return nil, fmt.Errorf("parse tasks file for bead scope: %w", err)
	}
	return buildBeadScopeFromTasks(tasksFile.Tasks), nil
}

func filterReadyBeadsBySelectors(beads []ReadyBead, ids []string, prefix string) []ReadyBead {
	idSet := map[string]bool{}
	for _, id := range ids {
		cleaned := sanitizeBeadID(id)
		if cleaned == "" {
			continue
		}
		idSet[cleaned] = true
	}
	prefix = sanitizeBeadID(prefix)

	if len(idSet) == 0 && prefix == "" {
		return beads
	}
	filtered := make([]ReadyBead, 0, len(beads))
	for _, bead := range beads {
		id := sanitizeBeadID(bead.ID)
		if id == "" {
			continue
		}
		if len(idSet) > 0 && idSet[id] {
			filtered = append(filtered, bead)
			continue
		}
		if prefix != "" && strings.HasPrefix(id, prefix) {
			filtered = append(filtered, bead)
		}
	}
	return filtered
}

func markBeadInProgress(id string) error {
	_, err := runBD("update", id, "--status", "in_progress")
	return err
}

func sanitizeBeadID(id string) string {
	return normalizeBeadID(id)
}
