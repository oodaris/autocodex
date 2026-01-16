package autonomy

import (
	"encoding/json"
	"fmt"
	"sort"
)

type ReadyBead struct {
	ID       string `json:"id"`
	Title    string `json:"title"`
	Status   string `json:"status"`
	Priority int    `json:"priority"`
}

func (c *Controller) selectBead(preferredID string) (*ReadyBead, error) {
	ready, err := listReadyBeads()
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

func markBeadInProgress(id string) error {
	_, err := runBD("update", id, "--status", "in_progress")
	return err
}

func sanitizeBeadID(id string) string {
	return normalizeBeadID(id)
}
