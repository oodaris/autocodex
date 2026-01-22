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
	PlanPath         string   `json:"plan_path"`
	RequiredSections []string `json:"required_sections"`
}

type output struct {
	Status          string   `json:"status"`
	MissingSections []string `json:"missing_sections"`
	OpenTasks       []string `json:"open_tasks"`
}

func main() {
	handshake := map[string]interface{}{
		"protocol_version": 1,
		"name":             "plan-compliance",
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
		case "check":
			out, err := checkPlan(req.Input)
			if err != nil {
				writeResponse(response{Error: &pluginError{Message: err.Error(), Code: "check_failed"}})
				continue
			}
			writeResponse(response{Output: out})
		default:
			writeResponse(response{Error: &pluginError{Message: "unknown capability", Code: "unknown_capability"}})
		}
	}
}

func checkPlan(inputRaw json.RawMessage) (output, error) {
	in := input{}
	if err := json.Unmarshal(inputRaw, &in); err != nil {
		return output{}, err
	}
	if strings.TrimSpace(in.PlanPath) == "" {
		return output{}, fmt.Errorf("plan_path is required")
	}
	path := in.PlanPath
	if !filepath.IsAbs(path) {
		cwd, _ := os.Getwd()
		path = filepath.Join(cwd, path)
	}
	content, err := os.ReadFile(path)
	if err != nil {
		return output{}, err
	}
	lines := strings.Split(string(content), "\n")
	sections := map[string]bool{}
	openTasks := []string{}
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "# ") || strings.HasPrefix(trimmed, "## ") {
			section := strings.TrimPrefix(trimmed, "# ")
			section = strings.TrimPrefix(section, "# ")
			section = strings.ToLower(strings.TrimSpace(section))
			if section != "" {
				sections[section] = true
			}
		}
		if strings.HasPrefix(trimmed, "- [ ]") {
			openTasks = append(openTasks, strings.TrimSpace(strings.TrimPrefix(trimmed, "- [ ]")))
		}
	}
	missing := []string{}
	for _, req := range in.RequiredSections {
		key := strings.ToLower(strings.TrimSpace(req))
		if key == "" {
			continue
		}
		if !sections[key] {
			missing = append(missing, req)
		}
	}
	status := "pass"
	if len(missing) > 0 || len(openTasks) > 0 {
		status = "fail"
	}
	return output{Status: status, MissingSections: missing, OpenTasks: openTasks}, nil
}

func writeResponse(resp response) {
	b, _ := json.Marshal(resp)
	fmt.Println(string(b))
}
