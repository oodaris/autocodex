#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
FAILURES=0

pass() {
  printf 'PASS: %s\n' "$1"
}

warn() {
  printf 'WARN: %s\n' "$1"
}

fail() {
  printf 'FAIL: %s\n' "$1"
  FAILURES=$((FAILURES + 1))
}

require_cmd() {
  local cmd="$1"
  local label="$2"
  if command -v "$cmd" >/dev/null 2>&1; then
    pass "$label"
  else
    fail "$label (missing command: $cmd)"
  fi
}

check_bd_state() {
  if ! command -v bd >/dev/null 2>&1; then
    return
  fi
  if bd info --json >/dev/null 2>&1; then
    pass "bd repository state is initialized"
  else
    fail "bd repository is not initialized (run: bd init --from-jsonl)"
  fi
}

check_harness_preflight() {
  if command -v autocodex >/dev/null 2>&1; then
    if autocodex harness preflight --config "$ROOT_DIR/config.example.yaml" --strict >/dev/null 2>&1; then
      pass "autocodex harness preflight passes"
    else
      fail "autocodex harness preflight failed"
    fi
  else
    warn "autocodex binary not on PATH; using go run harness preflight"
    if go run ./cmd/autocodex harness preflight --config "$ROOT_DIR/config.example.yaml" --strict >/dev/null 2>&1; then
      pass "go-run harness preflight passes"
    else
      fail "go-run harness preflight failed"
    fi
  fi
}

check_harness_lint() {
  if python3 "$ROOT_DIR/scripts/harness_config_lint.py" >/dev/null 2>&1; then
    pass "harness config lint passes"
  else
    fail "harness config lint failed"
  fi
}

main() {
  require_cmd "bd" "bd command available"
  require_cmd "codex" "codex command available"
  require_cmd "python3" "python3 command available"

  check_bd_state
  check_harness_preflight
  check_harness_lint

  if [[ "$FAILURES" -gt 0 ]]; then
    printf '\nHarness preflight failed with %s issue(s).\n' "$FAILURES"
    exit 1
  fi

  printf '\nHarness preflight passed.\n'
}

main
