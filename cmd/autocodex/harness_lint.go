package main

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/BurntSushi/toml"
	"github.com/oodaris/autocodex/internal/config"
)

const (
	harnessLintExpectedProfile = "max_capability"
	harnessLintExpectedModel   = "gpt-5.4"
	harnessLintExpectedSpark   = "gpt-5.3-codex-spark"
	harnessLintPassMessage     = "Harness config lint passed: autocodex harness role pack and governance assets validated."
)

var (
	harnessLintExpectedRoles = []string{
		"agentic_ai_architect",
		"backend_executor",
		"browser_validator",
		"commit_curator",
		"design_strategist",
		"frontend_executor",
		"independent_critic",
		"quality_gate_runner",
		"release_evidence_operator",
		"requirements_clarifier",
		"tracking_operator",
		"workflow_orchestrator",
	}
	harnessLintRequiredFeatures = []string{
		"multi_agent",
		"shell_tool",
		"unified_exec",
		"shell_snapshot",
		"runtime_metrics",
	}
	harnessLintPackMarkers = []string{
		"pattern a",
		"pattern e",
		"non-bypassable gate stack",
		"lifecycle/admission contract",
		"high-impact trigger criteria",
	}
	harnessLintRunbookMarkers = []string{
		"harness preflight",
		"scripts/dev/harness-cli-preflight.sh",
		"harness preflight passed",
	}
)

func runHarnessLint(cfg config.Config, configPath string) harnessCheck {
	repoRoot, err := resolveHarnessLintRepoRoot(configPath)
	if err != nil {
		return harnessCheck{Name: "harness.lint", Status: "error", Details: err.Error()}
	}
	issues := lintHarnessConfig(cfg, repoRoot)
	if len(issues) > 0 {
		return harnessCheck{Name: "harness.lint", Status: "error", Details: formatHarnessLintIssues(issues)}
	}
	return harnessCheck{Name: "harness.lint", Status: "ok", Details: harnessLintPassMessage}
}

func resolveHarnessLintRepoRoot(configPath string) (string, error) {
	trimmedConfigPath := strings.TrimSpace(configPath)
	if trimmedConfigPath != "" {
		absConfigPath, err := filepath.Abs(trimmedConfigPath)
		if err != nil {
			return "", fmt.Errorf("resolve config path: %w", err)
		}
		configDir := filepath.Dir(absConfigPath)
		if root, ok := findGitRoot(configDir); ok {
			return root, nil
		}
		return configDir, nil
	}

	cwd, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("get working directory: %w", err)
	}
	if root, ok := findGitRoot(cwd); ok {
		return root, nil
	}
	return cwd, nil
}

