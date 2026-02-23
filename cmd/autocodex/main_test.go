package main

import (
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/oodaris/autocodex/internal/config"
	"github.com/oodaris/autocodex/internal/state"
)

func captureStdout(t *testing.T, fn func()) string {
	t.Helper()
	original := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("pipe: %v", err)
	}
	os.Stdout = w
	fn()
	_ = w.Close()
	os.Stdout = original
	data, _ := io.ReadAll(r)
	return string(data)
}

func TestIsCommand(t *testing.T) {
	if !isCommand("run") {
		t.Fatalf("expected run to be a command")
	}
	if !isCommand("harness") {
		t.Fatalf("expected harness to be a command")
	}
	if isCommand("nope") {
		t.Fatalf("expected unknown command to be false")
	}
}

func TestIsVersionArg(t *testing.T) {
	if !isVersionArg("--version") || !isVersionArg("-v") {
		t.Fatalf("expected version args to be true")
	}
	if isVersionArg("version") {
		t.Fatalf("expected version command to be false for isVersionArg")
	}
}

func TestUsageOutput(t *testing.T) {
	out := captureStdout(t, usage)
	if !strings.Contains(out, "Usage: autocodex") {
		t.Fatalf("expected usage output")
	}
	if !strings.Contains(out, "Commands:") {
		t.Fatalf("expected commands list")
	}
}

func TestPrintVersionOutput(t *testing.T) {
	original := version
	version = "1.2.3-test"
	defer func() { version = original }()
	out := captureStdout(t, printVersion)
	out = strings.TrimSpace(out)
	if out != version {
		t.Fatalf("expected version %s, got %s", version, out)
	}
}

func TestRequestRunAction(t *testing.T) {
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
	run, err := store.CreateRun()
	if err != nil {
		t.Fatalf("create run: %v", err)
	}
	control, err := requestRunAction(store, run.ID, "kill", "shutdown")
	if err != nil {
		t.Fatalf("request run action: %v", err)
	}
	if control.LastAction == nil || *control.LastAction != "kill" {
		t.Fatalf("expected last action")
	}
	if control.StopReason == nil || *control.StopReason != "shutdown" {
		t.Fatalf("expected stop reason")
	}
}

func TestCollectRunStatuses(t *testing.T) {
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
	run, err := store.CreateRun()
	if err != nil {
		t.Fatalf("create run: %v", err)
	}
	action := "stop"
	now := time.Now().UTC()
	if err := store.SaveRunControl(state.RunControl{
		RunID:        run.ID,
		Status:       "running",
		LastAction:   &action,
		LastActionAt: &now,
		UpdatedAt:    now,
	}); err != nil {
		t.Fatalf("save run control: %v", err)
	}
	if err := store.SaveRunFeedback(state.RunFeedback{
		RunID:     run.ID,
		UpdatedAt: now,
		Sources:   []string{"memory"},
	}); err != nil {
		t.Fatalf("save run feedback: %v", err)
	}

	statuses, err := collectRunStatuses(store, run.ID)
	if err != nil {
		t.Fatalf("collect statuses: %v", err)
	}
	if len(statuses) != 1 {
		t.Fatalf("expected one status")
	}
	if statuses[0].LastAction == nil || *statuses[0].LastAction != "stop" {
		t.Fatalf("expected last action in status")
	}
	if statuses[0].Feedback == nil || len(statuses[0].Feedback.Sources) != 1 {
		t.Fatalf("expected feedback in status")
	}
}

func TestParseSources(t *testing.T) {
	sources := parseSources("memory, events", []string{"artifacts"})
	if len(sources) != 2 || sources[0] != "memory" || sources[1] != "events" {
		t.Fatalf("unexpected sources")
	}
	fallback := parseSources("", []string{"artifacts"})
	if len(fallback) != 1 || fallback[0] != "artifacts" {
		t.Fatalf("expected fallback sources")
	}
}

func TestParseList(t *testing.T) {
	items := parseList(" running,completed ,, failed ")
	if len(items) != 3 {
		t.Fatalf("expected 3 items")
	}
	if items[0] != "running" || items[1] != "completed" || items[2] != "failed" {
		t.Fatalf("unexpected parseList output: %v", items)
	}
	if parseList("") != nil {
		t.Fatalf("expected nil for empty input")
	}
}

