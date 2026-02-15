# Changelog

All notable changes to this project will be documented in this file.

The format is based on Keep a Changelog, and this project adheres to
Semantic Versioning.

## [Unreleased]

## [0.8.0] - 2026-02-15

### Added
- Doctor: warn when Codex CLI is older than recommended (warn-only version check).

### Changed
- Tooling: bump Go toolchain baseline to Go 1.26 (CI + go.mod).
- Dependencies: update Go module dependencies (`github.com/creack/pty`, `github.com/gorilla/websocket`).
- Beads: migrate this repo's Beads backend to Dolt (embedded) for better branch/diff workflows.

### Docs
- Clarify Codex Plan/Pair interactivity limits for `codex exec`, and document modern `web_search` config via `extra_args`.
- Document Beads Dolt backend expectations and release verification gates.

## [0.7.1] - 2026-01-22

- Plugins: satisfy staticcheck in dep-license-scanner.

## [0.7.0] - 2026-01-22

- Plugins: bundle plugin binaries in release archives.
- Plugin catalog: add reference plugin implementations and contribution docs.

## [0.6.0] - 2026-01-21

- Autonomy: coordinator "swarm" mode with guardrails and collaboration opt-out.
- API: respect `base_path` and harden JSON responses.
- Docs: collaboration/parallelism guidance and improved CLI reference.

## [0.5.2] - 2026-01-21

- Feedback: stop memory loop on size limit.

## [0.5.1] - 2026-01-21

- Docs: README install snippet for the latest release.
- Tests: improve Codex exec output flushing for CI stability.

## [0.5.0] - 2026-01-21

- Loop: `--start-phase` and artifact hints.
- Runs: record model and reasoning metadata.
- Autonomy: persist artifacts for resume hints.
- Cleanup: add run deletion and cleanup hints.
- Docs: document init, cleanup, and resume UX.

## [0.4.0] - 2026-01-21

- Init: auto-init git and beads, and improve autonomy resilience.

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
