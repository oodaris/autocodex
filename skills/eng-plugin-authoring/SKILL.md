---
name: eng-plugin-authoring
description: Build external Autocodex plugins with manifest + JSON-RPC.
version: 0.1.0
---

# Plugin Authoring

## Repo anchors (Autocodex)
- PLUGINS_PATH: `plugins/`
- DOCS_PATH: `docs/plugins/README.md`

## When to use
- Creating or updating an Autocodex plugin.

## Preconditions
- You know the capability inputs/outputs.

## Inputs to confirm
- Capability name and schemas
- Transport (jsonrpc for v1)

## Required artifacts
- `plugin.yaml` manifest
- Executable plugin binary/script
- Example invocation

## Quick path
- Write manifest.
- Implement handshake and JSON-RPC loop.
- Add a sample invocation.

## Steps
1) Define capability input/output schema.
2) Implement handshake and request loop.
3) Add usage docs.

## Failure modes and responses
- **Handshake missing**: plugin will fail to load.
- **Capability mismatch**: host rejects the request.

## Definition of done
- Plugin loads and responds to a capability call.

## Example (minimal)
- **Capability**: `summarize` with `{text:string}` input.
