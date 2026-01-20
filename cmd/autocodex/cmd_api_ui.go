package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io/fs"
	"net"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/oodaris/autocodex/internal/api"
	"github.com/oodaris/autocodex/internal/config"
	"github.com/oodaris/autocodex/internal/hub"
	"github.com/oodaris/autocodex/internal/logging"
	"github.com/oodaris/autocodex/internal/state"
	"github.com/oodaris/autocodex/internal/terminal"
	"github.com/oodaris/autocodex/web"
)

func runAPI(args []string) {
	flagSet := flag.NewFlagSet("api", flag.ExitOnError)
	action := flagSet.String("action", "serve", "Action: serve")
	configPath := flagSet.String("config", config.ResolveConfigPath(), "Config file path")
	flagSet.Parse(args)

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
	var uiFS fs.FS
	if cfg.UI.Enabled {
		uiFS, err = web.DistFS()
		if err != nil {
			exitErr(fmt.Errorf("load embedded ui: %w", err))
		}
	}

	serveAPI(cfg, *configPath, uiFS)
}

func runUI(args []string) {
	flagSet := flag.NewFlagSet("ui", flag.ExitOnError)
	configPath := flagSet.String("config", config.ResolveConfigPath(), "Config file path")
	flagSet.Parse(args)

	cfg, err := config.Load(*configPath)
	if err != nil {
		exitErr(err)
	}
	cfg.API.Enabled = true
	cfg.UI.Enabled = true

	uiFS, err := web.DistFS()
	if err != nil {
		exitErr(fmt.Errorf("load embedded ui: %w", err))
	}

	serveAPI(cfg, *configPath, uiFS)
}

func serveAPI(cfg config.Config, configPath string, uiFS fs.FS) {
	logger := logging.NewLogger(cfg.Logging.Level)
	store := state.NewStore(cfg.StateDir(), cfg.RunsDir(), cfg.MemoryDir(), cfg.LogsDir(), cfg.ArtifactsDir())
	if err := store.InitDirs(); err != nil {
		exitErr(err)
	}

	rootDir, err := filepath.Abs(filepath.Dir(configPath))
	if err != nil {
		rootDir = ""
	}
	if cfg.Hub.Enabled && len(cfg.Hub.Workspaces) == 0 && rootDir != "" {
		wsID := filepath.Base(rootDir)
		if wsID == "" || wsID == "." || wsID == string(filepath.Separator) {
			wsID = "local"
		}
		cfg.Hub.Workspaces = []config.WorkspaceConfig{{
			ID:         wsID,
			Name:       wsID,
			Root:       rootDir,
			ConfigPath: configPath,
		}}
	}
	var hubManager *hub.Manager
	if cfg.Hub.Enabled {
		hubManager, err = hub.NewManager(cfg, logger)
		if err != nil {
			exitErr(err)
		}
	}

	authConfig := api.NewAuthConfig(cfg.Auth)
	if authConfig.Enabled && len(authConfig.Tokens) == 0 {
		exitErr(errors.New("auth enabled but no tokens resolved"))
	}

	server := &api.Server{
		Store:    store,
		Logger:   logger,
		Hub:      hubManager,
		Terminal: terminal.NewManager(logger),
		Auth:     authConfig,
		Config:   cfg,
		RootDir:  rootDir,
		UIFS:     uiFS,
	}

	if cfg.Loop.StopConditions.MaxHeartbeatSeconds > 0 {
		watchdog := &api.RunWatchdog{
			Store:               store,
			Logger:              logger,
			MaxHeartbeatSeconds: cfg.Loop.StopConditions.MaxHeartbeatSeconds,
		}
		go watchdog.Start(context.Background())
	}
	addr := net.JoinHostPort(cfg.API.Host, fmt.Sprintf("%d", cfg.API.Port))
	baseURL := fmt.Sprintf("http://%s:%d", hostForLog(cfg.API.Host), cfg.API.Port)
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		if isAddrInUse(err) {
			logger.Error("api bind failed", "addr", addr, "error", err.Error(), "hint", "stop the existing process or change api.port in autocodex.yaml")
		}
		exitErr(err)
	}
	logger.Info("api server starting", "route", "/", "status", "starting", "latency_ms", 0, "addr", addr, "url", baseURL, "config", configPath)
	if uiFS != nil {
		logger.Info("ui embedded", "route", "/", "status", "enabled", "latency_ms", 0, "addr", addr, "url", baseURL)
	}
	logger.Info("api endpoints", "health", baseURL+"/health", "runs", baseURL+"/runs", "memory", baseURL+"/memory")
	if uiFS != nil {
		logger.Info("ui ready", "url", baseURL, "hub", baseURL+"/hub")
	}
	if err := http.Serve(listener, server.Handler()); err != nil {
		exitErr(err)
	}
}

func hostForLog(host string) string {
	trimmed := strings.TrimSpace(host)
	switch trimmed {
	case "", "0.0.0.0", "::", "[::]":
		return "127.0.0.1"
	default:
		return trimmed
	}
}

func isAddrInUse(err error) bool {
	if err == nil {
		return false
	}
	lower := strings.ToLower(err.Error())
	return strings.Contains(lower, "address already in use")
}
