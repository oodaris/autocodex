package autonomy

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/oodaris/autocodex/internal/config"
	"github.com/oodaris/autocodex/internal/state"
)

func TestExtractJSONBlockMarkers(t *testing.T) {
	input := "foo\nACTIONS_JSON_START\n{\"version\":\"1.0\",\"summary\":\"ok\",\"next\":{\"type\":\"none\"}}\nACTIONS_JSON_END\nbar"
	output, err := extractJSONBlock(input)
	if err != nil {
		t.Fatalf("expected JSON block, got error: %v", err)
	}
	expected := "{\"version\":\"1.0\",\"summary\":\"ok\",\"next\":{\"type\":\"none\"}}"
	if output != expected {
		t.Fatalf("unexpected payload: %s", output)
	}
}

func TestExtractJSONBlockFenced(t *testing.T) {
	input := "```json\n{\"version\":\"1.0\",\"summary\":\"ok\",\"next\":{\"type\":\"none\"}}\n```"
	output, err := extractJSONBlock(input)
	if err != nil {
		t.Fatalf("expected JSON block, got error: %v", err)
	}
	expected := "{\"version\":\"1.0\",\"summary\":\"ok\",\"next\":{\"type\":\"none\"}}"
	if output != expected {
		t.Fatalf("unexpected payload: %s", output)
	}
}

