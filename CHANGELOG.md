# Changelog

All notable changes to this project will be documented in this file.

The format is based on Keep a Changelog, and this project adheres to
Semantic Versioning.

## [Unreleased]

## [0.3.0] - 2026-01-20

- Autonomy hardening: strict ACTIONS validation, schema policy flags, and run-tagged artifact paths.
- Run registry metadata enrichment and resume safety guardrails.
- Added `autocodex doctor` preflight checks.
- Structured logs now include `run_id` and `stage`.
- Autonomy smoke harness + schema validation tests.
- UI: tolerate wrapped runs payloads.

## [0.2.2] - 2026-01-20

- Fix bead ID normalization to avoid duplicate prefixes.

## [0.2.1] - 2026-01-20

- Added CLI task-input and API middleware tests to improve coverage.

## [0.2.0] - 2026-01-19

- Embedded production UI and `autocodex ui` command (API + UI served together).
- Bootstrap now seeds config/templates/skills and supports embedded defaults.
- Beads integration respects repo issue prefix when generating tasks.
- Docs + release flow updates for embedded UI builds.

## [0.1.0] - 2026-01-16

- Initial public preview of autocodex (CLI, API, UI, plugins).
- Deterministic loop orchestration (ideate → plan → implement → review → test).
- Autonomy mode for spec/plan/tasks generation with beads tracking.
- Local API for runs, events, artifacts, and memory docs.
- Web UI for run history, phases, hub workspaces, terminal sessions, and memory.
- JSON‑RPC plugin system and run snapshots for handoffs.
- Configurable Codex exec integration with optional auth/token gating.
