# UI usage

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
- **Memory**: Review memory docs (`/memory`) that Autocodex uses across loops.
- **Detail views**: Inspect the event stream for each run.

## Vercel deploy notes

Vercel serves a static build of the UI. The API must be reachable from the browser.

- Project root: `web/`
- Build command: `npm run build`
- Output directory: `dist`
- Environment variable: `VITE_API_BASE_URL` (point to a reachable Autocodex API)

> Note: `127.0.0.1` only works for local development. Use a LAN host or a hosted API for remote access.
