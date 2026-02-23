package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/oodaris/autocodex/internal/config"
)

type harnessCheck struct {
	Name    string `json:"name"`
	Status  string `json:"status"`
	Details string `json:"details"`
}

func runHarness(args []string) {
	if len(args) == 0 {
		fmt.Println("Usage: autocodex harness <subcommand> [args]")
		fmt.Println("Subcommands: preflight")
		exitErr(errors.New("harness subcommand required"))
	}
	switch args[0] {
	case "preflight":
		runHarnessPreflight(args[1:])
	default:
		exitErr(fmt.Errorf("unknown harness subcommand: %s", args[0]))
	}
}

func runHarnessPreflight(args []string) {
	fs := flag.NewFlagSet("harness preflight", flag.ExitOnError)
	configPath := fs.String("config", config.ResolveConfigPath(), "Config file path")
	strict := fs.Bool("strict", false, "treat actionable warnings as failures")
	jsonOutput := fs.Bool("json", false, "output checks as JSON")
	fs.Parse(args)

	cfg, err := config.Load(*configPath)
	if err != nil {
		exitErr(err)
	}

	checks, hasFailure := runHarnessPreflightChecks(cfg, *configPath, *strict)

	if *jsonOutput {
		data, err := json.MarshalIndent(checks, "", "  ")
		if err != nil {
			exitErr(err)
		}
		fmt.Println(string(data))
	} else {
		for _, check := range checks {
			fmt.Printf("%-22s %-6s %s\n", check.Name, strings.ToUpper(check.Status), check.Details)
		}
	}

	if hasFailure {
		exitErr(fmt.Errorf("harness preflight found issues"))
	}
	fmt.Println("Harness preflight passed.")
}

func runHarnessPreflightChecks(cfg config.Config, configPath string, strict bool) ([]harnessCheck, bool) {
	checks := make([]harnessCheck, 0, 16)
	hasFailure := false

	doctor := runDoctorChecks(cfg, configPath)
	for _, result := range doctor {
		status := result.Status
		if strict && result.Status == "warn" && !isNonBlockingDoctorWarning(result.Name) {
			status = "error"
		}
		checks = append(checks, harnessCheck{
			Name:    "doctor." + result.Name,
			Status:  status,
			Details: result.Details,
		})
		if status == "error" {
			hasFailure = true
		}
	}

	rolePackPath := strings.TrimSpace(cfg.Autonomy.Harness.RolePackPath)
	if rolePackPath == "" {
		rolePackPath = ".codex"
	}
	if _, err := os.Stat(rolePackPath); err != nil {
		checks = append(checks, harnessCheck{
			Name:    "harness.role-pack",
			Status:  "error",
			Details: fmt.Sprintf("missing role pack path (%s)", rolePackPath),
		})
		hasFailure = true
	} else {
		checks = append(checks, harnessCheck{
			Name:    "harness.role-pack",
			Status:  "ok",
			Details: filepath.Clean(rolePackPath),
		})
	}

	lintResult := runHarnessLint()
	checks = append(checks, lintResult)
	if lintResult.Status == "error" {
		hasFailure = true
	}

	return checks, hasFailure
}

func isNonBlockingDoctorWarning(name string) bool {
	// Missing memory docs in a fresh clone is expected and should not block preflight.
	return name == "memory"
}

func runHarnessLint() harnessCheck {
	path := filepath.Join("scripts", "harness_config_lint.py")
	if _, err := os.Stat(path); err != nil {
		return harnessCheck{Name: "harness.lint", Status: "error", Details: fmt.Sprintf("missing (%s)", path)}
	}

	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "python3", path)
	out, err := cmd.CombinedOutput()
	if err != nil {
		details := strings.TrimSpace(string(out))
		if details == "" {
			details = err.Error()
		}
		return harnessCheck{Name: "harness.lint", Status: "error", Details: details}
	}
	return harnessCheck{Name: "harness.lint", Status: "ok", Details: strings.TrimSpace(string(out))}
}
