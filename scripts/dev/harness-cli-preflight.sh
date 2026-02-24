#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
FAILURES=0
TARGET_BD_VERSION="0.56.1"
ENFORCE_JSONL_HOOKS="${ENFORCE_JSONL_HOOKS:-0}"
AUTO_START_DOLT_SERVER="${AUTO_START_DOLT_SERVER:-1}"
DOLT_HOST="${DOLT_HOST:-127.0.0.1}"
DOLT_PORT="${DOLT_PORT:-3307}"
DOLT_LOG_PATH="${DOLT_LOG_PATH:-/tmp/autocodex-dolt-sql-server.log}"

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

dolt_start_cmd() {
  printf 'dolt sql-server --data-dir "%s/.beads/dolt" --host %s --port %s' "$ROOT_DIR" "$DOLT_HOST" "$DOLT_PORT"
}

bd_dolt_test_ok() {
  local raw
  raw="$(cd "$ROOT_DIR" && bd dolt test --json 2>/dev/null || true)"
  if [[ -z "$raw" ]]; then
    return 1
  fi
  BD_DOLT_TEST_JSON="$raw" python3 - <<'PY'
import json
import os
import sys

raw = os.environ.get("BD_DOLT_TEST_JSON", "")
try:
    data = json.loads(raw)
except Exception:
    sys.exit(1)

if data.get("connection_ok") is True:
    sys.exit(0)
sys.exit(1)
PY
}

ensure_dolt_server() {
  if ! command -v bd >/dev/null 2>&1; then
    return 1
  fi
  if bd_dolt_test_ok; then
    return 0
  fi

  if [[ "$AUTO_START_DOLT_SERVER" != "1" ]]; then
    return 1
  fi
  if ! command -v dolt >/dev/null 2>&1; then
    warn "dolt command not found; cannot auto-start Dolt SQL server"
    return 1
  fi

  warn "bd cannot reach Dolt server; attempting to auto-start local Dolt SQL server"
  nohup dolt sql-server --data-dir "$ROOT_DIR/.beads/dolt" --host "$DOLT_HOST" --port "$DOLT_PORT" >"$DOLT_LOG_PATH" 2>&1 < /dev/null &

  local i
  for i in {1..15}; do
    sleep 0.2
    if bd_dolt_test_ok; then
      pass "auto-started Dolt SQL server ($DOLT_HOST:$DOLT_PORT)"
      return 0
    fi
  done

  warn "auto-start attempt did not make Dolt reachable (see $DOLT_LOG_PATH)"
  return 1
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
  local output
  if output="$(cd "$ROOT_DIR" && bd info --json 2>&1)"; then
    pass "bd repository state is initialized"
  else
    local single_line
    single_line="$(printf '%s' "$output" | tr '\n' ' ')"
    if [[ "$single_line" == *"Dolt server unreachable"* ]] || [[ "$single_line" == *"connect: connection refused"* ]]; then
      if ensure_dolt_server && output="$(cd "$ROOT_DIR" && bd info --json 2>&1)"; then
        pass "bd repository state is initialized"
        return
      fi
      fail "bd cannot reach Dolt server (run: $(dolt_start_cmd))"
    elif [[ "$single_line" == *"bd init"* ]]; then
      fail "bd repository is not initialized (run: cd \"$ROOT_DIR\" && bd onboard; optional mirror setup: bd migrate sync beads-sync)"
    else
      fail "bd repository check failed ($single_line)"
    fi
  fi
}

check_bd_version() {
  if ! command -v bd >/dev/null 2>&1; then
    return
  fi
  local raw version
  raw="$(bd --version 2>/dev/null || true)"
  version="$(printf '%s' "$raw" | sed -nE 's/.*([0-9]+\.[0-9]+\.[0-9]+).*/\1/p' | head -n1)"
  if [[ -z "$version" ]]; then
    warn "unable to parse bd version output ($raw)"
    return
  fi
  if python3 - "$version" "$TARGET_BD_VERSION" <<'PY'
import sys

def parse(v: str) -> tuple[int, int, int]:
    return tuple(int(part) for part in v.split("."))

current = parse(sys.argv[1])
target = parse(sys.argv[2])
sys.exit(0 if current >= target else 1)
PY
  then
    pass "bd version $version meets target >= $TARGET_BD_VERSION"
  else
    fail "bd version $version is below target >= $TARGET_BD_VERSION"
  fi
}

