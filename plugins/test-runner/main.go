package main

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

type request struct {
	Capability string          `json:"capability"`
	Input      json.RawMessage `json:"input"`
}

type response struct {
	Output interface{}  `json:"output,omitempty"`
	Error  *pluginError `json:"error,omitempty"`
}

type pluginError struct {
	Message string `json:"message"`
	Code    string `json:"code,omitempty"`
}

type input struct {
	Root           string   `json:"root"`
	ChangedFiles   []string `json:"changed_files"`
	Commands       []string `json:"commands"`
	TimeoutSeconds int      `json:"timeout_seconds"`
	MaxOutputBytes int      `json:"max_output_bytes"`
}

type runResult struct {
	Command         string `json:"command"`
	ExitCode        int    `json:"exit_code"`
	DurationMs      int64  `json:"duration_ms"`
	Stdout          string `json:"stdout"`
	Stderr          string `json:"stderr"`
	TimedOut        bool   `json:"timed_out"`
	OutputTruncated bool   `json:"output_truncated"`
}

type output struct {
	Status         string      `json:"status"`
	FailedCommands int         `json:"failed_commands"`
	Runs           []runResult `json:"runs"`
}

func main() {
	handshake := map[string]interface{}{
		"protocol_version": 1,
		"name":             "test-runner",
		"transport":        "jsonrpc",
		"address":          "stdio",
	}
	b, _ := json.Marshal(handshake)
	fmt.Println(string(b))

	scanner := bufio.NewScanner(os.Stdin)
	scanner.Buffer(make([]byte, 0, 64*1024), 4*1024*1024)
	for scanner.Scan() {
		var req request
		if err := json.Unmarshal(scanner.Bytes(), &req); err != nil {
			writeResponse(response{Error: &pluginError{Message: "invalid request", Code: "bad_request"}})
			continue
		}
		switch req.Capability {
		case "run":
			out, err := runTests(req.Input)
			if err != nil {
				writeResponse(response{Error: &pluginError{Message: err.Error(), Code: "run_failed"}})
				continue
			}
			writeResponse(response{Output: out})
		default:
			writeResponse(response{Error: &pluginError{Message: "unknown capability", Code: "unknown_capability"}})
		}
	}
}

func runTests(inputRaw json.RawMessage) (output, error) {
	in := input{Root: ".", TimeoutSeconds: 900, MaxOutputBytes: 20000}
	_ = json.Unmarshal(inputRaw, &in)
	if strings.TrimSpace(in.Root) == "" {
		in.Root = "."
	}
	root, err := filepath.Abs(in.Root)
	if err != nil {
		return output{}, err
	}
	commands := in.Commands
	if len(commands) == 0 {
		commands = detectCommands(root, in.ChangedFiles)
	}
	if len(commands) == 0 {
		return output{Status: "passed", Runs: []runResult{}, FailedCommands: 0}, nil
	}

	results := make([]runResult, 0, len(commands))
	failed := 0
	for _, cmd := range commands {
		res := executeCommand(root, cmd, time.Duration(in.TimeoutSeconds)*time.Second, in.MaxOutputBytes)
		if res.ExitCode != 0 || res.TimedOut {
			failed++
		}
		results = append(results, res)
	}
	status := "passed"
	if failed > 0 {
		status = "failed"
	}
	return output{Status: status, Runs: results, FailedCommands: failed}, nil
}

func detectCommands(root string, changed []string) []string {
	commands := []string{}
	hasGo := fileExists(filepath.Join(root, "go.mod")) || hasExt(changed, ".go")
	hasNode := fileExists(filepath.Join(root, "package.json")) || hasExt(changed, ".js") || hasExt(changed, ".ts") || hasExt(changed, ".tsx")
	hasPy := fileExists(filepath.Join(root, "pyproject.toml")) || fileExists(filepath.Join(root, "requirements.txt")) || hasExt(changed, ".py")
	if hasGo {
		commands = append(commands, "go test ./...")
	}
	if hasNode {
		commands = append(commands, "npm test")
	}
	if hasPy {
		commands = append(commands, "pytest -q")
	}
	if fileExists(filepath.Join(root, "Makefile")) {
		commands = append(commands, "make test")
	}
	return unique(commands)
}

func executeCommand(root, command string, timeout time.Duration, maxBytes int) runResult {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	start := time.Now()
	cmd := exec.CommandContext(ctx, "bash", "-lc", command)
	cmd.Dir = root

	stdoutBuf := newLimitedBuffer(maxBytes)
	stderrBuf := newLimitedBuffer(maxBytes)
	cmd.Stdout = stdoutBuf
	cmd.Stderr = stderrBuf
	err := cmd.Run()

	exitCode := 0
	if err != nil {
		exitCode = 1
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		}
	}
	return runResult{
		Command:         command,
		ExitCode:        exitCode,
		DurationMs:      time.Since(start).Milliseconds(),
		Stdout:          stdoutBuf.String(),
		Stderr:          stderrBuf.String(),
		TimedOut:        ctx.Err() == context.DeadlineExceeded,
		OutputTruncated: stdoutBuf.Truncated || stderrBuf.Truncated,
	}
}

type limitedBuffer struct {
	buf       bytes.Buffer
	Limit     int
	Truncated bool
}

func newLimitedBuffer(limit int) *limitedBuffer {
	if limit <= 0 {
		limit = 20000
	}
	return &limitedBuffer{Limit: limit}
}

func (l *limitedBuffer) Write(p []byte) (int, error) {
	if l.buf.Len() >= l.Limit {
		l.Truncated = true
		return len(p), nil
	}
	remaining := l.Limit - l.buf.Len()
	if len(p) > remaining {
		l.buf.Write(p[:remaining])
		l.Truncated = true
		return len(p), nil
	}
	return l.buf.Write(p)
}

func (l *limitedBuffer) String() string {
	return l.buf.String()
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func hasExt(paths []string, ext string) bool {
	ext = strings.ToLower(ext)
	for _, p := range paths {
		if strings.EqualFold(filepath.Ext(p), ext) {
			return true
		}
	}
	return false
}

func unique(items []string) []string {
	seen := map[string]bool{}
	out := []string{}
	for _, item := range items {
		if seen[item] {
			continue
		}
		seen[item] = true
		out = append(out, item)
	}
	return out
}

func writeResponse(resp response) {
	b, _ := json.Marshal(resp)
	fmt.Println(string(b))
}
