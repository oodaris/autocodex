package main

import (
	"flag"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/oodaris/autocodex/internal/config"
)

type checkResult struct {
	Name     string
	Status   string
	Details  string
	Required bool
}

func runDoctor(args []string) {
	fs := flag.NewFlagSet("doctor", flag.ExitOnError)
	configPath := fs.String("config", config.ResolveConfigPath(), "Config file path")
	strict := fs.Bool("strict", false, "treat warnings as errors")
	fs.Parse(args)

	cfg, err := config.Load(*configPath)
	if err != nil {
		exitErr(err)
	}

	results := runDoctorChecks(cfg, *configPath)

	hasError := false
	for _, result := range results {
		if result.Status == "error" || (*strict && result.Status == "warn") {
			hasError = true
		}
		fmt.Printf("%-16s %-6s %s\n", result.Name, strings.ToUpper(result.Status), result.Details)
	}

	if hasError {
		exitErr(fmt.Errorf("doctor found issues"))
	}
}

func runDoctorChecks(cfg config.Config, configPath string) []checkResult {
	results := []checkResult{}

	results = append(results, checkConfigFile(configPath))
	results = append(results, checkConfigValidation(cfg))
	results = append(results, checkGitRepo())
	results = append(results, checkCommandOnPath("codex"))
	results = append(results, checkCodexVersion())
	if cfg.Autonomy.RequireBD != nil && *cfg.Autonomy.RequireBD {
		results = append(results, checkCommandOnPath("bd"))
	} else {
		results = append(results, checkCommandOptional("bd"))
	}
	results = append(results, checkMemoryDir(cfg))
	results = append(results, checkPortAvailability(cfg))
	return results
}

func checkConfigFile(path string) checkResult {
	if strings.TrimSpace(path) == "" {
		return checkResult{Name: "config", Status: "error", Details: "config path is empty", Required: true}
	}
	if _, err := os.Stat(path); err != nil {
		return checkResult{Name: "config", Status: "error", Details: fmt.Sprintf("missing (%s)", path), Required: true}
	}
	return checkResult{Name: "config", Status: "ok", Details: filepath.Clean(path), Required: true}
}

func checkConfigValidation(cfg config.Config) checkResult {
	if err := cfg.Validate(); err != nil {
		return checkResult{Name: "config", Status: "error", Details: fmt.Sprintf("invalid: %v", err), Required: true}
	}
	return checkResult{Name: "config", Status: "ok", Details: "validated", Required: true}
}

func checkGitRepo() checkResult {
	if _, err := os.Stat(".git"); err == nil {
		return checkResult{Name: "git", Status: "ok", Details: "repo detected", Required: true}
	}
	return checkResult{Name: "git", Status: "warn", Details: "no .git directory in cwd", Required: true}
}

func checkCommandOnPath(name string) checkResult {
	path, err := exec.LookPath(name)
	if err != nil {
		return checkResult{Name: name, Status: "error", Details: "not found in PATH", Required: true}
	}
	return checkResult{Name: name, Status: "ok", Details: path, Required: true}
}

func checkCommandOptional(name string) checkResult {
	path, err := exec.LookPath(name)
	if err != nil {
		return checkResult{Name: name, Status: "warn", Details: "not found in PATH", Required: false}
	}
	return checkResult{Name: name, Status: "ok", Details: path, Required: false}
}

type semver struct {
	Major int
	Minor int
	Patch int
}

func (v semver) String() string {
	return fmt.Sprintf("%d.%d.%d", v.Major, v.Minor, v.Patch)
}

func (v semver) Less(other semver) bool {
	if v.Major != other.Major {
		return v.Major < other.Major
	}
	if v.Minor != other.Minor {
		return v.Minor < other.Minor
	}
	return v.Patch < other.Patch
}

func recommendedMinCodexVersion() semver {
	// Keep this as a "warn-only" floor: autocodex likely works on older Codex CLIs,
	// but modern approvals/search/mode behavior changed significantly around 0.93+.
	// 0.94.0 is a practical baseline because Plan-mode defaults stabilized around there.
	return semver{Major: 0, Minor: 94, Patch: 0}
}

func parseFirstSemver(text string) (semver, bool) {
	re := regexp.MustCompile(`\b(\d+)\.(\d+)\.(\d+)\b`)
	m := re.FindStringSubmatch(text)
	if len(m) != 4 {
		return semver{}, false
	}
	major, err := strconv.Atoi(m[1])
	if err != nil {
		return semver{}, false
	}
	minor, err := strconv.Atoi(m[2])
	if err != nil {
		return semver{}, false
	}
	patch, err := strconv.Atoi(m[3])
	if err != nil {
		return semver{}, false
	}
	return semver{Major: major, Minor: minor, Patch: patch}, true
}

func checkCodexVersion() checkResult {
	path, err := exec.LookPath("codex")
	if err != nil {
		return checkResult{Name: "codex-version", Status: "warn", Details: "codex not found; skipping version check", Required: false}
	}

	out, err := exec.Command(path, "--version").CombinedOutput()
	if err != nil {
		return checkResult{Name: "codex-version", Status: "warn", Details: fmt.Sprintf("codex --version failed: %v", err), Required: false}
	}
	versionText := strings.TrimSpace(string(out))
	if versionText == "" {
		return checkResult{Name: "codex-version", Status: "warn", Details: "codex --version returned empty output", Required: false}
	}

	got, ok := parseFirstSemver(versionText)
	if !ok {
		// Show raw output to help debug unusual installs/wrappers.
		return checkResult{Name: "codex-version", Status: "warn", Details: fmt.Sprintf("unrecognized version output: %q", versionText), Required: false}
	}

	min := recommendedMinCodexVersion()
	if got.Less(min) {
		return checkResult{
			Name:     "codex-version",
			Status:   "warn",
			Details:  fmt.Sprintf("%s (recommended >= %s)", versionText, min.String()),
			Required: false,
		}
	}
	return checkResult{Name: "codex-version", Status: "ok", Details: versionText, Required: false}
}

func checkMemoryDir(cfg config.Config) checkResult {
	path := cfg.MemoryDir()
	if strings.TrimSpace(path) == "" {
		return checkResult{Name: "memory", Status: "warn", Details: "memory dir is empty", Required: false}
	}
	if _, err := os.Stat(path); err != nil {
		return checkResult{Name: "memory", Status: "warn", Details: fmt.Sprintf("missing (%s)", path), Required: false}
	}
	return checkResult{Name: "memory", Status: "ok", Details: path, Required: false}
}

func checkPortAvailability(cfg config.Config) checkResult {
	addr := fmt.Sprintf("%s:%d", cfg.API.Host, cfg.API.Port)
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return checkResult{Name: "port", Status: "warn", Details: fmt.Sprintf("%s unavailable", addr), Required: false}
	}
	_ = listener.Close()
	return checkResult{Name: "port", Status: "ok", Details: fmt.Sprintf("%s available", addr), Required: false}
}