check_bd_dolt_connection() {
  if ! command -v bd >/dev/null 2>&1; then
    return
  fi
  if ensure_dolt_server; then
    pass "bd dolt baseline connection test is healthy"
  fi

  local raw status details
  raw="$(cd "$ROOT_DIR" && bd dolt show --json 2>/dev/null || true)"
  if [[ -z "$raw" ]]; then
    fail "bd dolt show --json returned no output"
    return
  fi

  set +e
  details="$(
    BD_DOLT_SHOW_JSON="$raw" python3 - <<'PY'
import json
import os
import sys

raw = os.environ.get("BD_DOLT_SHOW_JSON", "")
try:
    data = json.loads(raw)
except Exception:
    print("invalid-json")
    sys.exit(2)

backend = data.get("backend", "")
host = data.get("host", "")
port = data.get("port", "")
mode = str(data.get("mode", "")).strip().lower()
ok = data.get("connection_ok")
reachable = data.get("server_reachable", data.get("reachable"))

def as_bool(value):
    if isinstance(value, bool):
        return value
    if isinstance(value, str):
        text = value.strip().lower()
        if text in {"true", "1", "yes", "ok", "reachable", "connected"}:
            return True
        if text in {"false", "0", "no", "unreachable", "failed", "disconnected"}:
            return False
    return None

if mode == "embedded":
    print(f"backend={backend} mode=embedded host={host} port={port}")
    sys.exit(0)

if as_bool(ok) is True or as_bool(reachable) is True:
    print(f"backend={backend} mode={mode} host={host} port={port}")
    sys.exit(0)
if as_bool(ok) is False or as_bool(reachable) is False:
    print(f"backend={backend} mode={mode} host={host} port={port}")
    sys.exit(1)

if mode == "server":
    print(f"backend={backend} mode=server host={host} port={port}")
    sys.exit(1)

print(f"backend={backend} mode={mode} host={host} port={port}")
sys.exit(1)
PY
)"
  status=$?
  set -e

  if [[ "$status" -eq 0 ]]; then
    pass "bd dolt connection is healthy ($details)"
  elif [[ "$status" -eq 1 ]]; then
    fail "bd dolt connection is not ready ($details). Start Dolt SQL server: $(dolt_start_cmd)"
  else
    fail "unable to parse bd dolt show --json output"
  fi
}

check_bd_hooks() {
  if ! command -v bd >/dev/null 2>&1; then
    return
  fi
  local raw missing status
  raw="$(cd "$ROOT_DIR" && bd hooks list --json 2>/dev/null || true)"
  if [[ -z "$raw" ]]; then
    warn "unable to inspect bd hooks status"
    return
  fi

  set +e
  missing="$(
    BD_HOOKS_JSON="$raw" python3 - <<'PY'
import json
import os
import sys

raw = os.environ.get("BD_HOOKS_JSON", "")
try:
    data = json.loads(raw)
except Exception:
    sys.exit(2)

hooks = data.get("hooks", [])
missing = [h.get("Name", "") for h in hooks if not h.get("Installed", False)]
if missing:
    print(", ".join([name for name in missing if name]))
    sys.exit(1)
sys.exit(0)
PY
)"
  status=$?
  set -e
  if [[ "$status" -eq 0 ]]; then
    pass "bd hooks are installed"
    return
  fi
  if [[ "$status" -eq 2 ]]; then
    warn "unable to parse bd hooks list --json output"
    return
  fi

  if [[ "$ENFORCE_JSONL_HOOKS" == "1" ]]; then
    fail "bd hooks missing ($missing). Run: cd \"$ROOT_DIR\" && bd hooks install"
  else
    warn "bd hooks missing ($missing). JSONL mirror may drift; set ENFORCE_JSONL_HOOKS=1 to require hooks"
  fi
}

check_harness_preflight() {
  if command -v go >/dev/null 2>&1; then
    if (cd "$ROOT_DIR" && go run ./cmd/autocodex harness preflight --config "$ROOT_DIR/config.example.yaml" --strict >/dev/null 2>&1); then
      pass "go-run harness preflight passes"
    else
      fail "go-run harness preflight failed"
    fi
    return
  fi

  if command -v autocodex >/dev/null 2>&1; then
    warn "go not found; using autocodex from PATH (may not match repo source)"
    if (cd "$ROOT_DIR" && autocodex harness preflight --config "$ROOT_DIR/config.example.yaml" --strict >/dev/null 2>&1); then
      pass "autocodex harness preflight passes"
    else
      fail "autocodex harness preflight failed"
    fi
    return
  fi

  fail "neither go nor autocodex command is available to run harness preflight"
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
  check_bd_version
  check_bd_dolt_connection
  check_bd_hooks
  check_harness_preflight
  check_harness_lint

  if [[ "$FAILURES" -gt 0 ]]; then
    printf '\nHarness preflight failed with %s issue(s).\n' "$FAILURES"
    exit 1
  fi

  printf '\nHarness preflight passed.\n'
}

main
