package terminal

import (
	"io"
	"os"
	"testing"
)

func TestDSRFilterRemovesQueryAndResponds(t *testing.T) {
	reader, writer, err := os.Pipe()
	if err != nil {
		t.Fatalf("pipe: %v", err)
	}
	defer reader.Close()

	filter := &DSRFilter{}
	out := filter.Filter(writer, []byte("hello\x1b[6n"))
	_ = writer.Close()

	resp, err := io.ReadAll(reader)
	if err != nil {
		t.Fatalf("read response: %v", err)
	}
	if string(out) != "hello" {
		t.Fatalf("expected output to strip query, got %q", string(out))
	}
	if string(resp) != string(dsrResponse) {
		t.Fatalf("expected DSR response, got %q", string(resp))
	}
}

func TestDSRFilterHandlesSplitQuery(t *testing.T) {
	filter := &DSRFilter{}
	out := filter.Filter(nil, []byte("\x1b["))
	if len(out) != 0 {
		t.Fatalf("expected no output for partial query, got %q", string(out))
	}
	out = filter.Filter(nil, []byte("6nOK"))
	if string(out) != "OK" {
		t.Fatalf("expected output after query to be OK, got %q", string(out))
	}
}

func TestDSRFilterStripsResponse(t *testing.T) {
	filter := &DSRFilter{}
	out := filter.Filter(nil, []byte("\x1b[1;1Rready"))
	if string(out) != "ready" {
		t.Fatalf("expected response to be stripped, got %q", string(out))
	}
}