func TestFilterRunStatuses(t *testing.T) {
	statuses := []RunStatus{{Status: "running"}, {Status: "completed"}, {Status: "failed"}}
	filtered := filterRunStatuses(statuses, []string{"running", "failed"})
	if len(filtered) != 2 {
		t.Fatalf("expected 2 statuses")
	}
	if filtered[0].Status != "running" || filtered[1].Status != "failed" {
		t.Fatalf("unexpected filter result: %v", filtered)
	}
	if len(filterRunStatuses(statuses, nil)) != 3 {
		t.Fatalf("expected no filtering when filters empty")
	}
}

func TestLimitRunStatuses(t *testing.T) {
	statuses := []RunStatus{{ID: "a"}, {ID: "b"}, {ID: "c"}}
	limited := limitRunStatuses(statuses, 2)
	if len(limited) != 2 {
		t.Fatalf("expected 2 statuses")
	}
	if limited[0].ID != "b" || limited[1].ID != "c" {
		t.Fatalf("unexpected limit result: %v", limited)
	}
	if len(limitRunStatuses(statuses, 0)) != 3 {
		t.Fatalf("expected no limit when <=0")
	}
}

func TestLatestRunID(t *testing.T) {
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
	if _, err := store.CreateRun(); err != nil {
		t.Fatalf("create run: %v", err)
	}
	id, err := latestRunID(store)
	if err != nil {
		t.Fatalf("latest run: %v", err)
	}
	if id == "" {
		t.Fatalf("expected run id")
	}
}

func TestBootstrapAutonomyAssets(t *testing.T) {
	base := t.TempDir()
	cfg := config.Config{
		Autonomy: config.AutonomyConfig{
			Enabled:       true,
			SpecTemplate:  filepath.Join(base, "docs/specs/TEMPLATE.md"),
			PlanTemplate:  filepath.Join(base, "docs/plans/TEMPLATE.md"),
			TasksSchema:   filepath.Join(base, "docs/contracts/autonomy-tasks.schema.json"),
			ActionsSchema: filepath.Join(base, "docs/contracts/autonomy-actions.schema.json"),
		},
	}
	if err := bootstrapAutonomyAssets(cfg, false); err != nil {
		t.Fatalf("bootstrap autonomy assets: %v", err)
	}
	spec, err := os.ReadFile(cfg.Autonomy.SpecTemplate)
	if err != nil {
		t.Fatalf("read spec template: %v", err)
	}
	if !strings.Contains(string(spec), "# <spec-title>") {
		t.Fatalf("unexpected spec template content")
	}
	actions, err := os.ReadFile(cfg.Autonomy.ActionsSchema)
	if err != nil {
		t.Fatalf("read actions schema: %v", err)
	}
	if !strings.Contains(string(actions), "\"autocodex autonomy actions\"") {
		t.Fatalf("unexpected actions schema content")
	}
}

func TestBootstrapSkills(t *testing.T) {
	base := t.TempDir()
	root := filepath.Join(base, "skills")
	if err := bootstrapSkills(root, false); err != nil {
		t.Fatalf("bootstrap skills: %v", err)
	}
	path := filepath.Join(root, "core-qna-synthesis", "SKILL.md")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read skill: %v", err)
	}
	if !strings.Contains(string(data), "core-qna-synthesis") {
		t.Fatalf("unexpected skill content")
	}
}

func TestEnsureConfigEmbeddedFallback(t *testing.T) {
	base := t.TempDir()
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("get wd: %v", err)
	}
	if err := os.Chdir(base); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	defer func() {
		_ = os.Chdir(cwd)
	}()

	path := filepath.Join(base, "autocodex.yaml")
	if err := ensureConfig(path); err != nil {
		t.Fatalf("ensure config: %v", err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read config: %v", err)
	}
	if !strings.Contains(string(data), "autonomy:") {
		t.Fatalf("expected autonomy section in embedded config")
	}
	if !strings.Contains(string(data), "skills:") {
		t.Fatalf("expected skills section in embedded config")
	}
}

func TestEmbeddedConfigMatchesExample(t *testing.T) {
	path := filepath.Clean(filepath.Join("..", "..", "config.example.yaml"))
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read config.example.yaml: %v", err)
	}
	if string(data) != string(embeddedConfigExample) {
		t.Fatalf("embedded config does not match config.example.yaml")
	}
}

