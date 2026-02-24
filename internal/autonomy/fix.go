package autonomy

import (
	"crypto/sha1"
	"fmt"
	"strings"
)

func (c *Controller) createFixBead(parentID, reason string) error {
	if !c.Config.Beads.Enabled || !c.Config.Beads.AutoCreate {
		return nil
	}
	prefix := sanitizeBeadPrefix(resolveBeadPrefix())
	if prefix == "" {
		prefix = defaultBeadPrefix
	}
	fixShort := fixBeadShort(parentID, prefix)
	signature := fixReasonSignature(parentID, reason)
	fixID := fmt.Sprintf("%s-fix-%s-%s", prefix, fixShort, signature)
	task := Task{
		ID:    fixID,
		Title: fmt.Sprintf("Fix gate failure for %s", parentID),
		Goal:  fmt.Sprintf("Resolve gate failure: %s", reason),
		Notes: "Auto-generated fix bead from autonomy gate failure.",
	}
	if beadExists(fixID) {
		return nil
	}
	if err := createBead(fixID, task); err != nil {
		return err
	}
	parentID = sanitizeBeadID(parentID)
	if parentID != "" {
		_ = addDependency(fixID, parentID)
	}
	return nil
}

func fixBeadShort(parentID, prefix string) string {
	parentShort := strings.TrimSpace(parentID)
	if parentShort == "" {
		return "unknown"
	}
	if prefix != "" && strings.HasPrefix(parentShort, prefix+"-") {
		parentShort = strings.TrimPrefix(parentShort, prefix+"-")
	}
	parentShort = strings.TrimPrefix(parentShort, "fix-")
	parentShort = strings.TrimPrefix(parentShort, "fix")
	parentShort = normalizeFixBeadSuffix(parentShort)
	if parentShort == "" {
		return "unknown"
	}
	return parentShort
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

func fixReasonSignature(parentID, reason string) string {
	input := strings.TrimSpace(parentID) + "|" + strings.ToLower(strings.TrimSpace(reason))
	sum := sha1.Sum([]byte(input))
	return fmt.Sprintf("%x", sum[:3])
}