func TestActionsFromRunValid(t *testing.T) {
	store := newTestStore(t)
	run, err := store.CreateRun()
	if err != nil {
		t.Fatalf("create run: %v", err)
	}
	artifactPath := filepath.Join(store.RunsDir, run.ID, "artifacts", "phase-final.txt")
	payload := "ACTIONS_JSON_START\n{\"version\":\"1.0\",\"summary\":\"ok\",\"next\":{\"type\":\"none\"}}\nACTIONS_JSON_END"
	if err := writeArtifact(artifactPath, payload); err != nil {
		t.Fatalf("write artifact: %v", err)
	}
	schemaPath := filepath.Join("..", "..", "docs", "contracts", "autonomy-actions.schema.json")
	ctrl := Controller{
		Config: configWithActionsSchema(schemaPath),
		Store:  store,
	}
	actions, err := ctrl.actionsFromRun(run.ID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if actions == nil || actions.Summary != "ok" {
		t.Fatalf("expected valid actions")
	}
}

func TestActionsFromRunMissing(t *testing.T) {
	store := newTestStore(t)
	run, err := store.CreateRun()
	if err != nil {
		t.Fatalf("create run: %v", err)
	}
	artifactPath := filepath.Join(store.RunsDir, run.ID, "artifacts", "phase-final.txt")
	if err := writeArtifact(artifactPath, "no actions here"); err != nil {
		t.Fatalf("write artifact: %v", err)
	}
	schemaPath := filepath.Join("..", "..", "docs", "contracts", "autonomy-actions.schema.json")
	ctrl := Controller{
		Config: configWithActionsSchema(schemaPath),
		Store:  store,
	}
	actions, err := ctrl.actionsFromRun(run.ID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if actions != nil {
		t.Fatalf("expected no actions")
	}
}

func TestActionsFromRunInvalidSchema(t *testing.T) {
	store := newTestStore(t)
	run, err := store.CreateRun()
	if err != nil {
		t.Fatalf("create run: %v", err)
	}
	artifactPath := filepath.Join(store.RunsDir, run.ID, "artifacts", "phase-final.txt")
	payload := "ACTIONS_JSON_START\n{\"version\":\"1.0\",\"summary\":\"ok\"}\nACTIONS_JSON_END"
	if err := writeArtifact(artifactPath, payload); err != nil {
		t.Fatalf("write artifact: %v", err)
	}
	schemaPath := filepath.Join("..", "..", "docs", "contracts", "autonomy-actions.schema.json")
	ctrl := Controller{
		Config: configWithActionsSchema(schemaPath),
		Store:  store,
	}
	_, err = ctrl.actionsFromRun(run.ID)
	if err == nil {
		t.Fatalf("expected schema error")
	}
}

func TestActionsFromRunInvalidJSON(t *testing.T) {
	store := newTestStore(t)
	run, err := store.CreateRun()
	if err != nil {
		t.Fatalf("create run: %v", err)
	}
	artifactPath := filepath.Join(store.RunsDir, run.ID, "artifacts", "phase-final.txt")
	payload := "ACTIONS_JSON_START\n{not-json}\nACTIONS_JSON_END"
	if err := writeArtifact(artifactPath, payload); err != nil {
		t.Fatalf("write artifact: %v", err)
	}
	schemaPath := filepath.Join("..", "..", "docs", "contracts", "autonomy-actions.schema.json")
	ctrl := Controller{
		Config: configWithActionsSchema(schemaPath),
		Store:  store,
	}
	_, err = ctrl.actionsFromRun(run.ID)
	if err == nil {
		t.Fatalf("expected json error")
	}
}

func newTestStore(t *testing.T) *state.Store {
	t.Helper()
	base := t.TempDir()
	store := state.NewStore(
		filepath.Join(base, "state"),
		filepath.Join(base, "runs"),
		filepath.Join(base, "memory"),
		filepath.Join(base, "logs"),
		filepath.Join(base, "artifacts"),
	)
	if err := store.InitDirs(); err != nil {
		t.Fatalf("init dirs: %v", err)
	}
	return store
}

func configWithActionsSchema(path string) config.Config {
	cfg := config.Config{}
	cfg.ApplyDefaults()
	cfg.Autonomy.ActionsSchema = path
	return cfg
}

func writeArtifact(path, payload string) error {
	return os.WriteFile(path, []byte(payload), 0o644)
}

func TestApplyActionsHighImpactRequiresGateVerdicts(t *testing.T) {
	cfg := config.Config{}
	cfg.ApplyDefaults()
	cfg.Autonomy.Harness.Enabled = true
	cfg.Autonomy.Harness.ImpactMode = "high"
	cfg.Beads.AutoUpdate = false

	ctrl := Controller{Config: cfg}
	stopReason, gateFailure, _, err := ctrl.applyActions("", &Actions{
		Version: "1.0",
		Summary: "test",
		Next:    ActionNext{Type: "none"},
		Gates:   &ActionGates{Blocking: false},
	})
	if err != nil {
		t.Fatalf("apply actions: %v", err)
	}
	if !gateFailure {
		t.Fatalf("expected gate failure when high-impact verdicts are missing")
	}
	if stopReason == "" {
		t.Fatalf("expected stop reason for high-impact gate failure")
	}
}

func TestApplyActionsHighImpactPassesWithRequiredVerdicts(t *testing.T) {
	cfg := config.Config{}
	cfg.ApplyDefaults()
	cfg.Autonomy.Harness.Enabled = true
	cfg.Autonomy.Harness.ImpactMode = "high"
	cfg.Beads.AutoUpdate = false

	qualityPassed := true
	evalScenarios := 6
	evalPassRate := 1.0
	evalSoftFails := 0
	ctrl := Controller{Config: cfg}
	stopReason, gateFailure, _, err := ctrl.applyActions("", &Actions{
		Version: "1.0",
		Summary: "test",
		Next:    ActionNext{Type: "none"},
		Gates: &ActionGates{
			Blocking:       false,
			CouncilVerdict: "GREEN",
			CriticVerdict:  "GO",
			QualityPassed:  &qualityPassed,
			EvalScenarios:  &evalScenarios,
			EvalPassRate:   &evalPassRate,
			EvalSoftFails:  &evalSoftFails,
		},
	})
	if err != nil {
		t.Fatalf("apply actions: %v", err)
	}
	if gateFailure {
		t.Fatalf("did not expect gate failure when required high-impact verdicts are present")
	}
	if stopReason != "" {
		t.Fatalf("expected empty stop reason, got %q", stopReason)
	}
}

func TestApplyActionsHighImpactFailsOnEvalThresholds(t *testing.T) {
	cfg := config.Config{}
	cfg.ApplyDefaults()
	cfg.Autonomy.Harness.Enabled = true
	cfg.Autonomy.Harness.ImpactMode = "high"
	cfg.Beads.AutoUpdate = false

	qualityPassed := true
	evalScenarios := 3
	evalPassRate := 0.5
	evalSoftFails := 2
	ctrl := Controller{Config: cfg}
	stopReason, gateFailure, _, err := ctrl.applyActions("", &Actions{
		Version: "1.0",
		Summary: "test",
		Next:    ActionNext{Type: "none"},
		Gates: &ActionGates{
			Blocking:       false,
			CouncilVerdict: "GREEN",
			CriticVerdict:  "GO",
			QualityPassed:  &qualityPassed,
			EvalScenarios:  &evalScenarios,
			EvalPassRate:   &evalPassRate,
			EvalSoftFails:  &evalSoftFails,
		},
	})
	if err != nil {
		t.Fatalf("apply actions: %v", err)
	}
	if !gateFailure {
		t.Fatalf("expected gate failure when eval thresholds are not met")
	}
	if stopReason == "" {
		t.Fatalf("expected stop reason for eval threshold failure")
	}
}
