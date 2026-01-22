package api

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/oodaris/autocodex/internal/config"
	"github.com/oodaris/autocodex/internal/terminal"
)

// TerminalSessionCreateRequest creates a new terminal session.
type TerminalSessionCreateRequest struct {
	WorkspaceID string   `json:"workspace_id,omitempty"`
	Command     string   `json:"command,omitempty"`
	Args        []string `json:"args,omitempty"`
	Env         []string `json:"env,omitempty"`
}

func (s *Server) handleTerminalSessions(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	if s.Terminal == nil {
		respondError(w, http.StatusNotFound, "terminal not enabled")
		s.log(r, http.StatusNotFound, start, nil)
		return
	}

	switch r.Method {
	case http.MethodGet:
		if err := respondJSON(w, http.StatusOK, s.Terminal.List()); err != nil {
			s.log(r, http.StatusInternalServerError, start, err)
			return
		}
		s.log(r, http.StatusOK, start, nil)
	case http.MethodPost:
		var req TerminalSessionCreateRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			respondError(w, http.StatusBadRequest, "invalid json")
			s.log(r, http.StatusBadRequest, start, err)
			return
		}
		workspaceID := strings.TrimSpace(req.WorkspaceID)
		if workspaceID != "" {
			r.Header.Set("X-Workspace-Id", workspaceID)
		}
		cfg, root, err := s.resolveTerminalConfig(workspaceID)
		if err != nil {
			respondError(w, http.StatusBadRequest, err.Error())
			s.log(r, http.StatusBadRequest, start, err)
			return
		}
		command := strings.TrimSpace(req.Command)
		if command == "" {
			command = cfg.Codex.CLIPath
		}
		if command == "" {
			command = defaultShellCommand()
		}
		args := req.Args
		if len(args) == 0 && len(cfg.Codex.ExtraArgs) > 0 {
			args = append([]string{}, cfg.Codex.ExtraArgs...)
		}
		env := appendEnv(cfg.Codex.Env, req.Env)
		if isCodexCommand(command, cfg.Codex.CLIPath) {
			env = ensureEnv(env, map[string]string{
				"CI":   "1",
				"TERM": "dumb",
			})
		}
		if workspaceID != "" {
			env = append(env, "AUTOCODEX_WORKSPACE_ID="+workspaceID)
		}

		session, err := s.Terminal.Create(context.Background(), terminal.SessionConfig{
			Command:     command,
			Args:        args,
			Cwd:         root,
			Env:         env,
			WorkspaceID: workspaceID,
		})
		if err != nil {
			respondError(w, http.StatusInternalServerError, "internal error")
			s.log(r, http.StatusInternalServerError, start, err)
			return
		}
		if err := respondJSON(w, http.StatusCreated, session.Summary()); err != nil {
			s.log(r, http.StatusInternalServerError, start, err)
			return
		}
		s.log(r, http.StatusCreated, start, nil)
	default:
		respondError(w, http.StatusMethodNotAllowed, "method not allowed")
		s.log(r, http.StatusMethodNotAllowed, start, nil)
	}
}

func (s *Server) handleTerminalSession(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	if s.Terminal == nil {
		respondError(w, http.StatusNotFound, "terminal not enabled")
		s.log(r, http.StatusNotFound, start, nil)
		return
	}
	path := strings.TrimPrefix(r.URL.Path, "/terminal/sessions/")
	if path == "" {
		respondError(w, http.StatusNotFound, "session not found")
		s.log(r, http.StatusNotFound, start, nil)
		return
	}
	parts := strings.SplitN(path, "/", 2)
	sessionID := parts[0]
	if sessionID == "" {
		respondError(w, http.StatusNotFound, "session not found")
		s.log(r, http.StatusNotFound, start, nil)
		return
	}

	if len(parts) == 2 && parts[1] == "ws" {
		s.handleTerminalSessionWS(w, r, sessionID)
		return
	}

	session, ok := s.Terminal.Get(sessionID)
	if !ok {
		respondError(w, http.StatusNotFound, "session not found")
		s.log(r, http.StatusNotFound, start, nil)
		return
	}
	if session.WorkspaceID != "" {
		r.Header.Set("X-Workspace-Id", session.WorkspaceID)
	}

	switch r.Method {
	case http.MethodGet:
		if err := respondJSON(w, http.StatusOK, session.Summary()); err != nil {
			s.log(r, http.StatusInternalServerError, start, err)
			return
		}
		s.log(r, http.StatusOK, start, nil)
	case http.MethodDelete:
		if err := s.Terminal.Close(sessionID); err != nil {
			respondError(w, http.StatusInternalServerError, "internal error")
			s.log(r, http.StatusInternalServerError, start, err)
			return
		}
		if err := respondJSON(w, http.StatusOK, session.Summary()); err != nil {
			s.log(r, http.StatusInternalServerError, start, err)
			return
		}
		s.log(r, http.StatusOK, start, nil)
	default:
		respondError(w, http.StatusMethodNotAllowed, "method not allowed")
		s.log(r, http.StatusMethodNotAllowed, start, nil)
	}
}

