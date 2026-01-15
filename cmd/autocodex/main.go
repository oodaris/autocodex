package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/oodaris/autocodex/internal/api"
	"github.com/oodaris/autocodex/internal/codex"
	"github.com/oodaris/autocodex/internal/config"
	"github.com/oodaris/autocodex/internal/logging"
	"github.com/oodaris/autocodex/internal/orchestrator"
	"github.com/oodaris/autocodex/internal/plugins"
	"github.com/oodaris/autocodex/internal/skills"
	"github.com/oodaris/autocodex/internal/state"
)

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(1)
	}

	cmd := os.Args[1]
	switch cmd {
	case "init":
		runInit(os.Args[2:])
	case "run":
		runRun(os.Args[2:])
	case "status":
		runStatus(os.Args[2:])
	case "beads":
		runBeads(os.Args[2:])
	case "plugins":
		runPlugins(os.Args[2:])
	case "api":
		runAPI(os.Args[2:])
	case "config":
		runConfig(os.Args[2:])
	default:
		usage()
		os.Exit(1)
	}
}

func usage() {
	fmt.Println("Usage: autocodex <command> [args]")
	fmt.Println("Commands: init, run, status, beads, plugins, api, config")
}

func runInit(args []string) {
	fs := flag.NewFlagSet("init", flag.ExitOnError)
	configPath := fs.String("config", config.ResolveConfigPath(), "Config file path")
	fs.Parse(args)

	if err := ensureConfig(*configPath); err != nil {
		exitErr(err)
	}

	cfg, err := config.Load(*configPath)
	if err != nil {
		exitErr(err)
	}

	store := state.NewStore(cfg.StateDir(), cfg.RunsDir(), cfg.MemoryDir(), cfg.LogsDir(), cfg.ArtifactsDir())
	if err := store.InitDirs(); err != nil {
		exitErr(err)
	}
	if err := store.EnsureMemoryDocs(); err != nil {
		exitErr(err)
	}

	fmt.Printf("Initialized Autocodex. Config: %s\n", *configPath)
}

func runRun(args []string) {
	fs := flag.NewFlagSet("run", flag.ExitOnError)
	configPath := fs.String("config", config.ResolveConfigPath(), "Config file path")
	fs.Parse(args)

	cfg, err := config.Load(*configPath)
	if err != nil {
		exitErr(err)
	}

	logger := logging.NewLogger(cfg.Logging.Level)
	store := state.NewStore(cfg.StateDir(), cfg.RunsDir(), cfg.MemoryDir(), cfg.LogsDir(), cfg.ArtifactsDir())
	loader := skills.Loader{Paths: cfg.Skills.Paths}
	runner := codex.Runner{
		CLIPath:         cfg.Codex.CLIPath,
		Model:           cfg.Codex.Model,
		ReasoningEffort: cfg.Codex.ReasoningEffort,
		ExtraArgs:       cfg.Codex.ExtraArgs,
		Mode:            cfg.Mode,
		ApprovalPolicy:  cfg.Codex.ApprovalPolicy,
		SandboxMode:     cfg.Codex.SandboxMode,
		Timeout:         time.Duration(cfg.Codex.TimeoutSeconds) * time.Second,
		Env:             cfg.Codex.Env,
	}

	orch := orchestrator.Orchestrator{
		Config: cfg,
		Logger: logger,
		Store:  store,
		Skills: loader,
		Codex:  runner,
	}

	ctx := context.Background()
	if _, err := orch.Run(ctx); err != nil {
		exitErr(err)
	}
}

func runStatus(args []string) {
	fs := flag.NewFlagSet("status", flag.ExitOnError)
	configPath := fs.String("config", config.ResolveConfigPath(), "Config file path")
	fs.Parse(args)

	cfg, err := config.Load(*configPath)
	if err != nil {
		exitErr(err)
	}
	store := state.NewStore(cfg.StateDir(), cfg.RunsDir(), cfg.MemoryDir(), cfg.LogsDir(), cfg.ArtifactsDir())
	runs, err := store.ListRuns()
	if err != nil {
		exitErr(err)
	}
	if len(runs) == 0 {
		fmt.Println("No runs found")
		return
	}
	for _, run := range runs {
		fmt.Printf("%s\t%s\t%s\n", run.ID, run.Status, run.CurrentPhase)
	}
}

func runBeads(args []string) {
	fs := flag.NewFlagSet("beads", flag.ExitOnError)
	action := fs.String("action", "ready", "bd action (ready|show)")
	issue := fs.String("issue", "", "issue id for show")
	fs.Parse(args)

	cmdArgs := []string{*action}
	if *action == "show" && *issue != "" {
		cmdArgs = append(cmdArgs, *issue)
	}

	cmd := exec.Command("bd", cmdArgs...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		exitErr(fmt.Errorf("bd %s failed: %w", *action, err))
	}
}

