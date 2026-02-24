package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/oodaris/autocodex/internal/config"
)

type harnessCheck struct {
	Name    string `json:"name"`
	Status  string `json:"status"`
	Details string `json:"details"`
}

func runHarness(args []string) {
	if len(args) == 0 {
		printHarnessUsage()
		exitErr(errors.New("harness subcommand required"))
	}
	if isHarnessHelpArg(args[0]) || args[0] == "help" {
		printHarnessUsage()
		return
	}
	switch args[0] {
	case "preflight":
		runHarnessPreflight(args[1:])
	case "lint":
		runHarnessLintCommand(args[1:])
	default:
		exitErr(fmt.Errorf("unknown harness subcommand: %s", args[0]))
	}
}

func printHarnessUsage() {
	fmt.Println("Usage: autocodex harness <subcommand> [args]")
	fmt.Println("Subcommands: preflight, lint")
}

func isHarnessHelpArg(value string) bool {
	switch value {
	case "-h", "--help":
		return true
	default:
		return false
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

	if err := printHarnessPreflightChecks(checks, *jsonOutput); err != nil {
		exitErr(err)
	}

	if hasFailure {
		exitErr(fmt.Errorf("harness preflight found issues"))
	}
	if !*jsonOutput {
		fmt.Println("Harness preflight passed.")
	}
}

func runHarnessLintCommand(args []string) {
	fs := flag.NewFlagSet("harness lint", flag.ExitOnError)
	configPath := fs.String("config", config.ResolveConfigPath(), "Config file path")
	jsonOutput := fs.Bool("json", false, "output check as JSON")
	fs.Parse(args)

	cfg, err := config.Load(*configPath)
	if err != nil {
		exitErr(err)
	}

	result := runHarnessLint(cfg)
	if err := printHarnessPreflightChecks([]harnessCheck{result}, *jsonOutput); err != nil {
		exitErr(err)
	}
	if result.Status == "error" {
		exitErr(fmt.Errorf("harness lint found issues"))
	}
	if !*jsonOutput {
		fmt.Println("Harness lint passed.")
	}
}

func printHarnessPreflightChecks(checks []harnessCheck, jsonOutput bool) error {
	if jsonOutput {
		data, err := json.MarshalIndent(checks, "", "  ")
		if err != nil {
			return err
		}
		fmt.Println(string(data))
		return nil
	}
	for _, check := range checks {
		fmt.Printf("%-22s %-6s %s\n", check.Name, strings.ToUpper(check.Status), check.Details)
	}
	return nil
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

	lintResult := runHarnessLint(cfg)
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