func findGitRoot(start string) (string, bool) {
	dir := filepath.Clean(start)
	for {
		if _, err := os.Stat(filepath.Join(dir, ".git")); err == nil {
			return dir, true
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return "", false
}

func lintHarnessConfig(cfg config.Config, repoRoot string) []string {
	issues := make([]string, 0, 24)

	rolePackPath := strings.TrimSpace(cfg.Autonomy.Harness.RolePackPath)
	if rolePackPath == "" {
		rolePackPath = ".codex"
	}
	rolePackRoot := filepath.Join(repoRoot, filepath.FromSlash(rolePackPath))
	configPath := filepath.Join(rolePackRoot, "config.toml")
	rolesDir := filepath.Join(rolePackRoot, "agents")

	operatingPack := filepath.Join(repoRoot, "docs", "agents", "autocodex-harness-v2-operating-pack.md")
	evalDocs := []string{
		filepath.Join(repoRoot, "docs", "agents", "harness-evals", "README.md"),
		filepath.Join(repoRoot, "docs", "agents", "harness-evals", "golden-task-catalog.md"),
		filepath.Join(repoRoot, "docs", "agents", "harness-evals", "failure-mode-catalog.md"),
	}
	preflightScript := filepath.Join(repoRoot, "scripts", "dev", "harness-cli-preflight.sh")
	preflightRunbook := filepath.Join(repoRoot, "docs", "runbooks", "harness-cli-preflight.md")

	rootCfg, ok := loadTOMLMap(configPath, repoRoot, &issues)
	if ok {
		profile := strings.TrimSpace(stringValue(rootCfg["profile"]))
		if profile != harnessLintExpectedProfile {
			issues = append(issues, fmt.Sprintf("%s profile must be %q", repoRelativePath(repoRoot, configPath), harnessLintExpectedProfile))
		}
		if model := strings.TrimSpace(stringValue(rootCfg["model"])); model != harnessLintExpectedModel {
			issues = append(issues, fmt.Sprintf("%s model must be %q", repoRelativePath(repoRoot, configPath), harnessLintExpectedModel))
		}
		if reviewModel := strings.TrimSpace(stringValue(rootCfg["review_model"])); reviewModel != harnessLintExpectedModel {
			issues = append(issues, fmt.Sprintf("%s review_model must be %q", repoRelativePath(repoRoot, configPath), harnessLintExpectedModel))
		}

		profilesTable, profilesOK := mapValue(rootCfg["profiles"])
		if !profilesOK {
			issues = append(issues, fmt.Sprintf("%s missing [profiles] table", repoRelativePath(repoRoot, configPath)))
		} else {
			maxCapability, maxCapabilityOK := mapValue(profilesTable["max_capability"])
			if !maxCapabilityOK {
				issues = append(issues, fmt.Sprintf("%s missing [profiles.max_capability] table", repoRelativePath(repoRoot, configPath)))
			} else {
				if model := strings.TrimSpace(stringValue(maxCapability["model"])); model != harnessLintExpectedModel {
					issues = append(issues, fmt.Sprintf("%s profiles.max_capability.model must be %q", repoRelativePath(repoRoot, configPath), harnessLintExpectedModel))
				}
				if reviewModel := strings.TrimSpace(stringValue(maxCapability["review_model"])); reviewModel != harnessLintExpectedModel {
					issues = append(issues, fmt.Sprintf("%s profiles.max_capability.review_model must be %q", repoRelativePath(repoRoot, configPath), harnessLintExpectedModel))
				}
			}

			sparkProfile, sparkOK := mapValue(profilesTable["spark"])
			if !sparkOK {
				issues = append(issues, fmt.Sprintf("%s missing [profiles.spark] table", repoRelativePath(repoRoot, configPath)))
			} else {
				if model := strings.TrimSpace(stringValue(sparkProfile["model"])); model != harnessLintExpectedSpark {
					issues = append(issues, fmt.Sprintf("%s profiles.spark.model must be %q", repoRelativePath(repoRoot, configPath), harnessLintExpectedSpark))
				}
				if reviewModel := strings.TrimSpace(stringValue(sparkProfile["review_model"])); reviewModel != harnessLintExpectedSpark {
					issues = append(issues, fmt.Sprintf("%s profiles.spark.review_model must be %q", repoRelativePath(repoRoot, configPath), harnessLintExpectedSpark))
				}
				if summary := strings.TrimSpace(stringValue(sparkProfile["model_reasoning_summary"])); summary != "none" {
					issues = append(issues, fmt.Sprintf("%s profiles.spark.model_reasoning_summary must be %q", repoRelativePath(repoRoot, configPath), "none"))
				}
			}
		}

		agentsTable, tableOK := mapValue(rootCfg["agents"])
		if !tableOK {
			issues = append(issues, fmt.Sprintf("%s missing [agents] table", repoRelativePath(repoRoot, configPath)))
		} else {
			roleEntries := make(map[string]any, len(agentsTable))
			for key, value := range agentsTable {
				if key == "max_threads" {
					continue
				}
				roleEntries[key] = value
			}

			missingRoles := setDifference(harnessLintExpectedRoles, mapKeys(roleEntries))
			extraRoles := setDifference(mapKeys(roleEntries), harnessLintExpectedRoles)
			if len(missingRoles) > 0 {
				issues = append(issues, fmt.Sprintf("missing required roles: %v", missingRoles))
			}
			if len(extraRoles) > 0 {
				issues = append(issues, fmt.Sprintf("unexpected roles present: %v", extraRoles))
			}

			for _, role := range harnessLintExpectedRoles {
				entry, exists := roleEntries[role]
				if !exists {
					continue
				}
				entryMap, entryOK := mapValue(entry)
				if !entryOK {
					issues = append(issues, fmt.Sprintf("[agents.%s] must be a table", role))
					continue
				}
				configFile := strings.TrimSpace(stringValue(entryMap["config_file"]))
				if configFile == "" {
					issues = append(issues, fmt.Sprintf("[agents.%s] missing config_file", role))
					continue
				}

				rolePath := filepath.Join(rolePackRoot, filepath.FromSlash(configFile))
				roleCfg, roleOK := loadTOMLMap(rolePath, repoRoot, &issues)
				if !roleOK {
					continue
				}
				if model := strings.TrimSpace(stringValue(roleCfg["model"])); model != harnessLintExpectedModel {
					issues = append(issues, fmt.Sprintf("%s model must be %q", repoRelativePath(repoRoot, rolePath), harnessLintExpectedModel))
				}

				features, featuresOK := mapValue(roleCfg["features"])
				if !featuresOK {
					issues = append(issues, fmt.Sprintf("%s missing [features] table", role))
				} else {
					for _, key := range harnessLintRequiredFeatures {
						if _, present := features[key]; !present {
							issues = append(issues, fmt.Sprintf("%s missing features.%s", role, key))
						}
					}
				}

				instructions := strings.TrimSpace(stringValue(roleCfg["developer_instructions"]))
				if instructions == "" {
					issues = append(issues, fmt.Sprintf("%s missing developer_instructions", role))
				}
			}
		}
	}

	if _, err := os.Stat(rolesDir); err != nil {
		issues = append(issues, fmt.Sprintf("missing roles directory: %s", repoRelativePath(repoRoot, rolesDir)))
	}

	packText, packOK := readTextFile(operatingPack, repoRoot, &issues)
	if packOK {
		containsMarkers(strings.ToLower(packText), harnessLintPackMarkers, "operating pack missing marker", &issues)
	}

	for _, path := range evalDocs {
		if _, err := os.Stat(path); err != nil {
			issues = append(issues, fmt.Sprintf("missing eval doc: %s", repoRelativePath(repoRoot, path)))
		}
	}

	scriptText, scriptOK := readTextFile(preflightScript, repoRoot, &issues)
	if scriptOK && !strings.HasPrefix(scriptText, "#!/usr/bin/env bash") {
		issues = append(issues, "preflight script must start with bash shebang")
	}

	runbookText, runbookOK := readTextFile(preflightRunbook, repoRoot, &issues)
	if runbookOK {
		containsMarkers(strings.ToLower(runbookText), harnessLintRunbookMarkers, "preflight runbook missing marker", &issues)
	}

	return issues
}

func loadTOMLMap(path, repoRoot string, issues *[]string) (map[string]any, bool) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			*issues = append(*issues, fmt.Sprintf("missing file: %s", repoRelativePath(repoRoot, path)))
		} else {
			*issues = append(*issues, fmt.Sprintf("read %s: %v", repoRelativePath(repoRoot, path), err))
		}
		return nil, false
	}

	payload := map[string]any{}
	if _, err := toml.Decode(string(data), &payload); err != nil {
		*issues = append(*issues, fmt.Sprintf("invalid TOML in %s: %v", repoRelativePath(repoRoot, path), err))
		return nil, false
	}
	return payload, true
}

