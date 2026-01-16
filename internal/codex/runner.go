package codex

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

type Runner struct {
	CLIPath         string
	Model           string
	ReasoningEffort string
	ExtraArgs       []string
	Mode            string
	ApprovalPolicy  string
	SandboxMode     string
	JSONOutput      bool
	OutputLast      bool
	PromptStdin     bool
	Timeout         time.Duration
	IdleTimeout     time.Duration
	Env             map[string]string
}

type Executor interface {
	Exec(ctx context.Context, prompt string) (ExecResult, error)
}

type ExecResult struct {
	Stdout string
	Stderr string
}

func (r Runner) Exec(ctx context.Context, prompt string) (ExecResult, error) {
	result := ExecResult{}
	if prompt == "" {
		return result, fmt.Errorf("prompt is empty")
	}

	useStdin := r.PromptStdin || strings.Contains(prompt, "\n") || len(prompt) > 4000
	args := []string{"exec"}
	if useStdin {
		args = append(args, "-")
	} else {
		args = append(args, prompt)
	}
	if r.Model != "" {
		args = append(args, "--model", r.Model)
	}
	if r.ReasoningEffort != "" {
		args = append(args, "-c", fmt.Sprintf(`reasoning.effort=%q`, r.ReasoningEffort))
	}
	if r.JSONOutput {
		args = append(args, "--json")
	}

	outputPath := outputPathFromContext(ctx)
	if r.OutputLast && strings.TrimSpace(outputPath) != "" {
		if err := os.MkdirAll(filepath.Dir(outputPath), 0o755); err != nil {
			return result, fmt.Errorf("create output directory: %w", err)
		}
		args = append(args, "--output-last-message", outputPath)
	}

	args = append(args, modeFlags(r.Mode, r.ApprovalPolicy, r.SandboxMode)...)
	args = append(args, r.ExtraArgs...)

	cmd := exec.CommandContext(ctx, r.CLIPath, args...)
	cmd.Env = mergeEnv(os.Environ(), r.Env)
	if useStdin {
		if !strings.HasSuffix(prompt, "\n") {
			prompt += "\n"
		}
		cmd.Stdin = strings.NewReader(prompt)
	}

	stdoutPipe, err := cmd.StdoutPipe()
	if err != nil {
		return result, fmt.Errorf("stdout pipe: %w", err)
	}
	stderrPipe, err := cmd.StderrPipe()
	if err != nil {
		return result, fmt.Errorf("stderr pipe: %w", err)
	}

	if err := cmd.Start(); err != nil {
		return result, fmt.Errorf("codex exec start failed: %w", err)
	}
	if reporter := pidReporterFromContext(ctx); reporter != nil && cmd.Process != nil {
		reporter(cmd.Process.Pid)
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	var mu sync.Mutex
	lastActivity := time.Now()
	touch := func() {
		mu.Lock()
		lastActivity = time.Now()
		mu.Unlock()
	}
	getLastActivity := func() time.Time {
		mu.Lock()
		defer mu.Unlock()
		return lastActivity
	}
	var needsFollowUp bool
	var needsFollowUpAt time.Time
	var needsMu sync.Mutex
	markNeedsFollowUp := func() {
		needsMu.Lock()
		needsFollowUp = true
		if needsFollowUpAt.IsZero() {
			needsFollowUpAt = time.Now()
		}
		needsMu.Unlock()
	}
	getNeedsFollowUp := func() bool {
		needsMu.Lock()
		defer needsMu.Unlock()
		return needsFollowUp
	}

	var wg sync.WaitGroup
	copyWithActivity := func(dst *bytes.Buffer, extra io.Writer, src io.Reader, detectFollowUp bool) {
		defer wg.Done()
		buf := make([]byte, 4096)
		var writer io.Writer = dst
		if extra != nil {
			writer = io.MultiWriter(dst, extra)
		}
		for {
			n, readErr := src.Read(buf)
			if n > 0 {
				_, _ = writer.Write(buf[:n])
				touch()
				if detectFollowUp {
					lower := strings.ToLower(string(buf[:n]))
					if strings.Contains(lower, "needs_follow_up: true") {
						markNeedsFollowUp()
					}
				}
			}
			if readErr != nil {
				return
			}
		}
	}

	sinks := outputSinksFromContext(ctx)

	wg.Add(2)
	go copyWithActivity(&stdout, sinks.Stdout, stdoutPipe, false)
	go copyWithActivity(&stderr, sinks.Stderr, stderrPipe, true)

	done := make(chan error, 1)
	go func() {
		done <- cmd.Wait()
	}()

	const outputSettle = 8 * time.Second
	const followUpIdle = 12 * time.Second
	var lastModTime time.Time
	var outputStableSince time.Time
	var terminatedForOutput bool
	var idleTimeoutTriggered bool
	idleTimeout := r.IdleTimeout
	if override := idleTimeoutFromContext(ctx); override > 0 {
		idleTimeout = override
	}

	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case err := <-done:
			wg.Wait()
			result.Stdout = stdout.String()
			result.Stderr = stderr.String()
			if terminatedForOutput {
				return result, nil
			}
			if getNeedsFollowUp() {
				return result, fmt.Errorf("codex exec requires follow-up; stderr: %s", result.Stderr)
			}
			if idleTimeoutTriggered {
				return result, fmt.Errorf("codex exec idle timeout after %s; stderr: %s", r.IdleTimeout, result.Stderr)
			}
			if err != nil {
				return result, fmt.Errorf("codex exec failed: %w; stderr: %s", err, result.Stderr)
			}
			return result, nil
		case <-ctx.Done():
			_ = cmd.Process.Kill()
			err := <-done
			wg.Wait()
			result.Stdout = stdout.String()
			result.Stderr = stderr.String()
			if terminatedForOutput {
				return result, nil
			}
			if getNeedsFollowUp() {
				return result, fmt.Errorf("codex exec requires follow-up; stderr: %s", result.Stderr)
			}
			if idleTimeoutTriggered {
				return result, fmt.Errorf("codex exec idle timeout after %s; stderr: %s", r.IdleTimeout, result.Stderr)
			}
			return result, fmt.Errorf("codex exec timed out: %w; stderr: %s", err, result.Stderr)
		case <-ticker.C:
			if r.OutputLast && strings.TrimSpace(outputPath) != "" {
				info, statErr := os.Stat(outputPath)
				if statErr == nil && info.Size() > 0 {
					modTime := info.ModTime()
					if modTime.After(lastModTime) {
						lastModTime = modTime
						outputStableSince = time.Now()
						touch()
					}
					if !outputStableSince.IsZero() &&
						time.Since(outputStableSince) >= outputSettle &&
						time.Since(getLastActivity()) >= outputSettle {
						terminatedForOutput = true
						_ = cmd.Process.Kill()
					}
				}
			}
			if idleTimeout > 0 && !idleTimeoutTriggered &&
				time.Since(getLastActivity()) >= idleTimeout {
				idleTimeoutTriggered = true
				_ = cmd.Process.Kill()
			}
			if getNeedsFollowUp() && time.Since(getLastActivity()) >= followUpIdle {
				_ = cmd.Process.Kill()
			}
		}
	}
}

func modeFlags(mode, approvalPolicy, sandboxMode string) []string {
	if mode == "yolo" {
		return []string{"--yolo"}
	}

	flags := []string{}
	if sandboxMode != "" {
		flags = append(flags, "--sandbox", sandboxMode)
	}
	if approvalPolicy != "" {
		flags = append(flags, "--ask-for-approval", approvalPolicy)
	}
	if len(flags) == 0 {
		flags = append(flags, "--full-auto")
	}
	return flags
}

func mergeEnv(base []string, override map[string]string) []string {
	env := append([]string{}, base...)
	for key, value := range override {
		env = append(env, fmt.Sprintf("%s=%s", key, value))
	}
	return env
}
