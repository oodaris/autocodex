#!/usr/bin/env python3
"""Lint invariants for the autocodex Harness v2 pack."""

from __future__ import annotations

import sys
from pathlib import Path

try:
    import tomllib
except Exception as exc:  # pragma: no cover
    raise SystemExit(f"tomllib unavailable: {exc}")

REPO_ROOT = Path(__file__).resolve().parent.parent
CONFIG_PATH = REPO_ROOT / ".codex" / "config.toml"
ROLES_DIR = REPO_ROOT / ".codex" / "agents"
OPERATING_PACK = REPO_ROOT / "docs" / "agents" / "autocodex-harness-v2-operating-pack.md"
EVAL_DOCS = [
    REPO_ROOT / "docs" / "agents" / "harness-evals" / "README.md",
    REPO_ROOT / "docs" / "agents" / "harness-evals" / "golden-task-catalog.md",
    REPO_ROOT / "docs" / "agents" / "harness-evals" / "failure-mode-catalog.md",
]
PREFLIGHT_SCRIPT = REPO_ROOT / "scripts" / "dev" / "harness-cli-preflight.sh"
PREFLIGHT_RUNBOOK = REPO_ROOT / "docs" / "runbooks" / "harness-cli-preflight.md"

EXPECTED_PROFILE = "max_capability"
EXPECTED_ROLES = {
    "workflow_orchestrator",
    "requirements_clarifier",
    "design_strategist",
    "tracking_operator",
    "backend_executor",
    "frontend_executor",
    "browser_validator",
    "quality_gate_runner",
    "independent_critic",
    "commit_curator",
    "release_evidence_operator",
    "agentic_ai_architect",
}
REQUIRED_PACK_MARKERS = [
    "pattern a",
    "pattern e",
    "non-bypassable gate stack",
    "lifecycle/admission contract",
    "high-impact trigger criteria",
]


def load_toml(path: Path, errors: list[str]) -> dict:
    if not path.exists():
        errors.append(f"missing file: {path}")
        return {}
    try:
        return tomllib.loads(path.read_text(encoding="utf-8"))
    except Exception as exc:
        errors.append(f"invalid TOML in {path}: {exc}")
        return {}


def main() -> int:
    errors: list[str] = []

    cfg = load_toml(CONFIG_PATH, errors)
    if not cfg:
        _report(errors)
        return 1

    if cfg.get("profile") != EXPECTED_PROFILE:
        errors.append(f".codex/config.toml profile must be '{EXPECTED_PROFILE}'")

    agents_table = cfg.get("agents")
    if not isinstance(agents_table, dict):
        errors.append(".codex/config.toml missing [agents] table")
        agents_table = {}

    role_entries = {k: v for k, v in agents_table.items() if k != "max_threads"}
    missing_roles = EXPECTED_ROLES - set(role_entries.keys())
    extra_roles = set(role_entries.keys()) - EXPECTED_ROLES
    if missing_roles:
        errors.append(f"missing required roles: {sorted(missing_roles)}")
    if extra_roles:
        errors.append(f"unexpected roles present: {sorted(extra_roles)}")

    for role in sorted(EXPECTED_ROLES):
        entry = role_entries.get(role)
        if not isinstance(entry, dict):
            errors.append(f"[agents.{role}] must be a table")
            continue
        config_file = entry.get("config_file")
        if not isinstance(config_file, str) or not config_file.strip():
            errors.append(f"[agents.{role}] missing config_file")
            continue
        role_path = REPO_ROOT / ".codex" / config_file
        role_cfg = load_toml(role_path, errors)
        features = role_cfg.get("features")
        if not isinstance(features, dict):
            errors.append(f"{role} missing [features] table")
        else:
            for key in ("multi_agent", "shell_tool", "unified_exec", "shell_snapshot", "runtime_metrics"):
                if key not in features:
                    errors.append(f"{role} missing features.{key}")
        instructions = role_cfg.get("developer_instructions")
        if not isinstance(instructions, str) or not instructions.strip():
            errors.append(f"{role} missing developer_instructions")

    if not ROLES_DIR.exists():
        errors.append(f"missing roles directory: {ROLES_DIR}")

    if not OPERATING_PACK.exists():
        errors.append(f"missing operating pack: {OPERATING_PACK}")
    else:
        text = OPERATING_PACK.read_text(encoding="utf-8").lower()
        for marker in REQUIRED_PACK_MARKERS:
            if marker not in text:
                errors.append(f"operating pack missing marker: {marker!r}")

    for path in EVAL_DOCS:
        if not path.exists():
            errors.append(f"missing eval doc: {path}")

    if not PREFLIGHT_SCRIPT.exists():
        errors.append(f"missing preflight script: {PREFLIGHT_SCRIPT}")
    elif not PREFLIGHT_SCRIPT.read_text(encoding="utf-8").startswith("#!/usr/bin/env bash"):
        errors.append("preflight script must start with bash shebang")

    if not PREFLIGHT_RUNBOOK.exists():
        errors.append(f"missing preflight runbook: {PREFLIGHT_RUNBOOK}")
    else:
        runbook = PREFLIGHT_RUNBOOK.read_text(encoding="utf-8").lower()
        for marker in ("harness preflight", "scripts/dev/harness-cli-preflight.sh", "harness preflight passed"):
            if marker not in runbook:
                errors.append(f"preflight runbook missing marker: {marker!r}")

    if errors:
        _report(errors)
        return 1

    print("Harness config lint passed: autocodex harness role pack and governance assets validated.")
    return 0


def _report(errors: list[str]) -> None:
    print("Harness config lint failed:")
    for issue in errors:
        print(f" - {issue}")


if __name__ == "__main__":
    raise SystemExit(main())
