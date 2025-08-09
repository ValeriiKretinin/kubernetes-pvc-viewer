# Changelog

All notable changes to this project will be documented in this file.

## Unreleased

- UI: dark theme contrast improvements
- UI: PVC list caching per namespace and loading indicator
- UI: actions for files (download/delete) and directories (delete/upload)
- Agent: uid/gid/mode in directory listings
- Agent: new endpoint POST /v1/empty to clear directory contents
- Backend: proxy route POST /api/v1/empty-dir -> agent /v1/empty
- Discovery: resolve storageClass from PV when missing in PVC; RWX-only
- Proxy: use service DNS instead of Endpoints IP
- Reconciler: recreate ns-agent on PVC set changes; logging for ensure
- GC: manual endpoint /api/v1/gc and shutdown GC with reconciliation disabled
- Helm: fixed Deployment YAML indentation; RBAC improvements

## 0.1.0

- Initial public release
