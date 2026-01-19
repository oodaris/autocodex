---
name: eng-ui-vite-react
description: Build autocodex UI with React + Vite.
version: 0.2.0
---

# UI: React + Vite

## Repo anchors (autocodex)
- UI_PATH: `web/`
- API_CONTRACT: `docs/contracts/local-api.openapi.yaml`

## When to use
- Implementing or updating the UI.

## Preconditions
- API contract exists.

## Inputs to confirm
- Target routes and data models
- Performance constraints

## Required artifacts
- Typed components
- API client module
- Build passes

## Quick path
- Scaffold components.
- Wire API client.
- Run `npm run build`.

## Steps
1) Define routes and data loading.
2) Implement components with typed props.
3) Add a basic build check.

## Failure modes and responses
- **No API contract**: stop and define the contract first.

## Definition of done
- UI builds and renders core views.

## Example (minimal)
- **View**: runs list reading from `/runs`.
