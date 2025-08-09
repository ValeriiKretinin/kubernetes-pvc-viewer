# PVC Viewer

![App CI + Trivy](https://github.com/ValeriiKretinin/kubernetes-pvc-viewer/actions/workflows/app.yml/badge.svg)
![Helm CI + Trivy](https://github.com/ValeriiKretinin/kubernetes-pvc-viewer/actions/workflows/helm.yml/badge.svg)

Browse, download, upload (optional), and delete files on Kubernetes PersistentVolumeClaims with a modern, lightweight UI. Hot-reload configuration via ConfigMap, smart include/exclude matchers, and two data-plane modes:

- agent-per-pvc (recommended): one lightweight agent Pod per matched PVC, no restarts on config changes
- mount-in-backend: mount multiple PVCs directly into backend Pod (requires restart on changes)

The single container image embeds the React UI, backend API gateway/orchestrator, and agent binary.

## Features

- Hot-reload ConfigMap: update watched namespaces/PVCs/storageClasses without restarts (agent-per-pvc)
- Glob matchers (doublestar): include/exclude for namespaces, PVCs, storageClasses
- Agent (per-namespace or per-PVC): POSIX FS access; list/tree (pagination), range download with ETag, safe delete, upload (multipart), path traversal protection
- Security per storageClass: fsGroup/supplementalGroups overrides, readOnly mode
- Simple UI (React + Vite + Tailwind): sidebar (namespaces/PVCs), breadcrumbs, table/grid, preview skeleton, context menu, progress bar, error toasts
- Prometheus metrics endpoint (/metrics)
- Helm chart with RBAC, Service, Ingress (optional), NetworkPolicy, ConfigMap

## Architecture

```
Web UI (static)  <—HTTPS—>  Backend (Go)
                                 ├─ ConfigMap watcher (hot reload)
                                 ├─ Reconciler (creates/deletes agent Pods & headless Services)
                                 └─ Proxy to agents (HTTP)

                 Agents (Go) — one Pod per PVC (RWM)
                 └─ mount PVC at /data, file API (list/get/rm/upload)
```

## Modes

- agent-per-namespace (default): one agent per namespace, mounts all matched PVCs at `/data/<pvc>`
- agent-per-pvc: backend manages one lightweight agent per matched PVC (полный hot-reload без рестартов)
- mount-in-backend: backend Pod mounts multiple PVCs (defined in values), requires restart on changes

## Installation (OCI Helm chart)

Install directly from public OCI registry (no login required).

```
# Stable release
helm upgrade --install pvc-viewer \
  oci://ghcr.io/valeriikretinin/charts/pvc-viewer \
  --version 0.1.0 \
  -n pvc-viewer --create-namespace \
  --set image.repository=ghcr.io/valeriikretinin/kubernetes-pvc-viewer \
  --set image.tag=v0.1.0

```

Optionally enable Ingress in `values.yaml` (or via `--set`).

Note:
- The container image `ghcr.io/valeriikretinin/kubernetes-pvc-viewer` and chart `ghcr.io/valeriikretinin/charts/pvc-viewer` are published as Public packages. If you fork, make sure to set your GHCR packages visibility to Public to allow anonymous pulls.

See also the chart sources and detailed values in `helm/pvc-viewer/`.

## Configuration (ConfigMap)

Rendered to `/config/config.yaml` and hot-reloaded by backend.

```
watch:
  namespaces:
    include: ["*"]      # glob list; empty => match nothing
    exclude: ["kube-*"]
  pvcs:
    include: ["*"]
    exclude: []
  storageClasses:
    include: ["*"]
    exclude: []
mode:
  dataPlane: agent-per-pvc  # or mount-in-backend
agents:
  securityDefaults:
    runAsUser: 1000
    runAsGroup: 1000
    fsGroup: 1000
    supplementalGroups: [65534]
    readOnly: false
  securityOverrides:
    - match: "cephfs*"
      fsGroup: 16777216
      supplementalGroups: [16777216]
    - match: "nfs*"
      fsGroup: 1000
```

### mount-in-backend specifics

```
config:
  mode:
    dataPlane: mount-in-backend
  mountPVCs:
    - pvcName: my-shared-cephfs
      mountPath: /mnt/cephfs
      readOnly: false
      subPath: ""   # optional
```

On changes to `mountPVCs` backend Pod will restart (checksum/config) to re-mount volumes.

## API (backend)

- `GET /api/v1/namespaces`
- `GET /api/v1/pvcs?namespace=<ns>&storageClass=<glob?>`
- `GET /api/v1/tree?ns=<ns>&pvc=<pvc>&path=<path>&limit=200&offset=0`
- `GET /api/v1/download?ns=<ns>&pvc=<pvc>&path=<file>` (Range/ETag supported)
- `DELETE /api/v1/file?ns=<ns>&pvc=<pvc>&path=<file|dir>`
- `POST /api/v1/upload?ns=<ns>&pvc=<pvc>&path=<dir>` (multipart)
- `GET /api/v1/pvc-status?ns=<ns>&pvc=<pvc>`
- `GET /api/v1/healthz`, `GET /api/v1/readyz`, `GET /metrics`

## Security

- agent Pods: runAsNonRoot, readOnlyRootFilesystem, allowPrivilegeEscalation=false, drop ALL caps; fsGroup/supplementalGroups via overrides
- path validation in agent: normalized, no `..`, symlink containment inside `/data`
- NetworkPolicy template to restrict traffic
- AuthN/Z intentionally disabled by default in this repo; wire OIDC/JWT/Basic later if needed

## Metrics

Exposes `/metrics` (Prometheus). Add a ServiceMonitor if you use Prometheus Operator.

## Security & CI/CD

- Security scanning: Trivy image scan in CI; Trivy IaC scan for Helm chart.
- CI pipeline: Go build/test, UI build, Helm lint, Docker build/push (GHCR), Trivy scans.
- Actions: see badges above or visit the Actions tab.

## Development

Optional (for contributors):

- Prereqs: Go 1.24+, Node 20 (only for local UI dev)
- Dev UI: `cd ui && npm i && npm run dev` (proxies `/api` to backend on :8080)
- Docker build (UI+Go inside image): `docker build -t pvc-viewer:dev .`
- Local binaries: `go build ./cmd/backend && go build ./cmd/agent`

## Project layout

```
cmd/backend        # backend main + embedded static
cmd/agent          # agent main
internal/backend   # reconcile/proxy/metrics/status
internal/agent     # HTTP file API
internal/config    # YAML config + hot-reload
internal/matcher   # include/exclude glob matchers
internal/kube      # k8s client factory
internal/fsutil    # secure path utils
ui/                # React + Vite + Tailwind UI
helm/pvc-viewer    # Helm chart
```

## Roadmap

- Leader election for HA
- Full PVC status set (MountBlocked/ReadOnly) and richer UI badges
- Rate limiting & size limits (upload/download) + audit log
- OIDC/JWT AuthN/Z and RBAC mapping (patterns)
- EndpointSlice caching for agent proxy

## License

Apache-2.0