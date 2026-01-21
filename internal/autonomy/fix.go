package autonomy

import (
	"fmt"
	"strings"
	"time"
)

func (c *Controller) createFixBead(parentID, reason string) error {
	if !c.Config.Beads.Enabled || !c.Config.Beads.AutoCreate {
		return nil
	}
	prefix := sanitizeBeadPrefix(resolveBeadPrefix())
	if prefix == "" {
		prefix = defaultBeadPrefix
	}
	timestamp := time.Now().UTC().Format("150405")
	fixShort := fixBeadShort(parentID, prefix)
	fixID := fmt.Sprintf("%s-%s%s", prefix, fixShort, timestamp)
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

func fixBeadShort(parentID, prefix string) string {
	parentShort := strings.TrimSpace(parentID)
	if parentShort == "" {
		return "fix"
	}
	if prefix != "" && strings.HasPrefix(parentShort, prefix+"-") {
		parentShort = strings.TrimPrefix(parentShort, prefix+"-")
	}
	parentShort = normalizeFixBeadSuffix(parentShort)
	if parentShort == "" {
		return "fix"
	}
	return "fix" + parentShort
}

func normalizeFixBeadSuffix(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	if value == "" {
		return ""
	}
	var b strings.Builder
	for _, r := range value {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			b.WriteRune(r)
		}
	}
	out := b.String()
	if len(out) > 8 {
		out = out[:8]
	}
	return out
}
