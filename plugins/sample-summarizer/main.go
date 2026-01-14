package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

type request struct {
	Capability string          `json:"capability"`
	Input      json.RawMessage `json:"input"`
}

type response struct {
	Output map[string]string `json:"output"`
	Error  *pluginError      `json:"error"`
}

type pluginError struct {
	Message string `json:"message"`
	Code    string `json:"code,omitempty"`
}

func main() {
	handshake := map[string]interface{}{
		"protocol_version": 1,
		"name":             "sample-summarizer",
		"transport":        "jsonrpc",
		"address":          "stdio",
	}
	b, _ := json.Marshal(handshake)
	fmt.Println(string(b))

	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		var req request
		if err := json.Unmarshal(scanner.Bytes(), &req); err != nil {
			writeResponse(response{Error: &pluginError{Message: "invalid request", Code: "bad_request"}})
			continue
		}
		switch req.Capability {
		case "summarize":
			writeResponse(response{Output: map[string]string{"summary": summarize(req.Input)}})
		default:
			writeResponse(response{Error: &pluginError{Message: "unknown capability", Code: "unknown_capability"}})
		}
	}
}

func summarize(input json.RawMessage) string {
	payload := struct {
		Text string `json:"text"`
	}{}
	_ = json.Unmarshal(input, &payload)

	text := strings.TrimSpace(payload.Text)
	if text == "" {
		return ""
	}
	if len(text) <= 80 {
		return text
	}
	return text[:80] + "..."
}

func writeResponse(resp response) {
	b, _ := json.Marshal(resp)
	fmt.Println(string(b))
}
