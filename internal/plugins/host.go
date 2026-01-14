package plugins

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"time"
)

type Host struct {
	Timeout time.Duration
}

type Handshake struct {
	ProtocolVersion int    `json:"protocol_version"`
	Name            string `json:"name"`
	Transport       string `json:"transport"`
	Address         string `json:"address"`
}

type Request struct {
	Capability string          `json:"capability"`
	Input      json.RawMessage `json:"input"`
}

type Response struct {
	Output json.RawMessage `json:"output"`
	Error  *RPCError       `json:"error"`
}

type RPCError struct {
	Message string `json:"message"`
	Code    string `json:"code,omitempty"`
}

func (h Host) Call(ctx context.Context, plugin Plugin, capability string, input json.RawMessage) (json.RawMessage, error) {
	if capability == "" {
		return nil, errors.New("capability is required")
	}
	if !hasCapability(plugin.Manifest, capability) {
		return nil, fmt.Errorf("capability not declared: %s", capability)
	}

	timeout := h.Timeout
	if timeout == 0 {
		timeout = 60 * time.Second
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	cmd, stdin, stdout, stderr, err := startProcess(ctx, plugin)
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = cmd.Process.Kill()
		_ = cmd.Wait()
	}()

	handshake, err := readHandshake(ctx, stdout)
	if err != nil {
		return nil, errWithStderr("handshake failed", err, stderr)
	}
	if handshake.ProtocolVersion != plugin.Manifest.ProtocolVersion {
		return nil, fmt.Errorf("protocol version mismatch: %d", handshake.ProtocolVersion)
	}
	if handshake.Name != plugin.Manifest.Name {
		return nil, fmt.Errorf("handshake name mismatch: %s", handshake.Name)
	}
	if handshake.Transport != plugin.Manifest.Transport {
		return nil, fmt.Errorf("handshake transport mismatch: %s", handshake.Transport)
	}

	switch handshake.Transport {
	case "jsonrpc":
		return callJSONRPC(ctx, stdin, stdout, capability, input)
	case "grpc":
		return nil, errors.New("grpc transport not implemented")
	default:
		return nil, fmt.Errorf("unsupported transport: %s", handshake.Transport)
	}
}

func startProcess(ctx context.Context, plugin Plugin) (*exec.Cmd, io.WriteCloser, *bufio.Reader, *bytes.Buffer, error) {
	entry := plugin.Manifest.Entrypoint
	if !filepath.IsAbs(entry) {
		entry = filepath.Join(plugin.Dir, entry)
	}

	cmd := exec.CommandContext(ctx, entry)
	cmd.Dir = plugin.Dir

	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, nil, nil, nil, fmt.Errorf("stdin pipe: %w", err)
	}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, nil, nil, nil, fmt.Errorf("stdout pipe: %w", err)
	}

	stderr := &bytes.Buffer{}
	cmd.Stderr = io.MultiWriter(os.Stderr, stderr)

	if err := cmd.Start(); err != nil {
		return nil, nil, nil, nil, fmt.Errorf("start plugin: %w", err)
	}

	return cmd, stdin, bufio.NewReader(stdout), stderr, nil
}

func readHandshake(ctx context.Context, reader *bufio.Reader) (Handshake, error) {
	line, err := readLine(ctx, reader)
	if err != nil {
		return Handshake{}, err
	}
	var h Handshake
	if err := json.Unmarshal(line, &h); err != nil {
		return Handshake{}, fmt.Errorf("invalid handshake: %w", err)
	}
	return h, nil
}

func callJSONRPC(ctx context.Context, stdin io.Writer, reader *bufio.Reader, capability string, input json.RawMessage) (json.RawMessage, error) {
	if len(input) == 0 {
		input = json.RawMessage([]byte("{}"))
	}
	req := Request{Capability: capability, Input: input}
	payload, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("encode request: %w", err)
	}
	if _, err := stdin.Write(append(payload, '\n')); err != nil {
		return nil, fmt.Errorf("write request: %w", err)
	}

	line, err := readLine(ctx, reader)
	if err != nil {
		return nil, err
	}
	var resp Response
	if err := json.Unmarshal(line, &resp); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}
	if resp.Error != nil {
		return nil, fmt.Errorf("plugin error: %s", resp.Error.Message)
	}
	return resp.Output, nil
}

func readLine(ctx context.Context, reader *bufio.Reader) ([]byte, error) {
	out := make(chan []byte, 1)
	errCh := make(chan error, 1)

	go func() {
		line, err := reader.ReadBytes('\n')
		if err != nil {
			errCh <- err
			return
		}
		out <- bytes.TrimSpace(line)
	}()

	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case err := <-errCh:
		return nil, err
	case line := <-out:
		if len(line) == 0 {
			return nil, errors.New("empty response")
		}
		return line, nil
	}
}

func errWithStderr(msg string, err error, stderr *bytes.Buffer) error {
	if stderr == nil || stderr.Len() == 0 {
		return fmt.Errorf("%s: %w", msg, err)
	}
	return fmt.Errorf("%s: %w; stderr: %s", msg, err, stderr.String())
}

func hasCapability(m Manifest, name string) bool {
	for _, cap := range m.Capabilities {
		if cap.Name == name {
			return true
		}
	}
	return false
}
