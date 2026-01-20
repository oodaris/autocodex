package autonomy

import "strings"

func normalizeBeadID(id string) string {
	id = strings.TrimSpace(id)
	if id == "" {
		return id
	}
	parts := strings.Split(id, "-")
	if len(parts) <= 2 {
		return id
	}
	prefix := strings.TrimSpace(parts[0])
	suffix := strings.Trim(strings.Join(parts[1:], "-"), "-")
	if prefix == "" || suffix == "" {
		return id
	}
	return prefix + "-" + suffix
}
