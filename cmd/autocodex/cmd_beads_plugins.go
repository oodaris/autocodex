package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"time"

	"github.com/oodaris/autocodex/internal/config"
	"github.com/oodaris/autocodex/internal/logging"
	"github.com/oodaris/autocodex/internal/plugins"
)

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

	logger := logging.NewLogger(cfg.Logging.Level, cfg.Logging.Format)
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

func findPlugin(pluginsList []plugins.Plugin, name string) (plugins.Plugin, error) {
	for _, p := range pluginsList {
		if p.Manifest.Name == name {
			return p, nil
		}
	}
	return plugins.Plugin{}, fmt.Errorf("plugin not found: %s", name)
}
