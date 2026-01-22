package autonomy

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/oodaris/autocodex/internal/codex"
	"github.com/oodaris/autocodex/internal/config"
	"github.com/oodaris/autocodex/internal/logging"
	"github.com/oodaris/autocodex/internal/skills"
	"github.com/oodaris/autocodex/internal/state"
)

type SmokeHarnessOptions struct {
	RequireActions *bool
	RequireNext    *bool
	RequireBD      *bool
}

type SmokeHarness struct {
	TempDir    string
	Config     config.Config
	Store      *state.Store
	Controller Controller
	BDState    string
}

func newSmokeHarness(t *testing.T, exec codex.Executor, opts SmokeHarnessOptions) *SmokeHarness {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("smoke test uses a shell-based bd stub")
	}

	tempDir := t.TempDir()
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

	cfg := config.Config{Version: "v1", Mode: "yolo"}
	cfg.ApplyDefaults()
	cfg.Paths.StateDir = filepath.Join(tempDir, "state")
	cfg.Skills.Paths = []string{filepath.Join(root, "skills")}
	cfg.Autonomy.Enabled = true
	cfg.Autonomy.SpecTemplate = specTemplate
	cfg.Autonomy.PlanTemplate = planTemplate
	cfg.Autonomy.TasksSchema = filepath.Join(root, "docs/contracts/autonomy-tasks.schema.json")
	cfg.Autonomy.ActionsSchema = filepath.Join(root, "docs/contracts/autonomy-actions.schema.json")
	cfg.Autonomy.TasksOutputTemplate = filepath.Join(tempDir, "%s-tasks.json")
	if opts.RequireActions != nil {
		cfg.Autonomy.RequireActions = opts.RequireActions
	}
	if opts.RequireNext != nil {
		cfg.Autonomy.RequireNext = opts.RequireNext
	}
	if opts.RequireBD != nil {
		cfg.Autonomy.RequireBD = opts.RequireBD
	}
	cfg.Codex.TimeoutSeconds = 1
	cfg.Beads.Enabled = true
	cfg.Beads.AutoCreate = true
	cfg.Beads.AutoUpdate = true

	store := state.NewStore(cfg.StateDir(), cfg.RunsDir(), cfg.MemoryDir(), cfg.LogsDir(), cfg.ArtifactsDir())
	loader := skills.Loader{Paths: cfg.Skills.Paths}
	controller := Controller{
		Config: cfg,
		Logger: logging.NewLogger("error", "json"),
		Store:  store,
		Skills: loader,
		Codex:  exec,
	}

	return &SmokeHarness{
		TempDir:    tempDir,
		Config:     cfg,
		Store:      store,
		Controller: controller,
		BDState:    bdState,
	}
}

func (h *SmokeHarness) ReadBDState(t *testing.T) string {
	t.Helper()
	content, err := os.ReadFile(h.BDState)
	if err != nil {
		t.Fatalf("read bd state: %v", err)
	}
	return string(content)
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
  close)
    id="$1"
    if [ -f "$STATE" ]; then
      awk -F'|' -v id="$id" 'BEGIN{OFS="|"} $1==id{$2="done"} {print}' "$STATE" > "$STATE.tmp"
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