func TestBootstrapRepoEndToEnd(t *testing.T) {
	base := t.TempDir()
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("get wd: %v", err)
	}
	if err := os.Chdir(base); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	defer func() {
		_ = os.Chdir(cwd)
	}()

	if err := bootstrapRepo("autocodex.yaml", false, false, false); err != nil {
		t.Fatalf("bootstrap repo: %v", err)
	}

	expectedFiles := []string{
		"autocodex.yaml",
		filepath.Join("skills", "core-qna-synthesis", "SKILL.md"),
		filepath.Join("skills", "core-holistic-planning-and-tracking", "SKILL.md"),
		filepath.Join("skills", "core-ask-questions-if-underspecified", "SKILL.md"),
		filepath.Join("docs", "specs", "TEMPLATE.md"),
		filepath.Join("docs", "plans", "TEMPLATE.md"),
		filepath.Join("docs", "contracts", "autonomy-tasks.schema.json"),
		filepath.Join("docs", "contracts", "autonomy-actions.schema.json"),
		filepath.Join(".autocodex", "memory", "TODO.md"),
	}
	for _, path := range expectedFiles {
		if _, err := os.Stat(path); err != nil {
			t.Fatalf("expected %s to exist: %v", path, err)
		}
	}
}

func TestWriteFileIfMissingForce(t *testing.T) {
	path := filepath.Join(t.TempDir(), "docs/specs/TEMPLATE.md")
	if err := writeFileIfMissing(path, []byte("first"), false); err != nil {
		t.Fatalf("write file: %v", err)
	}
	if err := writeFileIfMissing(path, []byte("second"), false); err != nil {
		t.Fatalf("write file without force: %v", err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read file: %v", err)
	}
	if string(data) != "first" {
		t.Fatalf("expected file to remain unchanged")
	}
	if err := writeFileIfMissing(path, []byte("third"), true); err != nil {
		t.Fatalf("write file with force: %v", err)
	}
	data, err = os.ReadFile(path)
	if err != nil {
		t.Fatalf("read file: %v", err)
	}
	if string(data) != "third" {
		t.Fatalf("expected file to be overwritten")
	}
}

func TestResolveTaskInputStdin(t *testing.T) {
	payload, err := resolveTaskInput("", "", true, nil, strings.NewReader("  hello world  "))
	if err != nil {
		t.Fatalf("resolve task stdin: %v", err)
	}
	if payload != "hello world" {
		t.Fatalf("expected trimmed payload")
	}
}

func TestResolveTaskInputStdinEmpty(t *testing.T) {
	_, err := resolveTaskInput("", "", true, nil, strings.NewReader("   "))
	if err == nil {
		t.Fatalf("expected error for empty stdin")
	}
}

func TestResolveTaskInputStdinConflict(t *testing.T) {
	_, err := resolveTaskInput("task", "", true, nil, strings.NewReader("ignored"))
	if err == nil {
		t.Fatalf("expected conflict error")
	}
}

func TestResolveTaskInputArgsFallback(t *testing.T) {
	payload, err := resolveTaskInput("", "", false, []string{"hello", "there"}, strings.NewReader(""))
	if err != nil {
		t.Fatalf("resolve task args: %v", err)
	}
	if payload != "hello there" {
		t.Fatalf("expected joined args payload")
	}
}

func TestSelectRunStatusesLatest(t *testing.T) {
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

	firstID := "20260101T000000Z-aaaa"
	secondID := "20260102T000000Z-bbbb"
	createRunWithID(t, store, firstID)
	createRunWithID(t, store, secondID)

	statuses, err := selectRunStatuses(store, "", true)
	if err != nil {
		t.Fatalf("select run statuses: %v", err)
	}
	if len(statuses) != 1 {
		t.Fatalf("expected one status")
	}
	if statuses[0].ID != secondID {
		t.Fatalf("expected latest status")
	}
}

func TestSelectRunStatusesLatestConflict(t *testing.T) {
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
	_, err := selectRunStatuses(store, "20260101T000000Z-aaaa", true)
	if err == nil {
		t.Fatalf("expected conflict error")
	}
}

func createRunWithID(t *testing.T, store *state.Store, id string) {
	t.Helper()
	runDir := filepath.Join(store.RunsDir, id)
	if err := os.MkdirAll(runDir, 0o755); err != nil {
		t.Fatalf("mkdir run dir: %v", err)
	}
	run := &state.Run{
		ID:        id,
		Status:    "running",
		StartedAt: time.Now().UTC(),
	}
	if err := store.SaveRun(run); err != nil {
		t.Fatalf("save run: %v", err)
	}
}
