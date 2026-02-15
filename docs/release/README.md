# Release process

This repo uses GoReleaser to package binaries.

## Smoke test a release

```bash
goreleaser release --snapshot --clean -f goreleaser.yml
```

Release archives now include prebuilt plugins under `plugins/`.

## Publish a release

1) Sync Beads (sync-branch mode: `bd sync` commits/pushes beads data to `beads-sync`):
```bash
bd sync
```
2) Update `CHANGELOG.md`.
3) Smoke test a snapshot build:
```bash
goreleaser release --snapshot --clean -f goreleaser.yml
```
4) Tag a version:
```bash
git tag -a v0.2.0 -m "v0.2.0"
git push origin v0.2.0
```
5) Run GoReleaser:
```bash
goreleaser release -f goreleaser.yml
```

## macOS signing + notarization (public releases)

For public releases, sign and notarize the macOS binaries before distribution.
This repo includes a helper script that signs a darwin binary and submits it
for notarization via Apple’s notarytool.

Prereqs:
- Apple Developer ID certificate installed locally.
- App-specific password for notarization.

Environment variables:
- `MACOS_SIGN_IDENTITY` (Developer ID Application: ...)
- `APPLE_ID`
- `APPLE_TEAM_ID`
- `APPLE_APP_PASSWORD`

Example (after `goreleaser release --skip-publish` or `goreleaser build`):
```bash
./scripts/notarize_macos.sh dist/autocodex_darwin_arm64_v1/autocodex
./scripts/notarize_macos.sh dist/autocodex_darwin_amd64_v1/autocodex
```

Then publish the release assets using your standard release flow.

## Release notes template

Use `docs/release/RELEASE_NOTES_TEMPLATE.md`.
