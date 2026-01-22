# autocodex config reference

This reference explains the `autocodex.yaml` configuration file. Use
`config.example.yaml` as a starting point and validate with the schema:

- Example: `config.example.yaml`
- Schema: `docs/contracts/config.schema.json`

## Required fields

- `version`: must be `v1`
- `mode`: `yolo` or `safe`

## Core sections

### `codex`
Controls the Codex CLI invocation.

- `cli_path`: path or binary name (default: `codex`)
- `model`: default `gpt-5.2-codex`
- `reasoning_effort`: default `xhigh` (passed to Codex as `-c model_reasoning_effort=...`)
  - Model-specific limits apply (examples):
    - `gpt-5.1*`: `low|medium|high` (no `xhigh`)
  - `xhigh` is model-dependent; use `medium`/`high` if unsure.
- `collaboration_enabled`: enable collaboration defaults (default: `true`)
- `collaboration_mode`: Codex collaboration mode (default: `auto`, passed as `-c collaboration_mode=...`)
- `preset`: collaboration preset (default: `default`, passed as `-c collaboration_mode_preset=...`, requires `collaboration_mode`)
  - Set `collaboration_enabled: false` to disable collaboration and clear mode/preset defaults.
- `timeout_seconds`: per-run timeout
- `extra_args`: additional CLI flags
- `approval_policy` and `sandbox_mode`: ignored in `mode: yolo`
- `json_output`: emit JSONL events from `codex exec` (requires `output_last_message`)
- `output_last_message`: write the final agent message to an artifact per phase
- `prompt_stdin`: force prompt to be sent via stdin (useful for multi-line tasks)

### `paths`
Local storage locations for run state and artifacts.

- `state_dir`: persistent state (default `.autocodex`)
- `memory_dir`, `logs_dir`, `artifacts_dir`, `runs_dir`

### `skills`
Skill resolution settings.

- `paths`: list of skill directories
- `allowlist`, `denylist`: optional filters

Note: `autocodex bootstrap` writes a minimal skill pack into `skills/` and
sets `skills.paths` to `["skills"]` in the default config. If you want to use
shared or external skills, add additional paths here.

### `plugins`
External plugin settings.

- `enabled`: toggle plugins
- `paths`: directories to search for `plugin.yaml` (defaults include repo `plugins/` and system share paths)
- `timeout_seconds`

### `hub`
Multi-repo workspace tracking.

- `enabled`: enable hub mode
- `workspaces`: list of repos; when empty, the current repo is used by default

### `api`
Local API server configuration.

- `enabled`: toggle API server
- `host`, `port`, `base_path` (API/UI served under the base path when set)

### `ui`
UI origin and enablement.

- `enabled`: serve the embedded UI when running `autocodex api`
- `origin`: UI base URL for CORS when running a separate dev server (e.g. `http://localhost:5173`)

### `beads`
Beads integration.

- `enabled`: toggle bead updates
- `auto_create`, `auto_update`

### `cleanup`
Local retention for run artifacts and logs.

- `retention_days`: remove completed runs older than this many days when running `autocodex cleanup`

### `logging`
Logging behavior.

- `level`: `debug|info|warn|error`
- `format`: `json` (default) or `text`

### `auth`
API token enforcement.

- `enabled`: require tokens
- `token_env`: optional env var to read a token from
- `tokens`: explicit token list for local usage

### `loop`
Autonomy loop controls.

- `mode`: `bounded` or `continuous`
- `max_iterations`
- `phases`: default `ideate → plan → implement → review → test`
- `stop_conditions`: duration, idle, failure, heartbeat
- `feedback`: controls memory/event/artifact context injection
  - Defaults to `on` when `autonomy.enabled: true`
  - `sources` may include `snapshot` (used by `autocodex resume` to inject snapshot context)
  - `snapshot_path` is set automatically by `autocodex resume` (usually leave empty)
- `memory_mode`: how memory docs are injected (`inline`, `summary_ref`, `ref_only`)
- `snapshot_mode`: how resume snapshots are injected (`inline`, `summary_ref`, `ref_only`)
- `summary_max_lines`: max lines for summaries when using `summary_ref`
- Recommendation: use `summary_ref` for speed; `inline` for full determinism
  - `ref_only` includes a short instruction telling the agent to read referenced files when needed
  - Summaries prefer headings/bullets; falls back to the first non-empty lines

### `autonomy`
Spec/plan/bead automation controls (feature-flagged).

- `enabled`: toggle autonomy controller
- `require_actions`: require a valid ACTIONS payload for autonomy runs (defaults to true when autonomy enabled)
- `require_next`: require `next` to be explicit when multiple beads are ready (defaults to true when autonomy enabled)
- `require_bd`: require `bd` to be installed when autonomy is enabled (defaults to true when autonomy enabled)
- `fail_on_schema_error`: fail immediately when task/action JSON fails schema validation (default true)
- `allow_fallback_tasks`: when task JSON is invalid, allow writing a fallback tasks file (default true)
- `keep_invalid_payloads`: keep invalid task/action outputs on disk for debugging (default true)
- `spec_template`, `plan_template`: template paths for generated docs
- `tasks_schema`, `actions_schema`: contract paths used by parsers
- `tasks_output_template`: where `<slug>-tasks.json` is written
- `coordinator`: optional parallel bead runner
  - `enabled`: spawn parallel bead runs instead of sequential loop
  - `max_parallel`: max concurrent beads (default 2, `0` = unlimited)
  - `strategy`: `bead` runs beads in parallel; `phase` runs isolated phases in parallel and does not share artifacts between phases
  - `fail_fast`: stop all parallel beads on the first error (default false)
  - Note: `require_next` is ignored in parallel mode
  - Note: parallel mode isolates memory docs per bead under `memory/beads/<id>`
  - `strategy`: `bead` (default) or `phase` (runs isolated phases in parallel; does not share phase artifacts)
- `stop_conditions`: `max_fix_attempts`, `max_beads`, `stop_on_gate_failure`

## Safety notes

- `mode: yolo` always runs the Codex CLI with `--yolo`. Use intentionally.
- If `auth.enabled` is set, configure `auth.token_env` or `auth.tokens`.
- `hub.enabled` reads other repos locally; ensure you trust those paths.
