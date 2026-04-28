package main

import (
	"bufio"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/oodaris/autocodex/internal/config"
)

type checkResult struct {
	Name     string
	Status   string
	Details  string
	Required bool
}

const doctorCommandTimeout = 15 * time.Second

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
	repoRoot := doctorRepoRoot(configPath)

	results = append(results, checkConfigFile(configPath))
	results = append(results, checkConfigValidation(cfg))
	results = append(results, checkGitRepo(repoRoot))
	results = append(results, checkCommandOnPath("codex"))
	results = append(results, checkCodexVersion())
	results = append(results, checkCodexFeatures())
	var bdCheck checkResult
	if cfg.Autonomy.RequireBD != nil && *cfg.Autonomy.RequireBD {
		bdCheck = checkCommandOnPath("bd")
	} else {
		bdCheck = checkCommandOptional("bd")
	}
	results = append(results, bdCheck)
	if bdCheck.Status == "ok" {
		results = append(results, checkBDVersion(bdCheck.Details))
		results = append(results, checkBDDoltReadiness(bdCheck.Details))
	}
	results = append(results, checkMemoryDir(cfg, repoRoot))
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

func checkGitRepo(repoRoot string) checkResult {
	root := repoRoot
	if strings.TrimSpace(root) == "" {
		root = "."
	}
	if _, err := os.Stat(filepath.Join(root, ".git")); err == nil {
		return checkResult{Name: "git", Status: "ok", Details: "repo detected", Required: true}
	}
	return checkResult{Name: "git", Status: "warn", Details: "no .git directory near config or cwd", Required: true}
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

func recommendedMinBDVersion() semver {
	return semver{Major: 0, Minor: 56, Patch: 1}
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

	out, err := runCommandOutput(path, "--version")
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

func checkBDVersion(path string) checkResult {
	out, err := runCommandOutput(path, "--version")
	if err != nil {
		return checkResult{Name: "bd-version", Status: "warn", Details: fmt.Sprintf("bd --version failed: %v", err), Required: false}
	}
	status, details := assessBDVersionOutput(strings.TrimSpace(string(out)))
	return checkResult{Name: "bd-version", Status: status, Details: details, Required: false}
}

func assessBDVersionOutput(raw string) (string, string) {
	text := strings.TrimSpace(raw)
	if text == "" {
		return "warn", "bd --version returned empty output"
	}
	got, ok := parseFirstSemver(text)
	if !ok {
		return "warn", fmt.Sprintf("unrecognized bd version output: %q", text)
	}
	min := recommendedMinBDVersion()
	if got.Less(min) {
		return "warn", fmt.Sprintf("%s (requires >= %s for this repo's Beads 0.56.1 workflow)", text, min.String())
	}
	return "ok", fmt.Sprintf("%s (target >= %s)", text, min.String())
}

func checkBDDoltReadiness(path string) checkResult {
	out, err := runCommandOutput(path, "dolt", "show", "--json")
	text := strings.TrimSpace(string(out))
	if err != nil {
		details := text
		if details == "" {
			details = err.Error()
		}
		return checkResult{Name: "bd-dolt", Status: "warn", Details: fmt.Sprintf("bd dolt show --json failed: %s", details), Required: false}
	}
	status, details := assessBDDoltShowOutput(text)
	return checkResult{Name: "bd-dolt", Status: status, Details: details, Required: false}
}

func assessBDDoltShowOutput(raw string) (string, string) {
	text := strings.TrimSpace(raw)
	if text == "" {
		return "warn", "bd dolt show returned empty output"
	}
	if strings.HasPrefix(text, "{") {
		if status, details, ok := assessBDDoltShowJSON(text); ok {
			return status, details
		}
	}
	summary := summarizeBDDoltShow(text)
	lower := strings.ToLower(text)
	mode := strings.ToLower(doltShowField(text, "Mode:"))
	if mode == "embedded" {
		return "ok", summary + " (embedded mode)"
	}
	switch {
	case strings.Contains(lower, "server not reachable"), strings.Contains(lower, "server unreachable"), strings.Contains(text, "✗"):
		return "warn", summary + " (server not reachable)"
	case strings.Contains(lower, "server reachable"), strings.Contains(text, "✓"):
		return "ok", summary + " (server reachable)"
	case mode == "server":
		return "warn", summary + " (server mode; reachability unknown)"
	default:
		return "warn", summary + " (unable to determine server reachability)"
	}
}

func assessBDDoltShowJSON(raw string) (string, string, bool) {
	var payload map[string]any
	if err := json.Unmarshal([]byte(raw), &payload); err != nil {
		return "", "", false
	}
	summary := summarizeBDDoltShowJSON(payload)
	mode := strings.ToLower(strings.TrimSpace(toString(payload["mode"])))
	backend := strings.ToLower(strings.TrimSpace(toString(payload["backend"])))
	embedded, hasEmbedded := jsonBool(payload, "embedded")

	if backend != "" && backend != "dolt" {
		return "warn", summary + fmt.Sprintf(" (unexpected backend %q)", backend), true
	}
	if connectionOK, ok := jsonBool(payload, "connection_ok", "server_reachable", "reachable"); ok {
		if connectionOK {
			return "ok", summary + " (server reachable)", true
		}
		return "warn", summary + " (server not reachable)", true
	}
	if mode == "embedded" {
		return "ok", summary + " (embedded mode)", true
	}
	if mode == "" && hasEmbedded && embedded {
		return "ok", summary + " (embedded mode)", true
	}
	if mode == "server" {
		if reachable, ok := jsonBool(payload, "server_reachable", "reachable"); ok {
			if reachable {
				return "ok", summary + " (server reachable)", true
			}
			return "warn", summary + " (server not reachable)", true
		}
		if statusText, ok := jsonString(payload, "connection_status", "status", "connectivity"); ok {
			lowerStatus := strings.ToLower(statusText)
			switch {
			case strings.Contains(lowerStatus, "unreachable"), strings.Contains(lowerStatus, "not reachable"), strings.Contains(lowerStatus, "failed"):
				return "warn", summary + " (server not reachable)", true
			case strings.Contains(lowerStatus, "reachable"), strings.Contains(lowerStatus, "ok"), strings.Contains(lowerStatus, "connected"):
				return "ok", summary + " (server reachable)", true
			}
		}
		return "warn", summary + " (server mode; reachability unknown)", true
	}
	if mode != "" {
		return "warn", summary + fmt.Sprintf(" (unsupported mode %q)", mode), true
	}
	return "warn", summary + " (unable to determine dolt mode)", true
}

func summarizeBDDoltShow(raw string) string {
	parts := make([]string, 0, 4)
	if mode := doltShowField(raw, "Mode:"); mode != "" {
		parts = append(parts, "mode="+mode)
	}
	if db := doltShowField(raw, "Database:"); db != "" {
		parts = append(parts, "database="+db)
	}
	if host := doltShowField(raw, "Host:"); host != "" {
		parts = append(parts, "host="+host)
	}
	if port := doltShowField(raw, "Port:"); port != "" {
		parts = append(parts, "port="+port)
	}
	if len(parts) == 0 {
		return "bd dolt show"
	}
	return strings.Join(parts, " ")
}

func summarizeBDDoltShowJSON(payload map[string]any) string {
	parts := make([]string, 0, 5)
	if backend, ok := jsonString(payload, "backend"); ok {
		parts = append(parts, "backend="+backend)
	}
	if mode, ok := jsonString(payload, "mode"); ok {
		parts = append(parts, "mode="+mode)
	} else if embedded, ok := jsonBool(payload, "embedded"); ok && embedded {
		parts = append(parts, "mode=embedded")
	}
	if database, ok := jsonString(payload, "database"); ok {
		parts = append(parts, "database="+database)
	}
	if host, ok := jsonString(payload, "host"); ok {
		parts = append(parts, "host="+host)
	}
	if port, ok := jsonNumber(payload, "port"); ok {
		parts = append(parts, "port="+port)
	}
	if len(parts) == 0 {
		return "bd dolt show"
	}
	return strings.Join(parts, " ")
}

func jsonString(payload map[string]any, keys ...string) (string, bool) {
	for _, key := range keys {
		raw, ok := payload[key]
		if !ok {
			continue
		}
		value := strings.TrimSpace(toString(raw))
		if value != "" {
			return value, true
		}
	}
	return "", false
}

func jsonNumber(payload map[string]any, keys ...string) (string, bool) {
	for _, key := range keys {
		raw, ok := payload[key]
		if !ok {
			continue
		}
		switch v := raw.(type) {
		case float64:
			return strconv.Itoa(int(v)), true
		case int:
			return strconv.Itoa(v), true
		case int64:
			return strconv.FormatInt(v, 10), true
		case json.Number:
			return v.String(), true
		case string:
			trimmed := strings.TrimSpace(v)
			if trimmed != "" {
				return trimmed, true
			}
		}
	}
	return "", false
}

func jsonBool(payload map[string]any, keys ...string) (bool, bool) {
	for _, key := range keys {
		raw, ok := payload[key]
		if !ok {
			continue
		}
		switch v := raw.(type) {
		case bool:
			return v, true
		case string:
			trimmed := strings.ToLower(strings.TrimSpace(v))
			switch trimmed {
			case "true", "yes", "1", "reachable", "ok", "connected":
				return true, true
			case "false", "no", "0", "unreachable", "failed", "disconnected":
				return false, true
			}
		}
	}
	return false, false
}

func toString(value any) string {
	switch v := value.(type) {
	case string:
		return v
	case fmt.Stringer:
		return v.String()
	case nil:
		return ""
	default:
		return fmt.Sprintf("%v", v)
	}
}

func doltShowField(raw, prefix string) string {
	for _, line := range strings.Split(raw, "\n") {
		trimmed := strings.TrimSpace(line)
		if !strings.HasPrefix(trimmed, prefix) {
			continue
		}
		return strings.TrimSpace(strings.TrimPrefix(trimmed, prefix))
	}
	return ""
}

func checkCodexFeatures() checkResult {
	path, err := exec.LookPath("codex")
	if err != nil {
		return checkResult{Name: "codex-features", Status: "warn", Details: "codex not found; skipping feature checks", Required: false}
	}
	out, err := runCommandOutput(path, "features", "list")
	if err != nil {
		return checkResult{Name: "codex-features", Status: "warn", Details: fmt.Sprintf("codex features list failed: %v", err), Required: false}
	}
	features := parseCodexFeatureList(string(out))
	if len(features) == 0 {
		return checkResult{Name: "codex-features", Status: "warn", Details: "no feature rows parsed from codex features list", Required: false}
	}

	required := []string{"shell_tool", "unified_exec", "shell_snapshot", "collaboration_modes"}
	recommended := []string{"multi_agent", "runtime_metrics", "memory_tool", "child_agents_md"}
	missingRequired := []string{}
	missingRecommended := []string{}
	for _, name := range required {
		if !features[name] {
			missingRequired = append(missingRequired, name)
		}
	}
	for _, name := range recommended {
		if !features[name] {
			missingRecommended = append(missingRecommended, name)
		}
	}

	if len(missingRequired) > 0 {
		details := fmt.Sprintf("missing required features: %s", strings.Join(missingRequired, ", "))
		if len(missingRecommended) > 0 {
			details += fmt.Sprintf("; missing recommended features: %s", strings.Join(missingRecommended, ", "))
		}
		return checkResult{Name: "codex-features", Status: "warn", Details: details, Required: false}
	}
	details := "required feature set available"
	if len(missingRecommended) > 0 {
		details += fmt.Sprintf("; missing recommended features: %s", strings.Join(missingRecommended, ", "))
	}
	return checkResult{Name: "codex-features", Status: "ok", Details: details, Required: false}
}

func runCommandOutput(path string, args ...string) ([]byte, error) {
	ctx, cancel := context.WithTimeout(context.Background(), doctorCommandTimeout)
	defer cancel()
	return exec.CommandContext(ctx, path, args...).CombinedOutput()
}

func parseCodexFeatureList(raw string) map[string]bool {
	rows := map[string]bool{}
	scanner := bufio.NewScanner(strings.NewReader(raw))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) < 3 {
			continue
		}
		name := fields[0]
		enabled := strings.EqualFold(fields[len(fields)-1], "true")
		rows[name] = enabled
	}
	return rows
}

func checkMemoryDir(cfg config.Config, repoRoot string) checkResult {
	path := cfg.MemoryDir()
	if strings.TrimSpace(path) == "" {
		return checkResult{Name: "memory", Status: "warn", Details: "memory dir is empty", Required: false}
	}
	resolvedPath := path
	if repoRoot != "" && !filepath.IsAbs(resolvedPath) {
		resolvedPath = filepath.Join(repoRoot, resolvedPath)
	}
	if _, err := os.Stat(resolvedPath); err != nil {
		return checkResult{Name: "memory", Status: "warn", Details: fmt.Sprintf("missing (%s)", displayDoctorPath(repoRoot, resolvedPath)), Required: false}
	}
	return checkResult{Name: "memory", Status: "ok", Details: displayDoctorPath(repoRoot, resolvedPath), Required: false}
}

func doctorRepoRoot(configPath string) string {
	root, err := resolveRepoRootFromConfigPath(configPath)
	if err != nil {
		return ""
	}
	return root
}

func displayDoctorPath(repoRoot, path string) string {
	if repoRoot == "" {
		return filepath.Clean(path)
	}
	return repoRelativePath(repoRoot, path)
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
