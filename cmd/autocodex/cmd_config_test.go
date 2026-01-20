package main

import (
	"strings"
	"testing"
)

func TestRunConfigOutput(t *testing.T) {
	out := captureStdout(t, func() {
		runConfig([]string{"--config", "custom.yaml"})
	})
	if !strings.Contains(out, "custom.yaml") {
		t.Fatalf("expected config path output, got %s", out)
	}
}
