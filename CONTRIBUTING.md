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
