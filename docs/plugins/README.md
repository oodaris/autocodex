# Plugins

autocodex plugins are external processes described by a `plugin.yaml` or `plugin.json` manifest.
The host launches the plugin, performs a JSON handshake, and then issues capability calls.

Protocol spec: `docs/contracts/plugin-protocol.md`.

## Catalog
| Plugin | Capability | Purpose | Output |
| --- | --- | --- | --- |
| `repo-indexer` | `index` | Project map: languages, key dirs, test commands, services | JSON summary for models + humans |
| `test-runner` | `run` | Run scoped tests with timeouts | Pass/fail + command logs |
| `diff-summarizer` | `summarize` | Summarize git diff + risk flags | Areas + risk flags |
| `dep-license-scanner` | `scan` | Extract dependencies + license risks | Dependencies + risk flags |
| `knowledge-extractor` | `extract` | Parse docs into structured summaries | Doc list + headings/snippets |
| `plan-compliance` | `check` | Validate plan sections + open tasks | Status + missing items |
| `evidence-collector` | `collect` | Capture evidence artifacts | Artifact manifest |
| `example-summarizer` | `summarize` | Minimal example plugin | Summary string |

## Distribution
Release archives include **prebuilt plugins + manifests** under `plugins/`.
The install script places them in:
```
${PREFIX}/share/autocodex/plugins
```
(`PREFIX` is the parent of your `DEST` bin directory.)

Override the install location:
```bash
PLUGIN_DEST=~/.local/share/autocodex/plugins \
  curl -fsSL https://raw.githubusercontent.com/oodaris/autocodex/main/scripts/install.sh | bash
```

Default plugin search paths include:
- `plugins` (repo‑local)
- `/usr/local/share/autocodex/plugins`
- `/usr/share/autocodex/plugins`
- `/opt/homebrew/share/autocodex/plugins`
- `~/.local/share/autocodex/plugins` (if present)

Override these via `plugins.paths` in `autocodex.yaml`.

If you move binaries manually, copy the `plugins/` folder from the release
archive to a path in `plugins.paths`, e.g.:
```bash
sudo mkdir -p /usr/local/share/autocodex/plugins
sudo cp -R plugins/. /usr/local/share/autocodex/plugins
```

Build all default plugins from source (optional):
```bash
for p in plugins/*; do
  if [ -f "$p/main.go" ]; then
    name="$(basename "$p")"
    go build -o "$p/$name" "./$p"
  fi
done
```

## Can I run multiple plugins at the same time?
Yes. Plugins are independent processes, so you can run them sequentially or in parallel. A few options:
- **Sequential (safe default)**: pipe outputs to files and chain the next step.
- **Parallel (faster)**: run multiple plugin calls in background shells (`&`) and `wait`.

Example (parallel):
```bash
autocodex plugins --action run --name diff-summarizer --capability summarize --input '{"root":"."}' > /tmp/diff.json &
autocodex plugins --action run --name dep-license-scanner --capability scan --input '{"root":"."}' > /tmp/licenses.json &
wait
```

## Recipes (common workflows)

### Repo onboarding
Create a quick project map and extract docs for context:
```bash
autocodex plugins --action run --name repo-indexer --capability index \
  --input '{"root":"."}' > /tmp/repo-index.json

autocodex plugins --action run --name knowledge-extractor --capability extract \
  --input '{"root":".","paths":["docs"],"max_files":50}' > /tmp/docs.json
```

### PR risk triage (no code changes yet)
```bash
autocodex plugins --action run --name diff-summarizer --capability summarize \
  --input '{"root":".","base":"origin/main","head":"HEAD"}' > /tmp/diff.json

autocodex plugins --action run --name dep-license-scanner --capability scan \
  --input '{"root":".","deny_licenses":["gpl-3.0","agpl-3.0"]}' > /tmp/licenses.json
```

### CI helper / fast tests
```bash
autocodex plugins --action run --name test-runner --capability run \
  --input '{"root":".","commands":["go test ./..."],"timeout_seconds":1200}' \
  > /tmp/tests.json
```

### Plan compliance + evidence bundle
```bash
autocodex plugins --action run --name plan-compliance --capability check \
  --input '{"plan_path":"docs/plans/my-plan.md","required_sections":["acceptance criteria","tests"]}' \
  > /tmp/plan-check.json

autocodex plugins --action run --name evidence-collector --capability collect \
  --input '{"root":".","globs":["logs/*.log","reports/*.json"],"output_dir":"artifacts"}' \
  > /tmp/evidence.json
```

## Output notes
- Plugins output **one JSON object per request** to stdout.
- Log output should go to **stderr** (stdout is reserved for protocol messages).
- For large inputs, pass `--input-file` and keep payloads small when possible.

## Layout conventions
Each plugin lives in its own folder under `plugins/`:
```
plugins/<name>/
  plugin.yaml
  main.go
  schemas/
    <capability>.input.json
    <capability>.output.json
```

## Manifest example
```yaml
name: repo-indexer
version: 0.1.0
protocol_version: 1
entrypoint: ./repo-indexer
transport: jsonrpc
capabilities:
  - name: index
    input_schema: schemas/index.input.json
    output_schema: schemas/index.output.json
```

## Example plugin
A minimal example lives in `plugins/example-summarizer/`.

Build it:
```bash
go build -o plugins/example-summarizer/example-summarizer ./plugins/example-summarizer
```

Run it via CLI once the plugin host command is wired:
```bash
autocodex plugins run --name example-summarizer --capability summarize --input '{"text":"hello world"}'
```

## Writing a plugin
1) Create a folder under `plugins/` with a `plugin.yaml` manifest.  
2) Implement JSON‑RPC over stdio (`capability` + `input` requests, `output` or `error` responses).  
3) Add JSON schemas for each capability input/output.  
4) Keep logs on **stderr** only; stdout is reserved for protocol messages.  

## Contributing to the catalog
See `CONTRIBUTING.md` for the checklist and submission steps.
