package plugins

import (
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"
)

func TestHostCallJSONRPC(t *testing.T) {
	if _, err := exec.LookPath("go"); err != nil {
		t.Skip("go toolchain not available")
	}

	tmp := t.TempDir()
	pluginDir := filepath.Join(tmp, "echo-plugin")
	if err := os.MkdirAll(pluginDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	if err := os.WriteFile(filepath.Join(pluginDir, "go.mod"), []byte("module echo-plugin\n\ngo 1.22\n"), 0o644); err != nil {
		t.Fatalf("write go.mod: %v", err)
	}

	source := `package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
)

type request struct {
	Capability string          ` + "`json:\"capability\"`" + `
	Input      json.RawMessage ` + "`json:\"input\"`" + `
}

type response struct {
	Output map[string]string ` + "`json:\"output\"`" + `
	Error  interface{}       ` + "`json:\"error\"`" + `
}

func main() {
	handshake := map[string]interface{}{
		"protocol_version": 1,
		"name": "echo-plugin",
		"transport": "jsonrpc",
		"address": "stdio",
	}
	b, _ := json.Marshal(handshake)
	fmt.Println(string(b))

	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		var req request
		if err := json.Unmarshal(scanner.Bytes(), &req); err != nil {
			continue
		}
		if req.Capability != "echo" {
			resp := response{Output: map[string]string{}, Error: "unknown"}
			b, _ := json.Marshal(resp)
			fmt.Println(string(b))
			return
		}
		resp := response{Output: map[string]string{"ok": "true"}, Error: nil}
		b, _ := json.Marshal(resp)
		fmt.Println(string(b))
		return
	}
}
`
	pluginPath := filepath.Join(pluginDir, "main.go")
	if err := os.WriteFile(pluginPath, []byte(source), 0o644); err != nil {
		t.Fatalf("write plugin: %v", err)
	}

	bin := filepath.Join(pluginDir, "echo-plugin")
	cmd := exec.Command("go", "build", "-o", bin, pluginPath)
	cmd.Dir = pluginDir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("build plugin: %v: %s", err, string(out))
	}

	manifest := `name: echo-plugin
version: 0.1.0
protocol_version: 1
entrypoint: ./echo-plugin
transport: jsonrpc
capabilities:
  - name: echo
`
	manifestPath := filepath.Join(pluginDir, "plugin.yaml")
	if err := os.WriteFile(manifestPath, []byte(manifest), 0o644); err != nil {
		t.Fatalf("write manifest: %v", err)
	}

	plugin, err := LoadManifest(manifestPath)
	if err != nil {
		t.Fatalf("load manifest: %v", err)
	}

	host := Host{Timeout: 5 * time.Second}
	out, err := host.Call(context.Background(), plugin, "echo", json.RawMessage(`{}`))
	if err != nil {
		t.Fatalf("call: %v", err)
	}
	if string(out) != `{"ok":"true"}` {
		t.Fatalf("unexpected output: %s", string(out))
	}
}
