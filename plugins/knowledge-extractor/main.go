package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type request struct {
	Capability string          `json:"capability"`
	Input      json.RawMessage `json:"input"`
}

type response struct {
	Output interface{}  `json:"output,omitempty"`
	Error  *pluginError `json:"error,omitempty"`
}

type pluginError struct {
	Message string `json:"message"`
	Code    string `json:"code,omitempty"`
}

type input struct {
	Root            string   `json:"root"`
	Paths           []string `json:"paths"`
	MaxFiles        int      `json:"max_files"`
	MaxBytesPerFile int      `json:"max_bytes_per_file"`
}

type document struct {
	Path      string   `json:"path"`
	Title     string   `json:"title"`
	Headings  []string `json:"headings"`
	SizeBytes int64    `json:"size_bytes"`
	Snippet   string   `json:"snippet"`
}

type output struct {
	Documents []document `json:"documents"`
}

func main() {
	handshake := map[string]interface{}{
		"protocol_version": 1,
		"name":             "knowledge-extractor",
		"transport":        "jsonrpc",
		"address":          "stdio",
	}
	b, _ := json.Marshal(handshake)
	fmt.Println(string(b))

	scanner := bufio.NewScanner(os.Stdin)
	scanner.Buffer(make([]byte, 0, 64*1024), 4*1024*1024)
	for scanner.Scan() {
		var req request
		if err := json.Unmarshal(scanner.Bytes(), &req); err != nil {
			writeResponse(response{Error: &pluginError{Message: "invalid request", Code: "bad_request"}})
			continue
		}
		switch req.Capability {
		case "extract":
			out, err := extract(req.Input)
			if err != nil {
				writeResponse(response{Error: &pluginError{Message: err.Error(), Code: "extract_failed"}})
				continue
			}
			writeResponse(response{Output: out})
		default:
			writeResponse(response{Error: &pluginError{Message: "unknown capability", Code: "unknown_capability"}})
		}
	}
}

func extract(inputRaw json.RawMessage) (output, error) {
	in := input{Root: ".", MaxFiles: 200, MaxBytesPerFile: 20000}
	_ = json.Unmarshal(inputRaw, &in)
	if strings.TrimSpace(in.Root) == "" {
		in.Root = "."
	}
	root, err := filepath.Abs(in.Root)
	if err != nil {
		return output{}, err
	}
	paths := in.Paths
	if len(paths) == 0 {
		paths = []string{"docs"}
	}

	documents := []document{}
	for _, p := range paths {
		base := filepath.Join(root, p)
		_ = filepath.WalkDir(base, func(path string, d os.DirEntry, err error) error {
			if err != nil {
				return nil
			}
			if d.IsDir() {
				return nil
			}
			if len(documents) >= in.MaxFiles {
				return filepath.SkipDir
			}
			if !strings.HasSuffix(strings.ToLower(d.Name()), ".md") {
				return nil
			}
			doc, err := summarizeDoc(root, path, in.MaxBytesPerFile)
			if err != nil {
				return nil
			}
			documents = append(documents, doc)
			return nil
		})
	}
	return output{Documents: documents}, nil
}

func summarizeDoc(root, path string, maxBytes int) (document, error) {
	info, err := os.Stat(path)
	if err != nil {
		return document{}, err
	}
	content, err := os.ReadFile(path)
	if err != nil {
		return document{}, err
	}
	if maxBytes > 0 && len(content) > maxBytes {
		content = content[:maxBytes]
	}
	text := string(content)
	lines := strings.Split(text, "\n")
	title := ""
	headings := []string{}
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "# ") && title == "" {
			title = strings.TrimSpace(strings.TrimPrefix(trimmed, "# "))
			continue
		}
		if strings.HasPrefix(trimmed, "## ") {
			headings = append(headings, strings.TrimSpace(strings.TrimPrefix(trimmed, "## ")))
		}
	}
	if title == "" {
		title = filepath.Base(path)
	}
	snippet := strings.TrimSpace(text)
	if len(snippet) > 200 {
		snippet = snippet[:200] + "..."
	}
	rel, _ := filepath.Rel(root, path)
	return document{
		Path:      rel,
		Title:     title,
		Headings:  headings,
		SizeBytes: info.Size(),
		Snippet:   snippet,
	}, nil
}

func writeResponse(resp response) {
	b, _ := json.Marshal(resp)
	fmt.Println(string(b))
}
