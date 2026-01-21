package autonomy

import "testing"

func TestFixBeadShortStripsPrefix(t *testing.T) {
	got := fixBeadShort("test-autocodex-jr01", "test-autocodex")
	if got != "fixjr01" {
		t.Fatalf("fixBeadShort() = %q, want %q", got, "fixjr01")
	}
}

func TestFixBeadShortNormalizesSuffix(t *testing.T) {
	got := fixBeadShort("weird--ID!!", "test-autocodex")
	if got != "fixweirdid" {
		t.Fatalf("fixBeadShort() = %q, want %q", got, "fixweirdid")
	}
}
