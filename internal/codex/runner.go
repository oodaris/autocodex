package codex

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
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
		cmd.Stdin = strings.NewReader(prompt)
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		result.Stdout = stdout.String()
		result.Stderr = stderr.String()
		return result, fmt.Errorf("codex exec failed: %w; stderr: %s", err, stderr.String())
	}
	result.Stdout = stdout.String()
	result.Stderr = stderr.String()
	return result, nil
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