func (s *Server) handleTerminalSessionWS(w http.ResponseWriter, r *http.Request, sessionID string) {
	start := time.Now()
	session, ok := s.Terminal.Get(sessionID)
	if !ok {
		respondError(w, http.StatusNotFound, "session not found")
		s.log(r, http.StatusNotFound, start, nil)
		return
	}
	if session.WorkspaceID != "" {
		r.Header.Set("X-Workspace-Id", session.WorkspaceID)
	}
	if !session.TryAttach() {
		respondError(w, http.StatusConflict, "session already attached")
		s.log(r, http.StatusConflict, start, nil)
		return
	}
	defer session.Detach()

	upgrader := websocket.Upgrader{CheckOrigin: s.allowOrigin}
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		s.log(r, http.StatusBadRequest, start, err)
		return
	}
	defer conn.Close()

	ptyFile := session.PTY()
	if ptyFile == nil {
		respondError(w, http.StatusGone, "session closed")
		s.log(r, http.StatusGone, start, nil)
		return
	}

	readDone := make(chan struct{})
	output := session.OutputSnapshot()
	if len(output) > 0 {
		_ = conn.WriteMessage(websocket.BinaryMessage, output)
	}
	sub := session.Subscribe()
	var cleanupOnce sync.Once
	cleanup := func() { cleanupOnce.Do(func() { session.Unsubscribe(sub) }) }
	go func() {
		for chunk := range sub {
			if err := conn.WriteMessage(websocket.BinaryMessage, chunk); err != nil {
				cleanup()
				break
			}
		}
		close(readDone)
	}()

	for {
		_, msg, err := conn.ReadMessage()
		if err != nil {
			break
		}
		if len(msg) == 0 {
			continue
		}
		_, _ = ptyFile.Write(msg)
	}

	cleanup()
	<-readDone
	s.log(r, http.StatusOK, start, nil)
}

func (s *Server) resolveTerminalConfig(workspaceID string) (config.Config, string, error) {
	if workspaceID == "" {
		root := s.RootDir
		if root == "" {
			root, _ = filepath.Abs(".")
		}
		return s.Config, root, nil
	}
	if s.Hub == nil {
		return config.Config{}, "", errors.New("hub not enabled")
	}
	ws, err := s.Hub.Workspace(workspaceID)
	if err != nil {
		return config.Config{}, "", err
	}
	if ws.Err != nil {
		return config.Config{}, "", ws.Err
	}
	if ws.Config == nil {
		return config.Config{}, "", errors.New("workspace config missing")
	}
	return *ws.Config, ws.Root, nil
}

func appendEnv(env map[string]string, extra []string) []string {
	result := make([]string, 0, len(env)+len(extra))
	for key, value := range env {
		if strings.TrimSpace(key) == "" {
			continue
		}
		result = append(result, key+"="+value)
	}
	for _, entry := range extra {
		value := strings.TrimSpace(entry)
		if value == "" {
			continue
		}
		result = append(result, value)
	}
	return result
}

func ensureEnv(env []string, overrides map[string]string) []string {
	if len(overrides) == 0 {
		return env
	}
	filtered := env[:0]
	for _, entry := range env {
		key := strings.SplitN(entry, "=", 2)[0]
		if _, ok := overrides[key]; ok {
			continue
		}
		filtered = append(filtered, entry)
	}
	for key, value := range overrides {
		filtered = append(filtered, key+"="+value)
	}
	return filtered
}

func isCodexCommand(command, cfgPath string) bool {
	command = strings.TrimSpace(command)
	if command == "" {
		return false
	}
	base := filepath.Base(command)
	if base == "codex" {
		return true
	}
	if cfgPath == "" {
		return false
	}
	return base == filepath.Base(cfgPath)
}

func defaultShellCommand() string {
	if runtime.GOOS == "windows" {
		if value := strings.TrimSpace(os.Getenv("COMSPEC")); value != "" {
			return value
		}
		return "cmd.exe"
	}
	if value := strings.TrimSpace(os.Getenv("SHELL")); value != "" {
		return value
	}
	return "bash"
}

func (s *Server) allowOrigin(r *http.Request) bool {
	if s.Config.UI.Origin == "" {
		return true
	}
	origin := r.Header.Get("Origin")
	if origin == "" {
		return true
	}
	return origin == s.Config.UI.Origin
}