func readTextFile(path, repoRoot string, issues *[]string) (string, bool) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			*issues = append(*issues, fmt.Sprintf("missing file: %s", repoRelativePath(repoRoot, path)))
		} else {
			*issues = append(*issues, fmt.Sprintf("read %s: %v", repoRelativePath(repoRoot, path), err))
		}
		return "", false
	}
	return string(data), true
}

func containsMarkers(text string, markers []string, format string, issues *[]string) {
	for _, marker := range markers {
		if !strings.Contains(text, marker) {
			*issues = append(*issues, fmt.Sprintf("%s: %q", format, marker))
		}
	}
}

func mapValue(value any) (map[string]any, bool) {
	switch typed := value.(type) {
	case map[string]any:
		return typed, true
	default:
		return nil, false
	}
}

func stringValue(value any) string {
	raw, ok := value.(string)
	if !ok {
		return ""
	}
	return raw
}

func setDifference(left, right []string) []string {
	rightSet := make(map[string]struct{}, len(right))
	for _, item := range right {
		rightSet[item] = struct{}{}
	}
	diff := make([]string, 0, len(left))
	for _, item := range left {
		if _, ok := rightSet[item]; !ok {
			diff = append(diff, item)
		}
	}
	sort.Strings(diff)
	return diff
}

func mapKeys(values map[string]any) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func repoRelativePath(repoRoot, path string) string {
	rel, err := filepath.Rel(repoRoot, path)
	if err != nil {
		return filepath.Clean(path)
	}
	return filepath.Clean(rel)
}

func formatHarnessLintIssues(issues []string) string {
	if len(issues) == 0 {
		return harnessLintPassMessage
	}
	var builder strings.Builder
	builder.WriteString("Harness config lint failed:")
	for _, issue := range issues {
		builder.WriteString("\n - ")
		builder.WriteString(issue)
	}
	return builder.String()
}
