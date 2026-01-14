package skills

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type Loader struct {
	Paths []string
}

type Skill struct {
	Name    string
	Path    string
	Content string
}

func (l Loader) LoadSkill(name string) (Skill, error) {
	for _, root := range l.Paths {
		candidate := filepath.Join(root, name, "SKILL.md")
		if exists(candidate) {
			return readSkill(name, candidate)
		}
	}

	for _, root := range l.Paths {
		var found Skill
		err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
			if err != nil || d.IsDir() || d.Name() != "SKILL.md" {
				return nil
			}
			skillName, ok := parseSkillName(path)
			if !ok || skillName != name {
				return nil
			}
			content, readErr := os.ReadFile(path)
			if readErr != nil {
				return readErr
			}
			found = Skill{Name: name, Path: path, Content: string(content)}
			return filepath.SkipDir
		})
		if err != nil {
			return Skill{}, err
		}
		if found.Name != "" {
			return found, nil
		}
	}

	return Skill{}, fmt.Errorf("skill not found: %s", name)
}

func readSkill(name, path string) (Skill, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return Skill{}, err
	}
	return Skill{Name: name, Path: path, Content: string(content)}, nil
}

func parseSkillName(path string) (string, bool) {
	file, err := os.Open(path)
	if err != nil {
		return "", false
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for i := 0; i < 25 && scanner.Scan(); i++ {
		line := strings.TrimSpace(scanner.Text())
		if strings.HasPrefix(line, "name:") {
			return strings.TrimSpace(strings.TrimPrefix(line, "name:")), true
		}
	}
	return "", false
}

func exists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
