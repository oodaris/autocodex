package state

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"time"
)

type RunEvent struct {
	ID      string            `json:"id"`
	RunID   string            `json:"run_id"`
	TS      time.Time         `json:"ts"`
	Type    string            `json:"type"`
	Phase   string            `json:"phase"`
	Message string            `json:"message"`
	Meta    map[string]string `json:"meta"`
}

func (s *Store) AppendEvent(event RunEvent) error {
	path := s.eventsPath(event.RunID)
	file, err := os.OpenFile(path, os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return fmt.Errorf("open events log: %w", err)
	}
	defer file.Close()

	data, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("marshal event: %w", err)
	}
	if _, err := file.Write(append(data, '\n')); err != nil {
		return fmt.Errorf("write event: %w", err)
	}
	return nil
}

func (s *Store) ListEvents(runID string) ([]RunEvent, error) {
	path := s.eventsPath(runID)
	file, err := os.Open(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return []RunEvent{}, nil
		}
		return nil, fmt.Errorf("open events: %w", err)
	}
	defer file.Close()

	return decodeEvents(file)
}

func decodeEvents(r io.Reader) ([]RunEvent, error) {
	scanner := bufio.NewScanner(r)
	var events []RunEvent
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		var event RunEvent
		if err := json.Unmarshal([]byte(line), &event); err != nil {
			continue
		}
		events = append(events, event)
	}
	return events, scanner.Err()
}
