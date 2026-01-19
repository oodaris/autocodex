package autonomy

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

func (c *Controller) createBeads(tasksFile TasksFile) error {
	if !c.Config.Beads.Enabled || !c.Config.Beads.AutoCreate {
		if c.Logger != nil {
			c.Logger.Info("beads auto-create disabled")
		}
		return nil
	}
	if !bdAvailable() {
		if c.Logger != nil {
			c.Logger.Warn("bd not found; skipping bead creation")
		}
		return nil
	}
	if len(tasksFile.Tasks) == 0 {
		return fmt.Errorf("no tasks to create")
	}

	created := map[string]bool{}
	for _, task := range tasksFile.Tasks {
		id := normalizeBeadID(task.ID)
		if id == "" {
			return fmt.Errorf("task id is required")
		}
		if created[id] {
			continue
		}
		if beadExists(id) {
			if c.Config.Beads.AutoUpdate {
				if err := updateBead(id, task); err != nil {
					return err
				}
			}
			created[id] = true
			continue
		}

		if err := createBead(id, task); err != nil {
			return err
		}
		created[id] = true
	}

	for _, task := range tasksFile.Tasks {
		id := normalizeBeadID(task.ID)
		if id == "" {
			continue
		}
		for _, dep := range task.Dependencies {
			depID := normalizeBeadID(dep)
			if depID == "" || depID == id {
				continue
			}
			if err := addDependency(id, depID); err != nil {
				return err
			}
		}
	}

	return nil
}

func createBead(id string, task Task) error {
	descFile, err := writeDescription(task)
	if err != nil {
		return err
	}
	defer os.Remove(descFile)

	args := []string{"create", "--id", id, "--title", task.Title, "--body-file", descFile}
	if task.Priority > 0 {
		args = append(args, "--priority", fmt.Sprintf("P%d", task.Priority))
	}
	if task.Owner != "" {
		args = append(args, "--assignee", task.Owner)
	}
	if task.ID != "" {
		args = append(args, "--external-ref", task.ID)
	}
	_, err = runBD(args...)
	return err
}

func updateBead(id string, task Task) error {
	descFile, err := writeDescription(task)
	if err != nil {
		return err
	}
	defer os.Remove(descFile)

	args := []string{"update", id, "--body-file", descFile}
	if task.Owner != "" {
		args = append(args, "--assignee", task.Owner)
	}
	_, err = runBD(args...)
	return err
}

func addDependency(id, dep string) error {
	_, err := runBD("dep", "add", id, dep)
	if err == nil {
		return nil
	}
	if isDependencyExistsError(err.Error()) {
		return nil
	}
	return err
}

func isDependencyExistsError(message string) bool {
	lower := strings.ToLower(message)
	return strings.Contains(lower, "unique constraint failed") ||
		strings.Contains(lower, "already exists") ||
		strings.Contains(lower, "duplicate")
}

func updateBeadStatus(id, status string) error {
	id = strings.TrimSpace(id)
	status = strings.TrimSpace(status)
	if id == "" || status == "" {
		return fmt.Errorf("bead id and status required")
	}
	switch strings.ToLower(status) {
	case "done", "closed", "complete", "completed":
		_, err := runBD("close", id)
		return err
	default:
		_, err := runBD("update", id, "--status", status)
		return err
	}
}

func beadExists(id string) bool {
	cmd := exec.Command("bd", "show", id)
	cmd.Stdout = &bytes.Buffer{}
	cmd.Stderr = &bytes.Buffer{}
	if err := cmd.Run(); err != nil {
		return false
	}
	return true
}

func runBD(args ...string) (string, error) {
	cmd := exec.Command("bd", args...)
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return stdout.String(), fmt.Errorf("bd %s failed: %w; stderr: %s", strings.Join(args, " "), err, stderr.String())
	}
	return stdout.String(), nil
}

func bdAvailable() bool {
	_, err := exec.LookPath("bd")
	return err == nil
}

func writeDescription(task Task) (string, error) {
	var b strings.Builder
	b.WriteString("Title: ")
	b.WriteString(task.Title)
	b.WriteString("\n")
	b.WriteString("Owner: ")
	b.WriteString(task.Owner)
	b.WriteString("\n")
	b.WriteString("Status: todo\n")
	b.WriteString("Goal: ")
	b.WriteString(task.Goal)
	b.WriteString("\n")
	b.WriteString("Scope: ")
	b.WriteString(strings.Join(task.Scope, ", "))
	b.WriteString("\n")
	b.WriteString("Files: ")
	b.WriteString(strings.Join(task.Files, ", "))
	b.WriteString("\n")
	b.WriteString("Dependencies: ")
	b.WriteString(strings.Join(task.Dependencies, ", "))
	b.WriteString("\n")
	b.WriteString("Constraints: \n")
	b.WriteString("Plan: \n")
	b.WriteString("Acceptance Criteria:\n")
	for _, item := range task.AcceptanceCriteria {
		b.WriteString("- ")
		b.WriteString(item)
		b.WriteString("\n")
	}
	b.WriteString("Contracts: \n")
	b.WriteString("Code: \n")
	b.WriteString("Tests: ")
	b.WriteString(strings.Join(task.Tests, ", "))
	b.WriteString("\n")
	b.WriteString("Docs: ")
	b.WriteString(strings.Join(task.Docs, ", "))
	b.WriteString("\n")
	b.WriteString("Rollout/Rollback: ")
	b.WriteString(task.Rollout)
	b.WriteString("\n")
	b.WriteString("Observability: ")
	b.WriteString(task.Observability)
	b.WriteString("\n")
	b.WriteString("Notes: ")
	b.WriteString(task.Notes)
	b.WriteString("\n")
	if len(task.Skills) > 0 {
		b.WriteString("Skills: ")
		b.WriteString(strings.Join(task.Skills, ", "))
		b.WriteString("\n")
	}

	file, err := os.CreateTemp("", "autocodex-bead-*.md")
	if err != nil {
		return "", fmt.Errorf("create temp description: %w", err)
	}
	if _, err := file.WriteString(b.String()); err != nil {
		_ = file.Close()
		return "", fmt.Errorf("write description: %w", err)
	}
	if err := file.Close(); err != nil {
		return "", fmt.Errorf("close description: %w", err)
	}
	return file.Name(), nil
}
