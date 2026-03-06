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

version_gte() {
  local current="$1"
  local target="$2"
  local IFS=.
  local -a current_parts target_parts
  read -r -a current_parts <<<"$current"
  read -r -a target_parts <<<"$target"
  local i
  for i in 0 1 2; do
    local cv="${current_parts[$i]:-0}"
    local tv="${target_parts[$i]:-0}"
    if ((10#$cv > 10#$tv)); then
      return 0
    fi
    if ((10#$cv < 10#$tv)); then
      return 1
    fi
  done
  return 0
}

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
  if printf '%s' "$raw" | tr -d '\n\r\t ' | grep -q '"connection_ok":true'; then
    return 0
  fi
  return 1
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
  if version_gte "$version" "$TARGET_BD_VERSION"; then
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

  local raw details
  raw="$(cd "$ROOT_DIR" && bd dolt show --json 2>/dev/null || true)"
  if [[ -z "$raw" ]]; then
    fail "bd dolt show --json returned no output"
    return
  fi

  local compact backend mode host port connection_ok reachable
  compact="$(printf '%s' "$raw" | tr -d '\n\r\t ')"
  backend="$(printf '%s' "$compact" | sed -nE 's/.*"backend":"?([^",}]*)"?.*/\1/p' | head -n1)"
  mode="$(printf '%s' "$compact" | sed -nE 's/.*"mode":"?([^",}]*)"?.*/\1/p' | head -n1)"
  host="$(printf '%s' "$compact" | sed -nE 's/.*"host":"?([^",}]*)"?.*/\1/p' | head -n1)"
  port="$(printf '%s' "$compact" | sed -nE 's/.*"port":([0-9]+).*/\1/p' | head -n1)"
  connection_ok=0
  reachable=0
  if printf '%s' "$compact" | grep -Eq '"connection_ok":true|"server_reachable":true|"reachable":true'; then
    connection_ok=1
  fi
  if printf '%s' "$compact" | grep -q '"mode":"embedded"'; then
    reachable=1
  fi
  details="backend=${backend:-unknown} mode=${mode:-unknown} host=${host:-unknown} port=${port:-unknown}"

  if [[ "$connection_ok" -eq 1 || "$reachable" -eq 1 ]]; then
    pass "bd dolt connection is healthy ($details)"
  else
    fail "bd dolt connection is not ready ($details). Start Dolt SQL server: $(dolt_start_cmd)"
  fi
}

check_bd_hooks() {
  if ! command -v bd >/dev/null 2>&1; then
    return
  fi
  local raw missing normalized parse_output status
  raw="$(cd "$ROOT_DIR" && bd hooks list --json 2>/dev/null || true)"
  if [[ -z "$raw" ]]; then
    warn "unable to inspect bd hooks status"
    return
  fi
  if ! printf '%s' "$raw" | grep -q '"hooks"'; then
    warn "unable to parse bd hooks list --json output"
    return
  fi
  normalized="$(printf '%s' "$raw" | sed 's/"Name":/\n"Name":/g; s/"Installed":/\n"Installed":/g')"

  set +e
  parse_output="$(
    printf '%s\n' "$normalized" | awk '
      /"Name":/ {
        line = $0
        sub(/^.*"Name":[[:space:]]*"/, "", line)
        sub(/".*$/, "", line)
        name = line
      }
      /"Installed":[[:space:]]*false/ {
        if (name != "") {
          missing = (missing == "" ? name : missing "," name)
        }
        parsed = 1
        name = ""
      }
      /"Installed":[[:space:]]*true/ {
        parsed = 1
        name = ""
      }
      END {
        if (parsed != 1) {
          exit 2
        }
        if (missing != "") {
          print missing
        }
      }
    ' | paste -sd, -
  )"
  status=$?
  set -e
  if [[ "$status" -eq 2 ]]; then
    warn "unable to parse bd hooks list --json output"
    return
  fi
  if [[ "$status" -ne 0 ]]; then
    warn "unable to parse bd hooks list --json output"
    return
  fi
  missing="${parse_output// /}"
  if [[ -z "$missing" ]]; then
    pass "bd hooks are installed"
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
    if (cd "$ROOT_DIR" && go run ./cmd/autocodex harness preflight --strict >/dev/null 2>&1); then
      pass "go-run harness preflight passes"
    else
      fail "go-run harness preflight failed"
    fi
    return
  fi

  if command -v autocodex >/dev/null 2>&1; then
    warn "go not found; using autocodex from PATH (may not match repo source)"
    if (cd "$ROOT_DIR" && autocodex harness preflight --strict >/dev/null 2>&1); then
      pass "autocodex harness preflight passes"
    else
      fail "autocodex harness preflight failed"
    fi
    return
  fi

  fail "neither go nor autocodex command is available to run harness preflight"
}

check_harness_lint() {
  if command -v go >/dev/null 2>&1; then
    if (cd "$ROOT_DIR" && go run ./cmd/autocodex harness lint >/dev/null 2>&1); then
      pass "go-run harness lint passes"
    else
      fail "go-run harness lint failed"
    fi
    return
  fi

  if command -v autocodex >/dev/null 2>&1; then
    warn "go not found; using autocodex from PATH (may not match repo source)"
    if (cd "$ROOT_DIR" && autocodex harness lint >/dev/null 2>&1); then
      pass "autocodex harness lint passes"
    else
      fail "autocodex harness lint failed"
    fi
    return
  fi

  fail "neither go nor autocodex command is available to run harness lint"
}

main() {
  require_cmd "bd" "bd command available"
  require_cmd "codex" "codex command available"

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
