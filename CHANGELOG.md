# Changelog

All notable changes to this project will be documented in this file.

The format is based on Keep a Changelog, and this project adheres to
Semantic Versioning.

## [Unreleased]

### Added
- Autonomy coordinator controls for run-scoped bead selection (`selection_mode`), selector filters (`bead_ids`, `bead_prefix`), and bounded failure aggregation (`failure_summary_limit`).
- CLI overrides for coordinator scoping/filtering on `run`, `once`, and `resume` (`--bead-scope`, `--allow-all-ready-fallback`, `--bead`, `--bead-prefix`).
- Persistent fix-attempt tracking controls under `autonomy.gate.*` with configurable run/global scope and store path.

### Changed
- High-impact Harness v2 gating now enforces eval metrics from ACTIONS (`eval_scenarios`, `eval_pass_rate`, `eval_soft_failures`) against configured thresholds.
- Coordinator `require_next` now enforces single-worker execution when parallelism is configured.
- Default fix-attempt store path is now `artifacts/autonomy/fix_attempts.json` and resolves under `paths.state_dir` when relative.

### Fixed
- Gate failures now report bead-scoped errors and aggregated coordinator failure summaries instead of only returning the first error.
- Fix-bead generation now uses deterministic IDs, avoids nested `fix-fix-*` prefixes, and links fix beads to parent dependencies.

## [0.8.7] - 2026-02-24

### Fixed
- `autocodex bootstrap --profile max_capability --ready` now bootstraps the Harness v2 role pack and governance assets in fresh repos so strict preflight/lint checks pass without manual file copying.

### Docs
- Updated `AGENTS.md`, `docs/AGENTS.md`, and config/release docs to reflect self-contained Harness v2 bootstrap behavior and add a bootstrap readiness smoke checkpoint to release prep.

## [0.8.6] - 2026-02-24

### Added
- `autocodex bootstrap --profile` support with `max_capability`, `balanced`, and `max_throughput` presets.
- `autocodex bootstrap --ready` to run strict harness readiness checks as part of bootstrap.
- `autocodex bootstrap --smoke-task "<task>"` to execute an optional post-bootstrap smoke task with the configured profile.
- `autocodex harness lint` as a first-class Go command for Harness v2 policy-pack validation.

### Changed
- Config now includes a top-level `profile` field (default `max_capability`) and bootstrap applies selected profile defaults automatically for new configs or explicit `--profile` runs.
- Bootstrap now supports an end-to-end readiness workflow (repo setup -> strict preflight/lint -> optional smoke run) for faster onboarding validation.
- Harness lint checks are now Go-native (no direct `python3 scripts/harness_config_lint.py` invocation required for CLI/bootstrap/preflight workflows).

### Fixed
- Harness preflight hook auditing now warns on malformed `bd hooks list --json` output instead of incorrectly passing hooks status.
- `autocodex harness lint` now resolves the repo root deterministically from config/git ancestry so behavior is consistent from nested working directories.

### Docs
- README Beads verification guidance now references `bd dolt show --json` and `bd dolt test --json` (removing stale `bd backend show --json` guidance).

## [0.8.5] - 2026-02-24

### Changed
- Repo default config profile now enables Harness v2 and coordinator-based multi-agent execution (`autonomy.harness.enabled: true`, `autonomy.coordinator.enabled: true`, `max_parallel: 4`).

### Docs
- Added an explicit configuration profile matrix (`max_capability`, `balanced`, `max_throughput`) with copy-paste YAML snippets and host tuning guidance.
- Linked profile guidance from README, quickstart, and parallelism docs so capability/performance tradeoffs are discoverable from primary entry points.

## [0.8.4] - 2026-02-24

### Fixed
- Orchestrator heartbeat loop now exits before writing when its context is already canceled, preventing late heartbeat file writes that could trigger flaky test tempdir cleanup failures in CI.

## [0.8.3] - 2026-02-24

### Fixed
- Web build now restores `web/dist/.gitkeep` after Vite output cleanup so release/build hooks no longer leave tracked-file deletions behind.

### Docs
- Release runbook now documents the standard validated `goreleaser release --clean -f goreleaser.yml` path for this repo.

## [0.8.2] - 2026-02-24

### Fixed
- Harness CLI preflight now auto-starts a local Dolt SQL server when `bd` reports the configured Dolt endpoint as unreachable.
- Auto-started Dolt server is now launched detached so connectivity persists after preflight exits.

### Changed
- Harness preflight runbook now documents automatic Dolt startup behavior and the `AUTO_START_DOLT_SERVER=0` opt-out.

### Docs
- CLI and release docs were aligned with current harness/preflight semantics and release publishing caveats.

## [0.8.1] - 2026-02-24

### Added
- Harness v2 operating pack assets under `.codex/agents/` and repo harness documentation/eval catalogs.
- `autocodex harness preflight` command and strict preflight runbook tooling.
- Beads 0.56.1 upgrade/runbook artifacts for Dolt-backed readiness checks.

### Changed
- Doctor checks now include Codex feature-capability validation plus Beads version (`>=0.56.1`) and Dolt readiness assessment.
- Harness strict mode now promotes actionable doctor warnings to blocking preflight failures.
- Preflight script now validates Beads hooks status, prefers repo-local `go run` CLI checks, and supports enforced JSONL hook gating (`ENFORCE_JSONL_HOOKS=1`).

### Fixed
- `autocodex harness --help` now returns harness usage instead of unknown-subcommand failure.
- `autocodex harness preflight --json` now emits parseable JSON without trailing non-JSON text.
- Dolt readiness reporting now treats ambiguous server-mode reachability as warning (not pass) and avoids indefinite subprocess hangs via command timeouts.

### Docs
- Updated release and operator docs for Beads 0.56.1 (`bd sync` deprecated/no-op, Dolt canonical state, hook-enforced JSONL mirror guidance).

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
