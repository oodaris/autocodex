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
| `sample-summarizer` | `summarize` | Minimal example plugin | Summary string |

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

## Sample plugin
A minimal example lives in `plugins/sample-summarizer/`.

Build it:
```bash
go build -o plugins/sample-summarizer/sample-summarizer ./plugins/sample-summarizer
```

Run it via CLI once the plugin host command is wired:
```bash
autocodex plugins run --name sample-summarizer --capability summarize --input '{"text":"hello world"}'
```

## Writing a plugin
1) Create a folder under `plugins/` with a `plugin.yaml` manifest.  
2) Implement JSON‑RPC over stdio (`capability` + `input` requests, `output` or `error` responses).  
3) Add JSON schemas for each capability input/output.  
4) Keep logs on **stderr** only; stdout is reserved for protocol messages.  

## Contributing to the catalog
See `CONTRIBUTING.md` for the checklist and submission steps.
