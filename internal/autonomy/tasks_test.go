package autonomy

import "testing"

func TestExtractJSONRawObject(t *testing.T) {
	input := `{"version":"1.0","tasks":[]}`
	output, err := extractJSON(input)
	if err != nil {
		t.Fatalf("expected JSON, got error: %v", err)
	}
	if output != input {
		t.Fatalf("unexpected payload: %s", output)
	}
}

func TestExtractJSONRawArray(t *testing.T) {
	input := `[{"id":"t1"}]`
	output, err := extractJSON(input)
	if err != nil {
		t.Fatalf("expected JSON array, got error: %v", err)
	}
	if output != input {
		t.Fatalf("unexpected payload: %s", output)
	}
}

func TestExtractJSONFenced(t *testing.T) {
	input := "```json\n{\"tasks\":[]}\n```"
	output, err := extractJSON(input)
	if err != nil {
		t.Fatalf("expected fenced JSON, got error: %v", err)
	}
	expected := "{\"tasks\":[]}"
	if output != expected {
		t.Fatalf("unexpected payload: %s", output)
	}
}

func TestExtractJSONEmbedded(t *testing.T) {
	input := "note\n{\"tasks\":[{\"id\":\"t1\"}]}\nthanks"
	output, err := extractJSON(input)
	if err != nil {
		t.Fatalf("expected embedded JSON, got error: %v", err)
	}
	expected := "{\"tasks\":[{\"id\":\"t1\"}]}"
	if output != expected {
		t.Fatalf("unexpected payload: %s", output)
	}
}

func TestExtractJSONInvalid(t *testing.T) {
	_, err := extractJSON("not json")
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
}

func TestExtractJSONEmpty(t *testing.T) {
	_, err := extractJSON("  ")
	if err == nil {
		t.Fatal("expected error for empty JSON")
	}
}
