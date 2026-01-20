package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/oodaris/autocodex/internal/config"
	"github.com/oodaris/autocodex/internal/state"
)

type RunStatus struct {
	ID           string             `json:"id"`
	Status       string             `json:"status"`
	CurrentPhase string             `json:"current_phase"`
	Iterations   int                `json:"iterations"`
	StartedAt    time.Time          `json:"started_at"`
	FinishedAt   *time.Time         `json:"finished_at"`
	LastAction   *string            `json:"last_action,omitempty"`
	LastActionAt *time.Time         `json:"last_action_at,omitempty"`
	StopReason   *string            `json:"stop_reason,omitempty"`
	LastError    *string            `json:"last_error,omitempty"`
	Feedback     *state.RunFeedback `json:"feedback,omitempty"`
	Control      *state.RunControl  `json:"control,omitempty"`
}

func runStatus(args []string) {
	fs := flag.NewFlagSet("status", flag.ExitOnError)
	configPath := fs.String("config", config.ResolveConfigPath(), "Config file path")
	runID := fs.String("run", "", "run id (optional)")
	latest := fs.Bool("latest", false, "show latest run only")
	jsonOut := fs.Bool("json", false, "output JSON")
	table := fs.Bool("table", false, "output table with headers")
	statusFilter := fs.String("status", "", "comma-separated status filter (running, completed, failed, canceled)")
	limit := fs.Int("limit", 0, "limit number of runs shown (0 = all)")
	fs.Parse(args)

	cfg, err := config.Load(*configPath)
	if err != nil {
		exitErr(err)
	}
	store := state.NewStore(cfg.StateDir(), cfg.RunsDir(), cfg.MemoryDir(), cfg.LogsDir(), cfg.ArtifactsDir())
	if finalized, err := store.FinalizeStaleRuns(cfg.Loop.StopConditions.MaxHeartbeatSeconds, "stale_after"); err == nil {
		_ = finalized
	}
	statuses, err := selectRunStatuses(store, *runID, *latest)
	if err != nil {
		exitErr(err)
	}
	statuses = filterRunStatuses(statuses, parseList(*statusFilter))
	statuses = limitRunStatuses(statuses, *limit)
	if len(statuses) == 0 {
		fmt.Println("No runs found")
		return
	}
	if *jsonOut {
		writeJSON(statuses)
		return
	}
	if *table {
		printRunTable(statuses)
		return
	}
	printRunRows(statuses)
}

func runRuns(args []string) {
	runStatus(append([]string{"--table"}, args...))
}

func selectRunStatuses(store *state.Store, runID string, latest bool) ([]RunStatus, error) {
	if latest && strings.TrimSpace(runID) != "" {
		return nil, fmt.Errorf("--latest cannot be used with --run")
	}
	statuses, err := collectRunStatuses(store, runID)
	if err != nil {
		return nil, err
	}
	if latest {
		return filterLatest(statuses), nil
	}
	return statuses, nil
}

func filterLatest(statuses []RunStatus) []RunStatus {
	if len(statuses) == 0 {
		return statuses
	}
	return statuses[len(statuses)-1:]
}

func collectRunStatuses(store *state.Store, runID string) ([]RunStatus, error) {
	var runs []state.Run
	if runID != "" {
		run, err := store.GetRun(runID)
		if err != nil {
			return nil, err
		}
		runs = []state.Run{run}
	} else {
		list, err := store.ListRuns()
		if err != nil {
			return nil, err
		}
		runs = list
	}

	statuses := make([]RunStatus, 0, len(runs))
	for _, run := range runs {
		control, err := store.GetRunControl(run.ID)
		if err != nil {
			return nil, err
		}
		feedback, err := store.GetRunFeedback(run.ID)
		if err != nil {
			return nil, err
		}
		statuses = append(statuses, RunStatus{
			ID:           run.ID,
			Status:       run.Status,
			CurrentPhase: run.CurrentPhase,
			Iterations:   run.Iterations,
			StartedAt:    run.StartedAt,
			FinishedAt:   run.FinishedAt,
			LastAction:   actionValue(control),
			LastActionAt: actionAtValue(control),
			StopReason:   stopReasonValue(control),
			LastError:    lastErrorValue(control),
			Feedback:     feedback,
			Control:      control,
		})
	}
	return statuses, nil
}

func actionValue(control *state.RunControl) *string {
	if control == nil {
		return nil
	}
	return control.LastAction
}

