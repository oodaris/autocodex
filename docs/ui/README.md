# UI usage

## Embedded UI

Run the embedded production UI and API together:

```bash
autocodex ui
```

## Local development

1. Start the local API (via `autocodex run` or by running the API command if you prefer).
2. From `web/`, install dependencies and start Vite:

```bash
cd web
npm install
npm run dev
```

3. If your API is not on the default `http://127.0.0.1:7788`, set:

```bash
export VITE_API_BASE_URL="http://127.0.0.1:7788"
```

## What you can do

- **Runs**: See run status, phases, and artifacts.
- **Memory**: Review memory docs (`/memory`) that autocodex uses across loops.
- **Detail views**: Inspect the event stream for each run.
- **Terminal**: Start a live terminal session from the UI.

## Hub mode

Hub mode lets you monitor multiple repos from a single UI.

1) Enable hub mode in `autocodex.yaml`:

```yaml
hub:
  enabled: true
  workspaces:
    - id: repo-a
      name: Repo A
      root: /path/to/repo-a
      config_path: autocodex.yaml
```

If `hub.enabled` is true and no `hub.workspaces` are configured, the API will
auto-register the current repo as a single workspace.

2) Start the API server and open the UI route at `/hub`.
3) Click a workspace to view runs and memory docs for that repo.

## Terminal sessions

The terminal view opens a websocket-backed PTY session.

- Start a session from the UI.
- Use the input field to send commands.
- Close sessions when finished.
- If you leave Command empty, the session starts with `codex` by default.

If API auth is enabled, the terminal websocket uses `?token=` in the URL.

## Vercel deploy notes

Vercel serves a static build of the UI. The API must be reachable from the browser.

- Project root: `web/`
- Build command: `npm run build`
- Output directory: `dist`
- Environment variable: `VITE_API_BASE_URL` (point to a reachable autocodex API)

> Note: `127.0.0.1` only works for local development. Use a LAN host or a hosted API for remote access.

## Embedded UI build

The production UI is embedded into the autocodex binary. When building from source,
run the UI build first so `web/dist` exists:

```bash
npm ci --prefix web
npm run build --prefix web
```

## API auth

If `auth.enabled` is true in `autocodex.yaml`, you must provide an API token.

- Set `auth.token_env` to read a token from an env var (for example `AUTOCODEX_API_TOKEN`).
- UI: paste the token in the header input (stored in session storage).
- API calls: send `Authorization: Bearer <token>`.
