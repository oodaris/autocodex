package autonomy

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/oodaris/autocodex/internal/codex"
	"github.com/oodaris/autocodex/internal/config"
	"github.com/oodaris/autocodex/internal/logging"
	"github.com/oodaris/autocodex/internal/skills"
	"github.com/oodaris/autocodex/internal/state"
)

type fakeExecutor struct{}

func (fakeExecutor) Exec(_ context.Context, prompt string) (codex.ExecResult, error) {
	result := codex.ExecResult{}
	switch {
	case strings.Contains(prompt, "Schema (MUST conform)") && strings.Contains(prompt, "Tasks Schema"):
		result.Stdout = `{"version":"1.0","generated_at":"2026-01-15T00:00:00Z","source_plan":"plan.md","tasks":[{"id":"autocodex-smoke","title":"Smoke bead","goal":"Verify autonomy loop","files":[],"acceptance_criteria":[]}]}`
		return result, nil
	case strings.Contains(prompt, "Template (fill in"):
		result.Stdout = "# Spec\n\nok\n"
		return result, nil
	case strings.Contains(prompt, "Phase: test"):
		result.Stdout = "tests ok\nACTIONS_JSON_START\n{\"version\":\"1.0\",\"summary\":\"done\",\"next\":{\"type\":\"none\"},\"updates\":{\"beads\":[{\"id\":\"autocodex-smoke\",\"status\":\"done\"}]},\"gates\":{\"review_required\":false,\"tests\":[],\"blocking\":false},\"stop\":null}\nACTIONS_JSON_END\n"
		return result, nil
	default:
		result.Stdout = "ok\n"
		return result, nil
	}
}

func TestAutonomyLoopSmoke(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("smoke test uses a shell-based bd stub")
	}

	tempDir := t.TempDir()
	stateDir := filepath.Join(tempDir, "state")
	if err := os.MkdirAll(stateDir, 0o755); err != nil {
		t.Fatal(err)
	}
	specTemplate := filepath.Join(tempDir, "spec-template.md")
	planTemplate := filepath.Join(tempDir, "plan-template.md")
	if err := os.WriteFile(specTemplate, []byte("# Spec Template\n"), 0o644); err != nil {
		t.Fatalf("write spec template: %v", err)
	}
	if err := os.WriteFile(planTemplate, []byte("# Plan Template\n"), 0o644); err != nil {
		t.Fatalf("write plan template: %v", err)
	}

	bdBin := filepath.Join(tempDir, "bd")
	bdState := filepath.Join(tempDir, "bd-state")
	if err := writeBDFake(bdBin); err != nil {
		t.Fatalf("write fake bd: %v", err)
	}

	origPath := os.Getenv("PATH")
	t.Setenv("PATH", fmt.Sprintf("%s:%s", tempDir, origPath))
	t.Setenv("BD_STATE", bdState)

	root, err := repoRoot()
	if err != nil {
		t.Fatalf("repo root: %v", err)
	}

	cfg := config.Config{
		Version: "v1",
		Mode:    "yolo",
	}
	cfg.ApplyDefaults()
	cfg.Paths.StateDir = stateDir
	cfg.Skills.Paths = []string{filepath.Join(root, "skills")}
	cfg.Autonomy.Enabled = true
	cfg.Autonomy.SpecTemplate = specTemplate
	cfg.Autonomy.PlanTemplate = planTemplate
	cfg.Autonomy.TasksSchema = filepath.Join(root, "docs/contracts/autonomy-tasks.schema.json")
	cfg.Autonomy.ActionsSchema = filepath.Join(root, "docs/contracts/autonomy-actions.schema.json")
	cfg.Autonomy.TasksOutputTemplate = filepath.Join(tempDir, "%s-tasks.json")
	cfg.Codex.TimeoutSeconds = 1
	cfg.Beads.Enabled = true
	cfg.Beads.AutoCreate = true
	cfg.Beads.AutoUpdate = true

	store := state.NewStore(cfg.StateDir(), cfg.RunsDir(), cfg.MemoryDir(), cfg.LogsDir(), cfg.ArtifactsDir())
	loader := skills.Loader{Paths: cfg.Skills.Paths}

	controller := Controller{
		Config: cfg,
		Logger: logging.NewLogger("error"),
		Store:  store,
		Skills: loader,
		Codex:  fakeExecutor{},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if _, err := controller.Run(ctx, Input{Task: "Smoke test autonomy loop"}); err != nil {
		t.Fatalf("controller run failed: %v", err)
	}

	content, err := os.ReadFile(bdState)
	if err != nil {
		t.Fatalf("read bd state: %v", err)
	}
	if !strings.Contains(string(content), "autocodex-smoke|done") {
		t.Fatalf("expected bead to be marked done, got: %s", string(content))
	}
}

func writeBDFake(path string) error {
	script := `#!/bin/sh
STATE="${BD_STATE:-/tmp/bd-state}"
cmd="$1"
shift
case "$cmd" in
  create)
    id=""
    title=""
    priority="0"
    while [ $# -gt 0 ]; do
      case "$1" in
        --id) id="$2"; shift 2;;
        --title) title="$2"; shift 2;;
        --priority) priority="$2"; shift 2;;
        *) shift;;
      esac
    done
    [ -z "$id" ] && exit 1
    [ -f "$STATE" ] || touch "$STATE"
    grep -v "^$id|" "$STATE" > "$STATE.tmp" 2>/dev/null || true
    mv "$STATE.tmp" "$STATE"
    echo "$id|todo|$title|$priority" >> "$STATE"
    exit 0
    ;;
  show)
    id="$1"
    if [ -f "$STATE" ] && grep -q "^$id|" "$STATE"; then
      exit 0
    fi
    exit 1
    ;;
  update)
    id="$1"
    shift
    status=""
    while [ $# -gt 0 ]; do
      case "$1" in
        --status) status="$2"; shift 2;;
        *) shift;;
      esac
    done
    [ -z "$status" ] && exit 1
    if [ -f "$STATE" ]; then
      awk -F'|' -v id="$id" -v status="$status" 'BEGIN{OFS="|"} $1==id{$2=status} {print}' "$STATE" > "$STATE.tmp"
      mv "$STATE.tmp" "$STATE"
    fi
    exit 0
    ;;
  ready)
    if [ "$1" = "--json" ]; then
      if [ ! -f "$STATE" ]; then
        echo "[]"
        exit 0
      fi
      awk -F'|' 'BEGIN{first=1; print "["} $2=="todo"{if(!first){print ","}; first=0; printf "{\"id\":\"%s\",\"title\":\"%s\",\"status\":\"%s\",\"priority\":0}", $1, $3, $2} END{print "]"}' "$STATE"
      exit 0
    fi
    ;;
  dep)
    exit 0
    ;;
esac
echo "unknown command" >&2
exit 1
`
	if err := os.WriteFile(path, []byte(script), 0o755); err != nil {
		return err
	}
	return nil
}

func repoRoot() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", fmt.Errorf("go.mod not found")
		}
		dir = parent
	}
}
