package autonomy

import "testing"

func TestFixBeadShortStripsPrefix(t *testing.T) {
	got := fixBeadShort("test-autocodex-jr01", "test-autocodex")
	if got != "jr01" {
		t.Fatalf("fixBeadShort() = %q, want %q", got, "jr01")
	}
}

func TestFixBeadShortNormalizesSuffix(t *testing.T) {
	got := fixBeadShort("weird--ID!!", "test-autocodex")
	if got != "weirdid" {
		t.Fatalf("fixBeadShort() = %q, want %q", got, "weirdid")
	}
}

func TestFixBeadShortStripsNestedFixPrefix(t *testing.T) {
	got := fixBeadShort("integration-fix001060934", "integration")
	if got != "00106093" {
		t.Fatalf("fixBeadShort() = %q, want %q", got, "00106093")
	}
}

func TestFixReasonSignatureStable(t *testing.T) {
	first := fixReasonSignature("integration-001", "Typecheck failed")
	second := fixReasonSignature("integration-001", "Typecheck failed")
	if first != second {
		t.Fatalf("fixReasonSignature must be stable, got %q and %q", first, second)
	}
}
