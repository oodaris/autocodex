---
name: eng-plugin-authoring
description: Build external Autocodex plugins with manifest + JSON-RPC.
version: 0.1.0
---

# Plugin Authoring

## Requirements
- Provide `plugin.yaml` or `plugin.json`.
- Implement JSON handshake on stdout.
- Serve JSON-RPC requests over stdio.

## Steps
1) Define capability input/output schema.
2) Implement handshake and request loop.
3) Add a sample invocation in README.
4) Add a simple test if possible.
