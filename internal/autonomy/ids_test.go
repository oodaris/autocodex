package autonomy

import "testing"

func TestNormalizeBeadIDPreservesSuffixHyphens(t *testing.T) {
	id := "test-autocodex-jr2"
	if normalizeBeadID(id) != id {
		t.Fatalf("expected %s to remain unchanged", id)
	}
}

func TestNormalizeTaskIDAvoidsDoublePrefix(t *testing.T) {
	prefix := "test-autocodex"
	id := "test-autocodex-jr2"
	normalized := normalizeTaskID(id, prefix)
	if normalized != id {
		t.Fatalf("expected %s, got %s", id, normalized)
	}
}

func TestNormalizeTaskIDAddsPrefix(t *testing.T) {
	prefix := "test-autocodex"
	id := "jr2"
	normalized := normalizeTaskID(id, prefix)
	if normalized != "test-autocodex-jr2" {
		t.Fatalf("expected prefixed id, got %s", normalized)
	}
}

func TestNormalizeTaskIDReplacesOtherPrefix(t *testing.T) {
	prefix := "test-autocodex"
	id := "autocodex-jr2"
	normalized := normalizeTaskID(id, prefix)
	if normalized != "test-autocodex-jr2" {
		t.Fatalf("expected %s, got %s", "test-autocodex-jr2", normalized)
	}
}
