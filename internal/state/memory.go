package state

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

type MemoryDocSummary struct {
	Name      string    `json:"name"`
	Path      string    `json:"path"`
	UpdatedAt time.Time `json:"updated_at"`
	SizeBytes int64     `json:"size_bytes"`
}

type MemoryDocDetail struct {
	Name      string    `json:"name"`
	Path      string    `json:"path"`
	UpdatedAt time.Time `json:"updated_at"`
	SizeBytes int64     `json:"size_bytes"`
	Content   string    `json:"content"`
}

func (s *Store) EnsureMemoryDocs() error {
	docs := []string{"TODO.md", "PROGRESS.md", "OPINIONS.md", "SPEC.md"}
	for _, name := range docs {
		path := filepath.Join(s.MemoryDir, name)
		if _, err := os.Stat(path); err == nil {
			continue
		}
		if err := os.WriteFile(path, []byte("# "+trimExt(name)+"\n"), 0o644); err != nil {
			return fmt.Errorf("write memory doc %s: %w", path, err)
		}
	}
	return nil
}

func (s *Store) ListMemoryDocs() ([]MemoryDocSummary, error) {
	entries, err := os.ReadDir(s.MemoryDir)
	if err != nil {
		return nil, fmt.Errorf("read memory dir: %w", err)
	}
	docs := make([]MemoryDocSummary, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if filepath.Ext(name) != ".md" {
			continue
		}
		info, err := entry.Info()
		if err != nil {
			return nil, fmt.Errorf("stat memory doc %s: %w", name, err)
		}
		docs = append(docs, MemoryDocSummary{
			Name:      name,
			Path:      name,
			UpdatedAt: info.ModTime().UTC(),
			SizeBytes: info.Size(),
		})
	}
	sort.Slice(docs, func(i, j int) bool {
		return docs[i].Name < docs[j].Name
	})
	return docs, nil
}

func (s *Store) GetMemoryDoc(name string) (*MemoryDocDetail, error) {
	if name == "" {
		return nil, errors.New("memory doc name required")
	}
	if filepath.Base(name) != name || strings.Contains(name, string(filepath.Separator)) {
		return nil, fmt.Errorf("invalid memory doc name: %s", name)
	}
	path := filepath.Join(s.MemoryDir, name)
	info, err := os.Stat(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, err
		}
		return nil, fmt.Errorf("stat memory doc: %w", err)
	}
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read memory doc: %w", err)
	}
	return &MemoryDocDetail{
		Name:      name,
		Path:      name,
		UpdatedAt: info.ModTime().UTC(),
		SizeBytes: info.Size(),
		Content:   string(content),
	}, nil
}

func (s *Store) AppendMemoryDoc(name, content string) error {
	if strings.TrimSpace(content) == "" {
		return nil
	}
	if name == "" {
		return errors.New("memory doc name required")
	}
	if filepath.Base(name) != name || strings.Contains(name, string(filepath.Separator)) {
		return fmt.Errorf("invalid memory doc name: %s", name)
	}
	if err := os.MkdirAll(s.MemoryDir, 0o755); err != nil {
		return fmt.Errorf("init memory dir: %w", err)
	}
	path := filepath.Join(s.MemoryDir, name)
	payload := strings.TrimRight(content, "\n")
	payload = "\n" + payload + "\n"
	file, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return fmt.Errorf("open memory doc: %w", err)
	}
	defer file.Close()
	if _, err := file.WriteString(payload); err != nil {
		return fmt.Errorf("append memory doc: %w", err)
	}
	return nil
}

func trimExt(name string) string {
	ext := filepath.Ext(name)
	return name[:len(name)-len(ext)]
}