func actionAtValue(control *state.RunControl) *time.Time {
	if control == nil {
		return nil
	}
	return control.LastActionAt
}

func stopReasonValue(control *state.RunControl) *string {
	if control == nil {
		return nil
	}
	return control.StopReason
}

func lastErrorValue(control *state.RunControl) *string {
	if control == nil {
		return nil
	}
	return control.LastError
}

func emptyOr(value *string) string {
	if value == nil || *value == "" {
		return "—"
	}
	return *value
}

func filterRunStatuses(statuses []RunStatus, filters []string) []RunStatus {
	if len(filters) == 0 {
		return statuses
	}
	allowed := map[string]bool{}
	for _, item := range filters {
		if item == "" {
			continue
		}
		allowed[strings.ToLower(item)] = true
	}
	if len(allowed) == 0 {
		return statuses
	}
	filtered := make([]RunStatus, 0, len(statuses))
	for _, status := range statuses {
		if allowed[strings.ToLower(status.Status)] {
			filtered = append(filtered, status)
		}
	}
	return filtered
}

func limitRunStatuses(statuses []RunStatus, limit int) []RunStatus {
	if limit <= 0 || len(statuses) <= limit {
		return statuses
	}
	return statuses[len(statuses)-limit:]
}

func parseList(input string) []string {
	if strings.TrimSpace(input) == "" {
		return nil
	}
	parts := strings.Split(input, ",")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		item := strings.TrimSpace(part)
		if item == "" {
			continue
		}
		out = append(out, item)
	}
	return out
}

func printRunRows(statuses []RunStatus) {
	for _, status := range statuses {
		fmt.Printf(
			"%s\t%s\t%s\t%d\t%s\t%s\t%s\n",
			status.ID,
			status.Status,
			status.CurrentPhase,
			status.Iterations,
			emptyOr(status.LastAction),
			emptyOr(status.StopReason),
			emptyOr(status.LastError),
		)
	}
}

func printRunTable(statuses []RunStatus) {
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "#\tRUN ID\tSTATUS\tPHASE\tITERS\tSTARTED\tFINISHED\tLAST ACTION\tSTOP REASON\tLAST ERROR")
	for idx, status := range statuses {
		started := formatTime(status.StartedAt)
		finished := formatTimePtr(status.FinishedAt)
		fmt.Fprintf(
			w,
			"%d\t%s\t%s\t%s\t%d\t%s\t%s\t%s\t%s\t%s\n",
			idx+1,
			status.ID,
			status.Status,
			status.CurrentPhase,
			status.Iterations,
			started,
			finished,
			truncate(emptyOr(status.LastAction), 20),
			truncate(emptyOr(status.StopReason), 40),
			truncate(emptyOr(status.LastError), 40),
		)
	}
	_ = w.Flush()
}

func formatTime(value time.Time) string {
	if value.IsZero() {
		return "—"
	}
	return value.UTC().Format("2006-01-02 15:04")
}

func formatTimePtr(value *time.Time) string {
	if value == nil || value.IsZero() {
		return "—"
	}
	return value.UTC().Format("2006-01-02 15:04")
}

func truncate(value string, max int) string {
	if max <= 0 || len(value) <= max {
		return value
	}
	return value[:max-1] + "…"
}

func isInteractiveTerminal(file *os.File) bool {
	if file == nil {
		return false
	}
	info, err := file.Stat()
	if err != nil {
		return false
	}
	return (info.Mode() & os.ModeCharDevice) != 0
}

func promptRunSelection(statuses []RunStatus, input io.Reader, output io.Writer) (string, error) {
	if len(statuses) == 0 {
		return "", fmt.Errorf("no runs available")
	}
	fmt.Fprint(output, "Select run by number or id: ")
	reader := bufio.NewReader(input)
	line, err := reader.ReadString('\n')
	if err != nil && err != io.EOF {
		return "", fmt.Errorf("read selection: %w", err)
	}
	selection := strings.TrimSpace(line)
	if selection == "" {
		return "", fmt.Errorf("selection is empty")
	}
	for _, status := range statuses {
		if status.ID == selection {
			return selection, nil
		}
	}
	index, err := strconv.Atoi(selection)
	if err == nil && index > 0 && index <= len(statuses) {
		return statuses[index-1].ID, nil
	}
	return "", fmt.Errorf("invalid selection: %s", selection)
}
