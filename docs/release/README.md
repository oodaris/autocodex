# Release process

This repo uses GoReleaser to package binaries.

## Smoke test a release

```bash
goreleaser release --snapshot --clean -f goreleaser.yml
```

## Publish a release

1) Update `CHANGELOG.md`.
2) Tag a version:
```bash
git tag -a v0.2.0 -m "v0.2.0"
git push origin v0.2.0
```
3) Run GoReleaser:
```bash
goreleaser release -f goreleaser.yml
```

## Release notes template

Use `docs/release/RELEASE_NOTES_TEMPLATE.md`.
