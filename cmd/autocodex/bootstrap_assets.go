package main

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/oodaris/autocodex/internal/config"
)

func bootstrapAutonomyAssets(cfg config.Config, force bool) error {
	if err := writeFileIfMissing(cfg.Autonomy.SpecTemplate, []byte(defaultSpecTemplate), force); err != nil {
		return err
	}
	if err := writeFileIfMissing(cfg.Autonomy.PlanTemplate, []byte(defaultPlanTemplate), force); err != nil {
		return err
	}
	if err := writeFileIfMissing(cfg.Autonomy.TasksSchema, []byte(defaultAutonomyTasksSchema), force); err != nil {
		return err
	}
	if err := writeFileIfMissing(cfg.Autonomy.ActionsSchema, []byte(defaultAutonomyActionsSchema), force); err != nil {
		return err
	}
	return nil
}

func writeFileIfMissing(path string, content []byte, force bool) error {
	if strings.TrimSpace(path) == "" {
		return errors.New("path is empty")
	}
	if !force {
		if _, err := os.Stat(path); err == nil {
			return nil
		} else if !errors.Is(err, os.ErrNotExist) {
			return err
		}
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("create dir %s: %w", filepath.Dir(path), err)
	}
	if err := os.WriteFile(path, content, 0o644); err != nil {
		return fmt.Errorf("write %s: %w", path, err)
	}
	return nil
}
