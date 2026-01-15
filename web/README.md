# autocodex Web Dashboard

A lightweight React/Vite dashboard for the local autocodex API. This UI is read-only in v1 and focuses on run visibility.

## Requirements
- Node.js 18+

## Setup
```bash
npm install
```

## Development
```bash
npm run dev
```

By default the UI expects the API at `http://127.0.0.1:7788`. Override with:

```bash
VITE_API_BASE_URL=http://127.0.0.1:7788 npm run dev
```

## Build
```bash
npm run build
```

## Accessibility + bundle checks
```bash
npm run a11y
npm run bundlesize
```

Notes:
- `npm run a11y` runs Playwright + axe on the core pages.
- `npm run bundlesize` checks raw JS/CSS output sizes in `dist/` (run `npm run build` first).

## Notes
- The UI is intentionally read-only for v1.
- Run the API server via `autocodex api --action serve`.
