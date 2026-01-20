package main

import (
	"errors"
	"path/filepath"
	"strings"
)

func bootstrapSkills(root string, force bool) error {
	if strings.TrimSpace(root) == "" {
		return errors.New("skills root is empty")
	}
	skills := []struct {
		Name    string
		Content string
	}{
		{Name: "core-ask-questions-if-underspecified", Content: defaultSkillAskQuestions},
		{Name: "core-qna-synthesis", Content: defaultSkillQnaSynthesis},
		{Name: "core-holistic-planning-and-tracking", Content: defaultSkillHolisticPlanning},
	}
	for _, skill := range skills {
		path := filepath.Join(root, skill.Name, "SKILL.md")
		if err := writeFileIfMissing(path, []byte(skill.Content), force); err != nil {
			return err
		}
	}
	return nil
}

func skillsPathConfigured(paths []string, root string) bool {
	if strings.TrimSpace(root) == "" {
		return false
	}
	root = filepath.Clean(root)
	for _, p := range paths {
		if strings.TrimSpace(p) == "" {
			continue
		}
		if filepath.Clean(p) == root {
			return true
		}
	}
	return false
}
