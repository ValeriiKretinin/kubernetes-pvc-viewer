# Contributing

Thanks for your interest in contributing!

## How to contribute

- Fork the repo and create a feature branch
- Keep changes focused and small; open separate PRs for unrelated changes
- Ensure `go build` and UI build succeed; run linters if available
- Add/update tests where reasonable
- Update docs/README/Helm values if behavior changes

## Development setup

- Go 1.24+
- Node 20+ (for UI dev)
- Build UI: `cd ui && npm i && npm run build`
- Build binaries: `go build ./cmd/backend && go build ./cmd/agent`
- Docker image (UI+Go inside): `docker build -t pvc-viewer:dev .`

## Pull requests

- Fill in a clear description and motivation
- Link related issues if any
- Keep commits clean and rebased on latest `main`
- CI must pass before merge

## Code style

- Go: prefer clarity over cleverness, handle errors explicitly
- TS/React: keep components small, avoid unnecessary state, handle errors in UI

## Security

- Do not include secrets or private data in issues/PRs
- Report vulnerabilities via the Security policy (see `SECURITY.md`)
