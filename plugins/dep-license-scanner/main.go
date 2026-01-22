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
	Root          string   `json:"root"`
	DenyLicenses  []string `json:"deny_licenses"`
	AllowLicenses []string `json:"allow_licenses"`
	IncludeDev    bool     `json:"include_dev"`
}

type dep struct {
	Name    string `json:"name"`
	Version string `json:"version"`
	Source  string `json:"source"`
	License string `json:"license"`
}

type output struct {
	ProjectLicense string   `json:"project_license"`
	Dependencies   []dep    `json:"dependencies"`
	RiskFlags      []string `json:"risk_flags"`
}

func main() {
	handshake := map[string]interface{}{
		"protocol_version": 1,
		"name":             "dep-license-scanner",
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
		case "scan":
			out, err := scan(req.Input)
			if err != nil {
				writeResponse(response{Error: &pluginError{Message: err.Error(), Code: "scan_failed"}})
				continue
			}
			writeResponse(response{Output: out})
		default:
			writeResponse(response{Error: &pluginError{Message: "unknown capability", Code: "unknown_capability"}})
		}
	}
}

func scan(inputRaw json.RawMessage) (output, error) {
	in := input{Root: ".", DenyLicenses: []string{"unknown", "unlicensed"}}
	_ = json.Unmarshal(inputRaw, &in)
	if strings.TrimSpace(in.Root) == "" {
		in.Root = "."
	}
	root, err := filepath.Abs(in.Root)
	if err != nil {
		return output{}, err
	}
	projectLicense := detectProjectLicense(root)
	deps := []dep{}
	deps = append(deps, readNpmDeps(root, in.IncludeDev)...)
	deps = append(deps, readGoDeps(root)...)
	deps = append(deps, readRequirements(root)...)

	deny := toSet(lowerList(in.DenyLicenses))
	allow := toSet(lowerList(in.AllowLicenses))
	riskFlags := []string{}
	for _, d := range deps {
		license := strings.ToLower(strings.TrimSpace(d.License))
		if license == "" {
			license = "unknown"
		}
		if len(allow) > 0 && !allow[license] {
			riskFlags = append(riskFlags, fmt.Sprintf("license_not_allowlisted:%s:%s", d.Name, license))
			continue
		}
		if deny[license] {
			riskFlags = append(riskFlags, fmt.Sprintf("license_denied:%s:%s", d.Name, license))
		}
	}
	return output{
		ProjectLicense: projectLicense,
		Dependencies:   deps,
		RiskFlags:      uniqueStrings(riskFlags),
	}, nil
}

func readNpmDeps(root string, includeDev bool) []dep {
	path := filepath.Join(root, "package.json")
	b, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	var payload struct {
		Dependencies    map[string]string `json:"dependencies"`
		DevDependencies map[string]string `json:"devDependencies"`
		License         string            `json:"license"`
	}
	_ = json.Unmarshal(b, &payload)
	deps := []dep{}
	for name, version := range payload.Dependencies {
		deps = append(deps, dep{Name: name, Version: version, Source: "npm", License: "unknown"})
	}
	if includeDev {
		for name, version := range payload.DevDependencies {
			deps = append(deps, dep{Name: name, Version: version, Source: "npm-dev", License: "unknown"})
		}
	}
	return deps
}

func readGoDeps(root string) []dep {
	path := filepath.Join(root, "go.mod")
	b, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	lines := strings.Split(string(b), "\n")
	deps := []dep{}
	inRequire := false
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}
		if strings.HasPrefix(trimmed, "require (") {
			inRequire = true
			continue
		}
		if inRequire && strings.HasPrefix(trimmed, ")") {
			inRequire = false
			continue
		}
		hadRequire := strings.HasPrefix(trimmed, "require ")
		trimmed = strings.TrimPrefix(trimmed, "require ")
		if !inRequire && !hadRequire {
			continue
		}
		fields := strings.Fields(trimmed)
		if len(fields) < 2 {
			continue
		}
		deps = append(deps, dep{Name: fields[0], Version: fields[1], Source: "go", License: "unknown"})
	}
	return deps
}

func readRequirements(root string) []dep {
	path := filepath.Join(root, "requirements.txt")
	b, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	deps := []dep{}
	lines := strings.Split(string(b), "\n")
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}
		name := trimmed
		version := ""
		if strings.Contains(trimmed, "==") {
			parts := strings.SplitN(trimmed, "==", 2)
			name = strings.TrimSpace(parts[0])
			version = strings.TrimSpace(parts[1])
		}
		deps = append(deps, dep{Name: name, Version: version, Source: "pip", License: "unknown"})
	}
	return deps
}

func detectProjectLicense(root string) string {
	licenseFiles := []string{"LICENSE", "LICENSE.txt", "LICENSE.md"}
	for _, name := range licenseFiles {
		if fileExists(filepath.Join(root, name)) {
			return name
		}
	}
	pkgPath := filepath.Join(root, "package.json")
	if b, err := os.ReadFile(pkgPath); err == nil {
		var payload struct {
			License string `json:"license"`
		}
		if json.Unmarshal(b, &payload) == nil && payload.License != "" {
			return payload.License
		}
	}
	return "unknown"
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func lowerList(items []string) []string {
	out := make([]string, 0, len(items))
	for _, item := range items {
		out = append(out, strings.ToLower(strings.TrimSpace(item)))
	}
	return out
}

func toSet(items []string) map[string]bool {
	set := map[string]bool{}
	for _, item := range items {
		if item == "" {
			continue
		}
		set[item] = true
	}
	return set
}

func uniqueStrings(items []string) []string {
	seen := map[string]bool{}
	out := []string{}
	for _, item := range items {
		if seen[item] {
			continue
		}
		seen[item] = true
		out = append(out, item)
	}
	return out
}

func writeResponse(resp response) {
	b, _ := json.Marshal(resp)
	fmt.Println(string(b))
}
