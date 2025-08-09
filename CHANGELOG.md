# Changelog

All notable changes to this project will be documented in this file.

## Unreleased

- UI: redesigned layout — header with namespace/PVC selectors and search, left folder tree, right file table, inline previews (text/images/PDF), iconized actions
- UI: dark theme contrast improvements, unified buttons, progress bar and error toasts
- Agent: uid/gid/mode in directory listings; new endpoint POST /v1/empty to clear directory contents
- Backend: POST /api/v1/empty-dir -> agent /v1/empty, service-DNS proxying
- Backend: agent-per-namespace now runs multiple agents per namespace grouped by effective security profile; routing picks the correct Service per PVC
- Discovery: resolve storageClass from PV when missing in PVC; RWX-only
- Reconciler: GC of stale per-profile ns agents; logging for ensure
- GC: manual endpoint /api/v1/gc and shutdown GC with reconciliation disabled
- Helm: minor docs and values clarifications; RBAC improvements

## 0.1.0

- Initial public release
