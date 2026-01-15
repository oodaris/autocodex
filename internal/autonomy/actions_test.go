package autonomy

import "testing"

func TestExtractJSONBlockMarkers(t *testing.T) {
	input := "foo\nACTIONS_JSON_START\n{\"version\":\"1.0\",\"summary\":\"ok\",\"next\":{\"type\":\"none\"}}\nACTIONS_JSON_END\nbar"
	output, err := extractJSONBlock(input)
	if err != nil {
		t.Fatalf("expected JSON block, got error: %v", err)
	}
	expected := "{\"version\":\"1.0\",\"summary\":\"ok\",\"next\":{\"type\":\"none\"}}"
	if output != expected {
		t.Fatalf("unexpected payload: %s", output)
	}
}

func TestExtractJSONBlockFenced(t *testing.T) {
	input := "```json\n{\"version\":\"1.0\",\"summary\":\"ok\",\"next\":{\"type\":\"none\"}}\n```"
	output, err := extractJSONBlock(input)
	if err != nil {
		t.Fatalf("expected JSON block, got error: %v", err)
	}
	expected := "{\"version\":\"1.0\",\"summary\":\"ok\",\"next\":{\"type\":\"none\"}}"
	if output != expected {
		t.Fatalf("unexpected payload: %s", output)
	}
}
