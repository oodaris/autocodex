package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
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
	Root          string `json:"root"`
	MaxFiles      int    `json:"max_files"`
	IncludeHidden bool   `json:"include_hidden"`
}

type langCount struct {
	Name  string `json:"name"`
	Files int    `json:"files"`
}

type service struct {
	Name string `json:"name"`
	Path string `json:"path"`
}

type output struct {
	Root         string      `json:"root"`
	Languages    []langCount `json:"languages"`
	TopDirs      []string    `json:"top_dirs"`
	TestCommands []string    `json:"test_commands"`
	Services     []service   `json:"services"`
	Notes        []string    `json:"notes,omitempty"`
}

func main() {
	handshake := map[string]interface{}{
		"protocol_version": 1,
		"name":             "repo-indexer",
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
		case "index":
			out, err := indexRepo(req.Input)
			if err != nil {
				writeResponse(response{Error: &pluginError{Message: err.Error(), Code: "index_failed"}})
				continue
			}
			writeResponse(response{Output: out})
		default:
			writeResponse(response{Error: &pluginError{Message: "unknown capability", Code: "unknown_capability"}})
		}
	}
}

func indexRepo(inputRaw json.RawMessage) (output, error) {
	in := input{Root: ".", MaxFiles: 200000}
	_ = json.Unmarshal(inputRaw, &in)
	if strings.TrimSpace(in.Root) == "" {
		in.Root = "."
	}
	root, err := filepath.Abs(in.Root)
	if err != nil {
		return output{}, err
	}

	topDirs := listTopDirs(root, in.IncludeHidden)
	langCounts := map[string]int{}
	filesScanned := 0
	ignoreDirs := map[string]bool{
		".git": true, "node_modules": true, "vendor": true, ".autocodex": true,
		"dist": true, "build": true, ".idea": true, ".vscode": true,
	}

	_ = filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() {
			name := d.Name()
			if !in.IncludeHidden && strings.HasPrefix(name, ".") && path != root {
				return filepath.SkipDir
			}
			if ignoreDirs[name] {
				return filepath.SkipDir
			}
			return nil
		}
		filesScanned++
		if filesScanned > in.MaxFiles {
			return filepath.SkipDir
		}
		ext := strings.ToLower(filepath.Ext(d.Name()))
		lang := extToLang(ext)
		if lang != "" {
			langCounts[lang]++
		}
		return nil
	})

	languages := make([]langCount, 0, len(langCounts))
	for name, count := range langCounts {
		languages = append(languages, langCount{Name: name, Files: count})
	}
	sort.Slice(languages, func(i, j int) bool { return languages[i].Files > languages[j].Files })

	tests := detectTestCommands(root)
	services := detectServices(root)
	return output{
		Root:         root,
		Languages:    languages,
		TopDirs:      topDirs,
		TestCommands: tests,
		Services:     services,
	}, nil
}

func listTopDirs(root string, includeHidden bool) []string {
	entries, err := os.ReadDir(root)
	if err != nil {
		return []string{}
	}
	out := []string{}
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		name := entry.Name()
		if !includeHidden && strings.HasPrefix(name, ".") {
			continue
		}
		out = append(out, name)
	}
	sort.Strings(out)
	return out
}

func extToLang(ext string) string {
	switch ext {
	case ".go":
		return "go"
	case ".ts", ".tsx":
		return "typescript"
	case ".js", ".jsx":
		return "javascript"
	case ".py":
		return "python"
	case ".rs":
		return "rust"
	case ".java":
		return "java"
	case ".kt":
		return "kotlin"
	case ".cs":
		return "csharp"
	case ".cpp", ".cc", ".cxx":
		return "cpp"
	case ".c":
		return "c"
	case ".rb":
		return "ruby"
	case ".php":
		return "php"
	case ".md":
		return "markdown"
	default:
		return ""
	}
}

func detectTestCommands(root string) []string {
	out := []string{}
	if fileExists(filepath.Join(root, "go.mod")) {
		out = append(out, "go test ./...")
	}
	if fileExists(filepath.Join(root, "package.json")) {
		out = append(out, "npm test")
	}
	if fileExists(filepath.Join(root, "pyproject.toml")) || fileExists(filepath.Join(root, "requirements.txt")) {
		out = append(out, "pytest -q")
	}
	if fileExists(filepath.Join(root, "Makefile")) {
		out = append(out, "make test")
	}
	return out
}

func detectServices(root string) []service {
	services := []service{}
	matches, _ := filepath.Glob(filepath.Join(root, "cmd", "*", "main.go"))
	for _, match := range matches {
		name := filepath.Base(filepath.Dir(match))
		rel, _ := filepath.Rel(root, filepath.Dir(match))
		services = append(services, service{Name: name, Path: rel})
	}
	sort.Slice(services, func(i, j int) bool { return services[i].Name < services[j].Name })
	return services
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func writeResponse(resp response) {
	b, _ := json.Marshal(resp)
	fmt.Println(string(b))
}