func runPlugins(args []string) {
	fs := flag.NewFlagSet("plugins", flag.ExitOnError)
	action := fs.String("action", "list", "Action: list|run")
	name := fs.String("name", "", "Plugin name (run)")
	capability := fs.String("capability", "", "Capability name (run)")
	input := fs.String("input", "", "JSON input string (run)")
	inputFile := fs.String("input-file", "", "Path to JSON input file (run)")
	configPath := fs.String("config", config.ResolveConfigPath(), "Config file path")
	fs.Parse(args)

	cfg, err := config.Load(*configPath)
	if err != nil {
		exitErr(err)
	}

	logger := logging.NewLogger(cfg.Logging.Level)
	traceID := fmt.Sprintf("trace-%d", time.Now().UnixNano())
	tenantID := os.Getenv("AUTOCODEX_TENANT_ID")
	if tenantID == "" {
		tenantID = "local"
	}
	logger = logger.With("trace_id", traceID, "tenant_id", tenantID, "route", "plugins."+*action)

	pluginsList, err := plugins.Discover(cfg.Plugins.Paths)
	if err != nil {
		logger.Error("plugin discovery failed", "status", "failed", "latency_ms", 0, "error", err.Error())
		exitErr(err)
	}

	switch *action {
	case "list":
		payload := make([]map[string]interface{}, 0, len(pluginsList))
		for _, p := range pluginsList {
			caps := make([]string, 0, len(p.Manifest.Capabilities))
			for _, cap := range p.Manifest.Capabilities {
				caps = append(caps, cap.Name)
			}
			payload = append(payload, map[string]interface{}{
				"name":         p.Manifest.Name,
				"version":      p.Manifest.Version,
				"transport":    p.Manifest.Transport,
				"capabilities": caps,
			})
		}
		writeJSON(payload)
	case "run":
		if *name == "" || *capability == "" {
			exitErr(fmt.Errorf("name and capability are required for run"))
		}
		plugin, err := findPlugin(pluginsList, *name)
		if err != nil {
			exitErr(err)
		}
		inputBytes, err := resolveInput(*input, *inputFile)
		if err != nil {
			exitErr(err)
		}
		host := plugins.Host{Timeout: time.Duration(cfg.Plugins.TimeoutSeconds) * time.Second}
		start := time.Now()
		output, err := host.Call(context.Background(), plugin, *capability, inputBytes)
		latency := time.Since(start).Milliseconds()
		if err != nil {
			logger.Error("plugin call failed", "status", "failed", "latency_ms", latency, "error", err.Error())
			writeJSON(map[string]interface{}{
				"output": nil,
				"error": map[string]string{
					"message": err.Error(),
				},
			})
			os.Exit(1)
		}
		logger.Info("plugin call completed", "status", "completed", "latency_ms", latency)
		writeJSON(map[string]interface{}{
			"output": json.RawMessage(output),
			"error":  nil,
		})
	default:
		exitErr(fmt.Errorf("unknown action: %s", *action))
	}
}

func runAPI(args []string) {
	fs := flag.NewFlagSet("api", flag.ExitOnError)
	action := fs.String("action", "serve", "Action: serve")
	configPath := fs.String("config", config.ResolveConfigPath(), "Config file path")
	fs.Parse(args)

	if *action != "serve" {
		exitErr(fmt.Errorf("unknown action: %s", *action))
	}

	cfg, err := config.Load(*configPath)
	if err != nil {
		exitErr(err)
	}
	if !cfg.API.Enabled {
		exitErr(errors.New("api is disabled in config"))
	}

	logger := logging.NewLogger(cfg.Logging.Level)
	store := state.NewStore(cfg.StateDir(), cfg.RunsDir(), cfg.MemoryDir(), cfg.LogsDir(), cfg.ArtifactsDir())
	if err := store.InitDirs(); err != nil {
		exitErr(err)
	}

	server := &api.Server{Store: store, Logger: logger}
	addr := net.JoinHostPort(cfg.API.Host, fmt.Sprintf("%d", cfg.API.Port))
	logger.Info("api server starting", "route", "/", "status", "starting", "latency_ms", 0, "addr", addr)
	if err := http.ListenAndServe(addr, server.Handler()); err != nil {
		exitErr(err)
	}
}

func runConfig(args []string) {
	fs := flag.NewFlagSet("config", flag.ExitOnError)
	configPath := fs.String("config", config.ResolveConfigPath(), "Config file path")
	fs.Parse(args)
	fmt.Printf("Config path: %s\n", *configPath)
}

func ensureConfig(path string) error {
	if _, err := os.Stat(path); err == nil {
		return nil
	} else if !errors.Is(err, os.ErrNotExist) {
		return err
	}

	examplePath := filepath.Join("config.example.yaml")
	data, err := os.ReadFile(examplePath)
	if err != nil {
		return fmt.Errorf("read config.example.yaml: %w", err)
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return fmt.Errorf("write %s: %w", path, err)
	}
	return nil
}

func resolveInput(input, inputFile string) (json.RawMessage, error) {
	if inputFile != "" {
		data, err := os.ReadFile(inputFile)
		if err != nil {
			return nil, fmt.Errorf("read input file: %w", err)
		}
		return json.RawMessage(data), nil
	}
	if strings.TrimSpace(input) == "" {
		return json.RawMessage([]byte("{}")), nil
	}
	return json.RawMessage([]byte(input)), nil
}

func findPlugin(pluginsList []plugins.Plugin, name string) (plugins.Plugin, error) {
	for _, p := range pluginsList {
		if p.Manifest.Name == name {
			return p, nil
		}
	}
	return plugins.Plugin{}, fmt.Errorf("plugin not found: %s", name)
}

func writeJSON(v interface{}) {
	enc := json.NewEncoder(os.Stdout)
	enc.SetEscapeHTML(false)
	_ = enc.Encode(v)
}

func exitErr(err error) {
	fmt.Fprintf(os.Stderr, "error: %v\n", err)
	os.Exit(1)
}
