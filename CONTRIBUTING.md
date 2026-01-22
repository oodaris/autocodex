# Contributing to autocodex

Thanks for contributing!

## Quickstart
1) Fork and clone.
2) Install Go.
3) Run tests: `go test ./...`

## Workflow
- Use Beads for tasks (`bd create`, `bd dep add`).
- Follow the golden workflow: Plan → Contracts → Code → Tests → Docs → Rollout.
- Keep changes small and scoped.

## Plugin catalog contributions
Add new plugins to the catalog by following the conventions in `docs/plugins/README.md`.

Checklist:
1) Create `plugins/<name>/` with:
   - `plugin.yaml`
   - `main.go` (JSON‑RPC over stdio)
   - `schemas/<capability>.input.json` + `schemas/<capability>.output.json`
2) Build + run locally (use the sample plugin as a reference).
3) Update the catalog table in `docs/plugins/README.md` and `README.md`.
4) Keep logs on **stderr** (stdout is reserved for protocol messages).
5) Run tests: `go test ./...`

## Coding standards
- Go: run `gofmt -w $(rg --files -g '*.go')`
- Tests: add tests for behavior changes.
- Logging: emit JSON logs with required fields.

## Commit style
Use conventional commits:
```
feat(scope): short subject
```

## Pull requests
- Include tests and docs if behavior changed.
- Update AGENTS.md if new workflows are added.
