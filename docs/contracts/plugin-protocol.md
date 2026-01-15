# autocodex Plugin Protocol (v1)

## Overview
Plugins are external processes launched by autocodex. Communication happens via a versioned RPC protocol to avoid Go native plugin limitations and to allow multi-language plugins.

## Discovery
A plugin is discovered by scanning configured plugin paths for a manifest file named `plugin.yaml` or `plugin.json`.

## Manifest (v1)
Minimal required fields:
```yaml
name: example-plugin
version: 0.1.0
protocol_version: 1
entrypoint: ./example-plugin
transport: grpc  # grpc | jsonrpc
capabilities:
  - name: summarize
    input_schema: schemas/summarize.input.json
    output_schema: schemas/summarize.output.json
```

## Handshake
When launched, the plugin must emit a single JSON line to stdout within the configured timeout:
```json
{
  "protocol_version": 1,
  "name": "example-plugin",
  "transport": "grpc",
  "address": "127.0.0.1:0"
}
```
- For `grpc`, the plugin listens on a TCP port and returns `address`.
- For `jsonrpc`, the plugin communicates over stdio and sets `address` to `"stdio"`.

## Capability calls
Requests contain a capability name and a JSON payload:
```json
{
  "capability": "summarize",
  "input": { "text": "..." }
}
```
Responses return a JSON payload or error:
```json
{
  "output": { "summary": "..." },
  "error": null
}
```

## Versioning
- `protocol_version` must match the host’s supported version.
- Plugins must fail fast if the host requests an unsupported version.

## Timeouts & limits
- Handshake must complete within `plugins.timeout_seconds`.
- Each capability call must enforce a per-call timeout.

## Security
- autocodex binds only to localhost.
- Plugins are untrusted; consider running with minimal OS permissions.
