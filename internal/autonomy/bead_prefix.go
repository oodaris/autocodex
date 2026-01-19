package autonomy

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

const defaultBeadPrefix = "autocodex"

func resolveBeadPrefix() string {
	if prefix, err := bdIssuePrefix(); err == nil && prefix != "" {
		return prefix
	}
	if prefix, err := beadsConfigPrefix(); err == nil && prefix != "" {
		return prefix
	}
	return defaultBeadPrefix
}

func bdIssuePrefix() (string, error) {
	if !bdAvailable() {
		return "", errors.New("bd not available")
	}
	cmd := exec.Command("bd", "config", "get", "issue_prefix")
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("bd config get issue_prefix failed: %w; stderr: %s", err, stderr.String())
	}
	prefix := parseIssuePrefix(stdout.String())
	if prefix == "" {
		return "", errors.New("issue_prefix empty")
	}
	return prefix, nil
}

func parseIssuePrefix(output string) string {
	lines := strings.Split(strings.TrimSpace(output), "\n")
	first := ""
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed != "" {
			first = trimmed
			break
		}
	}
	if first == "" {
		return ""
	}
	if strings.Contains(first, ":") {
		parts := strings.SplitN(first, ":", 2)
		first = parts[1]
	}
	first = strings.TrimSpace(first)
	first = strings.Trim(first, "\"'")
	return sanitizeBeadPrefix(first)
}

func beadsConfigPrefix() (string, error) {
	path := findBeadsConfigPath()
	if path == "" {
		return "", errors.New("beads config not found")
	}
	content, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("read beads config: %w", err)
	}
	var cfg map[string]any
	if err := yaml.Unmarshal(content, &cfg); err != nil {
		return "", fmt.Errorf("parse beads config: %w", err)
	}
	prefix := extractPrefix(cfg)
	if prefix == "" {
		return "", errors.New("beads issue_prefix missing")
	}
	return prefix, nil
}

func extractPrefix(data map[string]any) string {
	if data == nil {
		return ""
	}
	for _, key := range []string{"issue_prefix", "issue-prefix", "prefix"} {
		if value, ok := data[key]; ok {
			if str, ok := value.(string); ok {
				if prefix := sanitizeBeadPrefix(str); prefix != "" {
					return prefix
				}
			}
		}
	}
	for _, key := range []string{"issue", "beads"} {
		if nested, ok := data[key].(map[string]any); ok {
			if prefix := extractPrefix(nested); prefix != "" {
				return prefix
			}
		}
	}
	return ""
}

func findBeadsConfigPath() string {
	dir, err := os.Getwd()
	if err != nil {
		return ""
	}
	for {
		for _, name := range []string{"config.yaml", "config.yml"} {
			candidate := filepath.Join(dir, ".beads", name)
			if _, err := os.Stat(candidate); err == nil {
				return candidate
			}
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return ""
		}
		dir = parent
	}
}

func sanitizeBeadPrefix(prefix string) string {
	prefix = strings.TrimSpace(prefix)
	prefix = strings.Trim(prefix, "-")
	prefix = strings.Trim(prefix, "\"'")
	return prefix
}

func applyBeadPrefix(id, prefix string) string {
	id = strings.TrimSpace(id)
	if id == "" {
		return id
	}
	prefix = sanitizeBeadPrefix(prefix)
	if prefix == "" {
		return id
	}
	parts := strings.SplitN(id, "-", 2)
	suffix := ""
	if len(parts) == 1 {
		suffix = strings.TrimSpace(parts[0])
	} else {
		suffix = strings.Trim(parts[1], "-")
	}
	if suffix == "" {
		return id
	}
	return prefix + "-" + suffix
}

func normalizeTaskID(id, prefix string) string {
	normalized := normalizeBeadID(id)
	return applyBeadPrefix(normalized, prefix)
}

func beadIDPattern(prefix string) string {
	prefix = sanitizeBeadPrefix(prefix)
	if prefix == "" {
		prefix = defaultBeadPrefix
	}
	return fmt.Sprintf("%s-<short>", prefix)
}

func fallbackBeadID(prefix string) string {
	prefix = sanitizeBeadPrefix(prefix)
	if prefix == "" {
		prefix = defaultBeadPrefix
	}
	return prefix + "-fallback"
}
