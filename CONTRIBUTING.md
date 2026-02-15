# Contributing to autocodex

Thanks for contributing!

## Quickstart
1) Fork and clone.
2) Install Go 1.26+.
3) Run tests: `go test ./...`

## Requirements
- Go 1.26+ (build + tests)
- Node.js + npm (UI work and release packaging)
- Beads (`bd`) (recommended for this repo's task tracking)

## Workflow
- Use Beads for tasks (`bd create`, `bd dep add`).
- Follow the golden workflow: Plan → Contracts → Code → Tests → Docs → Rollout.
- Keep changes small and scoped.

## Verification (maintainers)
Run the full gate suite before cutting a release:

```bash
# Go
go test ./...
go vet ./...
staticcheck ./...
govulncheck ./...

# UI
cd web
npm ci
npm run lint
npm test
npm run build
cd ..

# Release packaging (builds UI + runs go mod tidy automatically)
goreleaser release --snapshot --clean -f goreleaser.yml
```

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
