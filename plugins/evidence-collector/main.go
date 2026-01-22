package main

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
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
	Root           string   `json:"root"`
	Globs          []string `json:"globs"`
	OutputDir      string   `json:"output_dir"`
	Command        string   `json:"command"`
	MaxOutputBytes int      `json:"max_output_bytes"`
}

type artifact struct {
	Source    string `json:"source"`
	Path      string `json:"path"`
	SizeBytes int64  `json:"size_bytes"`
}

type output struct {
	Artifacts []artifact `json:"artifacts"`
	Errors    []string   `json:"errors"`
}

func main() {
	handshake := map[string]interface{}{
		"protocol_version": 1,
		"name":             "evidence-collector",
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
		case "collect":
			out, err := collect(req.Input)
			if err != nil {
				writeResponse(response{Error: &pluginError{Message: err.Error(), Code: "collect_failed"}})
				continue
			}
			writeResponse(response{Output: out})
		default:
			writeResponse(response{Error: &pluginError{Message: "unknown capability", Code: "unknown_capability"}})
		}
	}
}

func collect(inputRaw json.RawMessage) (output, error) {
	in := input{Root: ".", MaxOutputBytes: 20000}
	_ = json.Unmarshal(inputRaw, &in)
	if strings.TrimSpace(in.OutputDir) == "" {
		return output{}, fmt.Errorf("output_dir is required")
	}
	root, err := filepath.Abs(in.Root)
	if err != nil {
		return output{}, err
	}
	outDir := in.OutputDir
	if !filepath.IsAbs(outDir) {
		outDir = filepath.Join(root, outDir)
	}
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return output{}, err
	}

	artifacts := []artifact{}
	errors := []string{}

	for _, pattern := range in.Globs {
		globPattern := pattern
		if !filepath.IsAbs(globPattern) {
			globPattern = filepath.Join(root, pattern)
		}
		matches, err := filepath.Glob(globPattern)
		if err != nil {
			errors = append(errors, err.Error())
			continue
		}
		for _, match := range matches {
			info, err := os.Stat(match)
			if err != nil || info.IsDir() {
				continue
			}
			rel, _ := filepath.Rel(root, match)
			dest := filepath.Join(outDir, rel)
			if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
				errors = append(errors, err.Error())
				continue
			}
			if err := copyFile(match, dest); err != nil {
				errors = append(errors, err.Error())
				continue
			}
			artifacts = append(artifacts, artifact{Source: rel, Path: dest, SizeBytes: info.Size()})
		}
	}

	if strings.TrimSpace(in.Command) != "" {
		cmdPath := filepath.Join(outDir, "command.txt")
		content, err := runCommand(root, in.Command, in.MaxOutputBytes)
		if err != nil {
			errors = append(errors, err.Error())
		}
		if writeErr := os.WriteFile(cmdPath, content, 0o644); writeErr != nil {
			errors = append(errors, writeErr.Error())
		} else {
			artifacts = append(artifacts, artifact{Source: "command", Path: cmdPath, SizeBytes: int64(len(content))})
		}
	}

	return output{Artifacts: artifacts, Errors: errors}, nil
}

func runCommand(root, command string, maxBytes int) ([]byte, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	cmd := exec.CommandContext(ctx, "bash", "-lc", command)
	cmd.Dir = root
	var buf bytes.Buffer
	writer := &limitedWriter{Writer: &buf, Limit: maxBytes}
	cmd.Stdout = writer
	cmd.Stderr = writer
	if err := cmd.Run(); err != nil {
		return buf.Bytes(), err
	}
	return buf.Bytes(), nil
}

type limitedWriter struct {
	Writer io.Writer
	Limit  int
	Used   int
}

func (l *limitedWriter) Write(p []byte) (int, error) {
	if l.Limit <= 0 {
		return l.Writer.Write(p)
	}
	remaining := l.Limit - l.Used
	if remaining <= 0 {
		return len(p), nil
	}
	if len(p) > remaining {
		n, _ := l.Writer.Write(p[:remaining])
		l.Used += n
		return len(p), nil
	}
	return l.Writer.Write(p)
}

func copyFile(src, dest string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.Create(dest)
	if err != nil {
		return err
	}
	defer out.Close()
	if _, err := io.Copy(out, in); err != nil {
		return err
	}
	return out.Close()
}

func writeResponse(resp response) {
	b, _ := json.Marshal(resp)
	fmt.Println(string(b))
}
