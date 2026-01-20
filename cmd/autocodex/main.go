package main

import (
	"fmt"
	"os"
	"strings"
)

var version = "dev"

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(1)
	}

	cmd := os.Args[1]
	if isVersionArg(cmd) {
		printVersion()
		return
	}
	if !isCommand(cmd) && !strings.HasPrefix(cmd, "-") {
		task := strings.Join(os.Args[1:], " ")
		runRun([]string{"-task", task})
		return
	}

	switch cmd {
	case "bootstrap":
		runBootstrap(os.Args[2:])
	case "init":
		runInit(os.Args[2:])
	case "run":
		runRun(os.Args[2:])
	case "once":
		runOnce(os.Args[2:])
	case "resume":
		runResume(os.Args[2:])
	case "cleanup":
		runCleanup(os.Args[2:])
	case "kill":
		runKill(os.Args[2:])
	case "snapshot":
		runSnapshot(os.Args[2:])
	case "runs":
		runRuns(os.Args[2:])
	case "status":
		runStatus(os.Args[2:])
	case "beads":
		runBeads(os.Args[2:])
	case "plugins":
		runPlugins(os.Args[2:])
	case "api":
		runAPI(os.Args[2:])
	case "ui":
		runUI(os.Args[2:])
	case "version":
		printVersion()
	case "config":
		runConfig(os.Args[2:])
	default:
		usage()
		os.Exit(1)
	}
}

func usage() {
	fmt.Println("Usage: autocodex <command> [args]")
	fmt.Println("Commands: bootstrap, init, run, once, resume, kill, snapshot, runs, cleanup, status, beads, plugins, api, ui, version, config")
	fmt.Println("Shortcut: autocodex \"<task>\" (implicit run with --task)")
}

func isCommand(value string) bool {
	switch value {
	case "bootstrap", "init", "run", "once", "resume", "kill", "snapshot", "runs", "cleanup", "status", "beads", "plugins", "api", "ui", "version", "config":
		return true
	default:
		return false
	}
}

func isVersionArg(arg string) bool {
	switch arg {
	case "--version", "-v":
		return true
	default:
		return false
	}
}

func printVersion() {
	fmt.Println(version)
}
