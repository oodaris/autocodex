package terminal

import (
	"bytes"
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
	StatusRunning  = "running"
	StatusClosed   = "closed"
	defaultPTYCols = 120
	defaultPTYRows = 40
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

	outputMu sync.Mutex
	output   bytes.Buffer
	subsMu   sync.Mutex
	subs     map[chan []byte]struct{}
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

	ptyFile, err := pty.StartWithSize(cmd, &pty.Winsize{
		Cols: defaultPTYCols,
		Rows: defaultPTYRows,
	})
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
		subs:        map[chan []byte]struct{}{},
	}

	m.mu.Lock()
	m.sessions[id] = session
	m.mu.Unlock()

	session.startStream(m.logger)
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

func (s *Session) OutputSnapshot() []byte {
	s.outputMu.Lock()
	defer s.outputMu.Unlock()
	if s.output.Len() == 0 {
		return nil
	}
	data := s.output.Bytes()
	out := make([]byte, len(data))
	copy(out, data)
	return out
}

func (s *Session) Subscribe() chan []byte {
	ch := make(chan []byte, 16)
	s.subsMu.Lock()
	s.subs[ch] = struct{}{}
	s.subsMu.Unlock()
	return ch
}

func (s *Session) Unsubscribe(ch chan []byte) {
	if ch == nil {
		return
	}
	s.subsMu.Lock()
	if _, ok := s.subs[ch]; ok {
		delete(s.subs, ch)
		close(ch)
	}
	s.subsMu.Unlock()
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
	s.closeSubscribers()
	if logger != nil {
		logger.Info("terminal session closed", "session_id", s.ID, "exit_code", exitCode)
	}
}

func (s *Session) startStream(logger *slog.Logger) {
	if s.ptyFile == nil {
		return
	}
	go func() {
		buf := make([]byte, 4096)
		filter := &DSRFilter{}
		for {
			n, err := s.ptyFile.Read(buf)
			if err != nil {
				break
			}
			filtered := filter.Filter(s.ptyFile, buf[:n])
			if len(filtered) == 0 {
				continue
			}
			s.appendOutput(filtered)
			s.broadcast(filtered)
		}
		s.closeSubscribers()
		if logger != nil {
			logger.Debug("terminal stream closed", "session_id", s.ID)
		}
	}()
}

func (s *Session) appendOutput(chunk []byte) {
	const maxOutputBytes = 512 * 1024
	if len(chunk) == 0 {
		return
	}
	s.outputMu.Lock()
	defer s.outputMu.Unlock()

	if len(chunk) >= maxOutputBytes {
		s.output.Reset()
		s.output.Write(chunk[len(chunk)-maxOutputBytes:])
		return
	}
	if s.output.Len()+len(chunk) > maxOutputBytes {
		existing := s.output.Bytes()
		overflow := s.output.Len() + len(chunk) - maxOutputBytes
		if overflow < len(existing) {
			existing = existing[overflow:]
		} else {
			existing = nil
		}
		s.output.Reset()
		s.output.Write(existing)
	}
	s.output.Write(chunk)
}

func (s *Session) broadcast(chunk []byte) {
	s.subsMu.Lock()
	defer s.subsMu.Unlock()
	for ch := range s.subs {
		select {
		case ch <- chunk:
		default:
		}
	}
}

func (s *Session) closeSubscribers() {
	s.subsMu.Lock()
	for ch := range s.subs {
		close(ch)
		delete(s.subs, ch)
	}
	s.subsMu.Unlock()
}

func randomID() string {
	buf := make([]byte, 6)
	if _, err := rand.Read(buf); err != nil {
		return fmt.Sprintf("session-%d", time.Now().UnixNano())
	}
	return "session-" + hex.EncodeToString(buf)
}
