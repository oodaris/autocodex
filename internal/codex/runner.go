package codex

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"time"
)

type Runner struct {
	CLIPath        string
	Model          string
	ExtraArgs      []string
	Mode           string
	ApprovalPolicy string
	SandboxMode    string
	Timeout        time.Duration
	Env            map[string]string
}

func (r Runner) Exec(ctx context.Context, prompt string) (string, error) {
	if prompt == "" {
		return "", fmt.Errorf("prompt is empty")
	}

	args := []string{"exec", prompt}
	if r.Model != "" {
		args = append(args, "--model", r.Model)
	}

	args = append(args, modeFlags(r.Mode, r.ApprovalPolicy, r.SandboxMode)...)
	args = append(args, r.ExtraArgs...)

	cmd := exec.CommandContext(ctx, r.CLIPath, args...)
	cmd.Env = mergeEnv(os.Environ(), r.Env)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return stdout.String(), fmt.Errorf("codex exec failed: %w; stderr: %s", err, stderr.String())
	}
	return stdout.String(), nil
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
