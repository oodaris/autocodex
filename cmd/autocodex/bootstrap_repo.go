package main

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/oodaris/autocodex/internal/config"
)

func ensureRepoPrereqs(cfg config.Config, initGit bool, initBD bool) error {
	needsBD := cfg.Beads.Enabled
	if cfg.Autonomy.Enabled && cfg.Autonomy.RequireBD != nil && *cfg.Autonomy.RequireBD {
		needsBD = true
	}
	if needsBD {
		if err := ensureGitRepo(initGit); err != nil {
			return err
		}
		if err := ensureBeads(initBD); err != nil {
			return err
		}
	}
	return nil
}

func ensureGitRepo(initGit bool) error {
	if !initGit {
		return nil
	}
	if _, err := os.Stat(".git"); err == nil {
		return nil
	}
	if _, err := exec.LookPath("git"); err != nil {
		return fmt.Errorf("git not found in PATH")
	}
	cmd := exec.Command("git", "init")
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("git init failed: %w; stderr: %s", err, stderr.String())
	}
	fmt.Println("Initialized git repository.")
	return nil
}

func ensureBeads(initBD bool) error {
	if !initBD {
		return nil
	}
	if _, err := os.Stat(".beads"); err == nil {
		return nil
	}
	if _, err := exec.LookPath("bd"); err != nil {
		return fmt.Errorf("bd not found in PATH")
	}
	cmd := exec.Command("bd", "init")
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("bd init failed: %w; stderr: %s", err, stderr.String())
	}
	prefix := defaultBeadPrefix()
	if prefix != "" {
		if err := ensureBeadPrefix(prefix); err != nil {
			return err
		}
	}
	fmt.Println("Initialized beads.")
	return nil
}

func ensureBeadPrefix(prefix string) error {
	current, err := bdConfigGet("issue_prefix")
	if err == nil && strings.TrimSpace(current) != "" {
		return nil
	}
	cmd := exec.Command("bd", "config", "set", "issue_prefix", prefix)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("bd config set issue_prefix failed: %w; stderr: %s", err, stderr.String())
	}
	fmt.Printf("Beads prefix set to %q.\n", prefix)
	return nil
}

func bdConfigGet(key string) (string, error) {
	cmd := exec.Command("bd", "config", "get", key)
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("bd config get %s failed: %w; stderr: %s", key, err, stderr.String())
	}
	return strings.TrimSpace(stdout.String()), nil
}

func defaultBeadPrefix() string {
	dir, err := os.Getwd()
	if err != nil {
		return ""
	}
	base := filepath.Base(dir)
	return sanitizePrefix(base)
}

func sanitizePrefix(prefix string) string {
	prefix = strings.ToLower(strings.TrimSpace(prefix))
	prefix = strings.Trim(prefix, "-")
	var b strings.Builder
	lastDash := false
	for _, r := range prefix {
		switch {
		case r >= 'a' && r <= 'z':
			b.WriteRune(r)
			lastDash = false
		case r >= '0' && r <= '9':
			b.WriteRune(r)
			lastDash = false
		case r == '-':
			b.WriteRune(r)
			lastDash = true
		default:
			if b.Len() > 0 && !lastDash {
				b.WriteRune('-')
				lastDash = true
			}
		}
	}
	out := strings.Trim(b.String(), "-")
	return out
}
