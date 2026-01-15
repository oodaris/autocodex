package terminal

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/creack/pty"
)

const (
	StatusRunning = "running"
	StatusClosed  = "closed"
)

type SessionConfig struct {
	Command     string
	Args        []string
	Cwd         string
	Env         []string
	WorkspaceID string
}

type SessionSummary struct {
	ID          string    `json:"id"`
	Status      string    `json:"status"`
	Command     string    `json:"command"`
	Args        []string  `json:"args"`
	Cwd         string    `json:"cwd"`
	WorkspaceID string    `json:"workspace_id,omitempty"`
	PID         int       `json:"pid"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
	ExitCode    *int      `json:"exit_code,omitempty"`
}

type Session struct {
	ID          string
	Command     string
	Args        []string
	Cwd         string
	WorkspaceID string
	PID         int
	CreatedAt   time.Time
	UpdatedAt   time.Time
	ExitCode    *int
	Status      string

	cmd      *exec.Cmd
	ptyFile  *os.File
	attached bool
	mu       sync.Mutex
}

type Manager struct {
	sessions map[string]*Session
	mu       sync.RWMutex
	logger   *slog.Logger
}

func NewManager(logger *slog.Logger) *Manager {
	return &Manager{sessions: map[string]*Session{}, logger: logger}
}

func (m *Manager) Create(ctx context.Context, cfg SessionConfig) (*Session, error) {
	command := strings.TrimSpace(cfg.Command)
	if command == "" {
		return nil, errors.New("command is required")
	}
	cmd := exec.CommandContext(ctx, command, cfg.Args...)
	if cfg.Cwd != "" {
		cmd.Dir = cfg.Cwd
	}
	if len(cfg.Env) > 0 {
		cmd.Env = append(os.Environ(), cfg.Env...)
	}

	ptyFile, err := pty.Start(cmd)
	if err != nil {
		return nil, fmt.Errorf("start pty: %w", err)
	}

	id := randomID()
	createdAt := time.Now().UTC()
	session := &Session{
		ID:          id,
		Command:     command,
		Args:        append([]string{}, cfg.Args...),
		Cwd:         cfg.Cwd,
		WorkspaceID: cfg.WorkspaceID,
		PID:         cmd.Process.Pid,
		CreatedAt:   createdAt,
		UpdatedAt:   createdAt,
		Status:      StatusRunning,
		cmd:         cmd,
		ptyFile:     ptyFile,
	}

	m.mu.Lock()
	m.sessions[id] = session
	m.mu.Unlock()

	go session.wait(m.logger)
	return session, nil
}

func (m *Manager) List() []SessionSummary {
	m.mu.RLock()
	sessions := make([]*Session, 0, len(m.sessions))
	for _, session := range m.sessions {
		sessions = append(sessions, session)
	}
	m.mu.RUnlock()

	summaries := make([]SessionSummary, 0, len(sessions))
	for _, session := range sessions {
		summaries = append(summaries, session.Summary())
	}
	sort.Slice(summaries, func(i, j int) bool {
		return summaries[i].CreatedAt.After(summaries[j].CreatedAt)
	})
	return summaries
}

func (m *Manager) Get(id string) (*Session, bool) {
	m.mu.RLock()
	session, ok := m.sessions[id]
	m.mu.RUnlock()
	return session, ok
}

func (m *Manager) Close(id string) error {
	session, ok := m.Get(id)
	if !ok {
		return fmt.Errorf("session not found: %s", id)
	}
	return session.Close()
}

func (s *Session) Summary() SessionSummary {
	s.mu.Lock()
	defer s.mu.Unlock()

	summary := SessionSummary{
		ID:          s.ID,
		Status:      s.Status,
		Command:     s.Command,
		Args:        append([]string{}, s.Args...),
		Cwd:         s.Cwd,
		WorkspaceID: s.WorkspaceID,
		PID:         s.PID,
		CreatedAt:   s.CreatedAt,
		UpdatedAt:   s.UpdatedAt,
		ExitCode:    s.ExitCode,
	}
	return summary
}

func (s *Session) PTY() *os.File {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.ptyFile
}

func (s *Session) TryAttach() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.attached || s.Status != StatusRunning {
		return false
	}
	s.attached = true
	return true
}

func (s *Session) Detach() {
	s.mu.Lock()
	s.attached = false
	s.mu.Unlock()
}

func (s *Session) Close() error {
	s.mu.Lock()
	if s.Status == StatusClosed {
		s.mu.Unlock()
		return nil
	}
	cmd := s.cmd
	s.mu.Unlock()
	if cmd == nil || cmd.Process == nil {
		return nil
	}
	return cmd.Process.Kill()
}

func (s *Session) wait(logger *slog.Logger) {
	err := s.cmd.Wait()

	s.mu.Lock()
	defer s.mu.Unlock()
	s.Status = StatusClosed
	s.UpdatedAt = time.Now().UTC()
	exitCode := 0
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		} else {
			exitCode = 1
		}
	}
	s.ExitCode = &exitCode
	if s.ptyFile != nil {
		_ = s.ptyFile.Close()
	}
	if logger != nil {
		logger.Info("terminal session closed", "session_id", s.ID, "exit_code", exitCode)
	}
}

func randomID() string {
	buf := make([]byte, 6)
	if _, err := rand.Read(buf); err != nil {
		return fmt.Sprintf("session-%d", time.Now().UnixNano())
	}
	return "session-" + hex.EncodeToString(buf)
}
