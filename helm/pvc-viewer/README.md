# pvc-viewer Helm Chart

Deploys PVC Viewer backend (serves UI and API). Agents are created dynamically by the backend based on configuration.

## Installation

From public OCI registry (no login required):

```
helm upgrade --install pvc-viewer \
  oci://ghcr.io/valeriikretinin/charts/pvc-viewer \
  --version 0.1.0 \
  -n pvc-viewer --create-namespace \
  --set image.repository=ghcr.io/valeriikretinin/kubernetes-pvc-viewer \
  --set image.tag=v0.1.0
```

## Values overview

- `image.repository`, `image.tag` — container image for backend and agents (same image)
- `config.watch.*` — glob include/exclude for namespaces, pvcs, storageClasses. Empty include means “match nothing”.
- `config.allowRWO` — when false (default) skip PVC without ReadWriteMany (RWO is shown as MountBlocked in UI); when true allow RWO.
- `config.mode.dataPlane` — data-plane mode:
  - `agent-per-namespace` (default): one agent per namespace, mounts all matched PVCs at `/data/<pvc>`; adding/removing PVC requires recreating that agent Pod.
  - `agent-per-pvc`: one lightweight agent Pod per matched PVC; no restarts on changes (recommended for full hot-reload experience).
  - `mount-in-backend`: mount listed PVCs into the backend Pod via `config.mountPVCs` (requires Pod restart on changes).
- `config.mountPVCs[]` — list of PVCs to mount when `mount-in-backend` is selected.
- `config.agents.securityDefaults` / `securityOverrides` — pod security (fsGroup/supplementalGroups per storageClass), and readOnly flag.
- `ingress.*` — optional Ingress config.
- `resources.backend` — backend Pod resources.

## How agents are managed

The backend watches the cluster according to `config.watch.*` and reconciles desired agents:
- For `agent-per-namespace`: it creates one Service+Pod per namespace (`pvc-viewer-agent-ns-<hash>`), with multiple volume mounts `/data/<pvc>`.
- For `agent-per-pvc`: it creates one Service+Pod per PVC (`pvc-viewer-agent-<hash>`), each mounting its PVC at `/data`.

Agents expose internal HTTP endpoints used by backend proxy:
- `GET /v1/tree?path=...`
- `GET /v1/file?path=...` (Range/ETag)
- `DELETE /v1/file?path=...`
- `POST /v1/upload?path=/dir` (multipart)

## Security

- Backend and agents run as non-root (`runAsUser: 65532`, distroless), readOnly root fs, no privilege escalation, drop ALL capabilities.
- Path traversal protection in agent, symlink containment under `/data`.
- NetworkPolicy template provided to limit traffic.

## Notes

- UI is embedded into backend image and served from `/`.
- Prometheus metrics available on `/metrics`.
- For production, set specific `watch` patterns and enable Ingress/TLS as needed.
