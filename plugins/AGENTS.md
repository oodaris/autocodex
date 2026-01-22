# Plugins Subproject

This folder contains autocodex plugins (external processes) and their manifests.

## Rules
- Plugins must have a `plugin.yaml` or `plugin.json` manifest.
- JSON-RPC (stdio) is the v1 transport.
- Plugins must emit a single-line JSON handshake before serving requests.
- Keep plugins language-agnostic and portable.

## Commands
- Build example plugin: `go build -o plugins/example-summarizer/example-summarizer ./plugins/example-summarizer`

## Safety
- Plugins are untrusted; never execute arbitrary input without validation.
