package autonomy

import (
	"fmt"
	"time"
)

func (c *Controller) createFixBead(parentID, reason string) error {
	if !c.Config.Beads.Enabled || !c.Config.Beads.AutoCreate {
		return nil
	}
	timestamp := time.Now().UTC().Format("150405")
	fixID := fmt.Sprintf("%s-fix-%s", parentID, timestamp)
	task := Task{
		ID:    fixID,
		Title: fmt.Sprintf("Fix gate failure for %s", parentID),
		Goal:  fmt.Sprintf("Resolve gate failure: %s", reason),
		Notes: "Auto-generated fix bead from autonomy gate failure.",
	}
	if beadExists(fixID) {
		return nil
	}
	return createBead(fixID, task)
}
