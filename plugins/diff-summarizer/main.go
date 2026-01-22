package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
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
	Root     string `json:"root"`
	Base     string `json:"base"`
	Head     string `json:"head"`
	MaxFiles int    `json:"max_files"`
}

type output struct {
	FilesChanged     int            `json:"files_changed"`
	FileStatusCounts map[string]int `json:"file_status_counts"`
	Areas            map[string]int `json:"areas"`
	RiskFlags        []string       `json:"risk_flags"`
	Files            []string       `json:"files"`
}

func main() {
	handshake := map[string]interface{}{
		"protocol_version": 1,
		"name":             "diff-summarizer",
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
		case "summarize":
			out, err := summarizeDiff(req.Input)
			if err != nil {
				writeResponse(response{Error: &pluginError{Message: err.Error(), Code: "summarize_failed"}})
				continue
			}
			writeResponse(response{Output: out})
		default:
			writeResponse(response{Error: &pluginError{Message: "unknown capability", Code: "unknown_capability"}})
		}
	}
}

func summarizeDiff(inputRaw json.RawMessage) (output, error) {
	in := input{Root: ".", MaxFiles: 2000}
	_ = json.Unmarshal(inputRaw, &in)
	if strings.TrimSpace(in.Root) == "" {
		in.Root = "."
	}
	root, err := filepath.Abs(in.Root)
	if err != nil {
		return output{}, err
	}
	args := []string{"-C", root, "diff", "--name-status"}
	if strings.TrimSpace(in.Base) != "" {
		rangeArg := in.Base
		if strings.TrimSpace(in.Head) != "" {
			rangeArg = fmt.Sprintf("%s..%s", in.Base, in.Head)
		}
		args = append(args, rangeArg)
	}
	cmd := exec.Command("git", args...)
	out, err := cmd.Output()
	if err != nil {
		return output{}, err
	}
	trimmed := strings.TrimSpace(string(out))
	lines := []string{}
	if trimmed != "" {
		lines = strings.Split(trimmed, "\n")
	}
	files := []string{}
	filesChanged := 0
	statusCounts := map[string]int{}
	areas := map[string]int{}
	riskFlags := map[string]bool{}
	for _, line := range lines {
		if strings.TrimSpace(line) == "" {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}
		status := fields[0]
		status = strings.ToUpper(status)
		if len(status) > 1 {
			status = string(status[0])
		}
		path := fields[len(fields)-1]
		if len(files) < in.MaxFiles {
			files = append(files, path)
		}
		statusCounts[status]++
		filesChanged++
		area := strings.Split(path, string(os.PathSeparator))[0]
		if area == "" || area == path {
			area = "(root)"
		}
		areas[area]++
		flagRisk(path, riskFlags)
	}
	riskList := []string{}
	for flag := range riskFlags {
		riskList = append(riskList, flag)
	}
	sort.Strings(riskList)
	return output{
		FilesChanged:     filesChanged,
		FileStatusCounts: statusCounts,
		Areas:            areas,
		RiskFlags:        riskList,
		Files:            files,
	}, nil
}

func flagRisk(path string, flags map[string]bool) {
	lower := strings.ToLower(path)
	switch {
	case strings.Contains(lower, "terraform") || strings.Contains(lower, "infra"):
		flags["infra_changes"] = true
	case strings.Contains(lower, ".github/workflows"):
		flags["ci_changes"] = true
	case strings.Contains(lower, "security") || strings.Contains(lower, "auth"):
		flags["auth_security_changes"] = true
	case strings.Contains(lower, "migrations") || strings.Contains(lower, "schema"):
		flags["data_contract_changes"] = true
	case strings.HasSuffix(lower, "go.mod") || strings.HasSuffix(lower, "go.sum"):
		flags["go_dependencies_changed"] = true
	case strings.HasSuffix(lower, "package.json") || strings.HasSuffix(lower, "package-lock.json"):
		flags["node_dependencies_changed"] = true
	}
}

func writeResponse(resp response) {
	b, _ := json.Marshal(resp)
	fmt.Println(string(b))
}
