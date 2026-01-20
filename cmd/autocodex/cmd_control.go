package main

import (
	"flag"
	"fmt"
	"strings"
	"time"

	"github.com/oodaris/autocodex/internal/config"
	"github.com/oodaris/autocodex/internal/state"
)

func runKill(args []string) {
	runControlAction(args, "kill")
}

func runControlAction(args []string, action string) {
	fs := flag.NewFlagSet(action, flag.ExitOnError)
	configPath := fs.String("config", config.ResolveConfigPath(), "Config file path")
	runID := fs.String("run", "", "run id")
	reason := fs.String("reason", "", "reason (optional)")
	jsonOut := fs.Bool("json", false, "output JSON")
	fs.Parse(args)

	if *runID == "" && fs.NArg() > 0 {
		*runID = strings.TrimSpace(fs.Arg(0))
	}

	if *runID == "" {
		exitErr(fmt.Errorf("run id is required"))
	}
	cfg, err := config.Load(*configPath)
	if err != nil {
		exitErr(err)
	}
	store := state.NewStore(cfg.StateDir(), cfg.RunsDir(), cfg.MemoryDir(), cfg.LogsDir(), cfg.ArtifactsDir())
	if err := store.InitDirs(); err != nil {
		exitErr(err)
	}
	control, err := requestRunAction(store, *runID, action, *reason)
	if err != nil {
		exitErr(err)
	}
	if *jsonOut {
		writeJSON(control)
		return
	}
	fmt.Printf("Requested %s for run %s\n", action, *runID)
}

func requestRunAction(store *state.Store, runID, action, reason string) (*state.RunControl, error) {
	run, err := store.GetRun(runID)
	if err != nil {
		return nil, err
	}
	now := time.Now().UTC()
	control := state.RunControl{
		RunID:        run.ID,
		Status:       run.Status,
		LastAction:   &action,
		LastActionAt: &now,
		UpdatedAt:    now,
	}
	if reason != "" && action != "resume" {
		control.StopReason = &reason
	}
	if action == "kill" {
		lock, _ := store.GetRunLock(run.ID)
		if lock != nil && lock.PID > 0 && state.IsProcessAlive(lock.PID) {
			_ = state.TerminateProcess(lock.PID)
		} else {
			stopReason := reason
			if strings.TrimSpace(stopReason) == "" {
				stopReason = "killed"
			}
			if err := store.FinalizeRun(run.ID, "failed", stopReason, "kill"); err != nil {
				return nil, err
			}
			return store.GetRunControl(run.ID)
		}
	}
	if err := store.SaveRunControl(control); err != nil {
		return nil, err
	}
	return &control, nil
}
