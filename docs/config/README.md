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
- `reasoning_effort`: default `xhigh`
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
- `paths`: directories to search for `plugin.yaml`
- `timeout_seconds`

### `hub`
Multi-repo workspace tracking.

- `enabled`: enable hub mode
- `workspaces`: list of repos; when empty, the current repo is used by default

### `api`
Local API server configuration.

- `enabled`: toggle API server
- `host`, `port`, `base_path`

### `ui`
UI origin and enablement.

- `enabled`: toggle UI integration
- `origin`: UI base URL (e.g. `http://localhost:5173`)

### `beads`
Beads integration.

- `enabled`: toggle bead updates
- `auto_create`, `auto_update`

### `logging`
Logging behavior.

- `level`: `debug|info|warn|error`
- `format`: `json` recommended

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

### `autonomy`
Spec/plan/bead automation controls (feature-flagged).

- `enabled`: toggle autonomy controller
- `spec_template`, `plan_template`: template paths for generated docs
- `tasks_schema`, `actions_schema`: contract paths used by parsers
- `tasks_output_template`: where `<slug>-tasks.json` is written
- `stop_conditions`: `max_fix_attempts`, `max_beads`, `stop_on_gate_failure`

## Safety notes

- `mode: yolo` always runs the Codex CLI with `--yolo`. Use intentionally.
- If `auth.enabled` is set, configure `auth.token_env` or `auth.tokens`.
- `hub.enabled` reads other repos locally; ensure you trust those paths.
