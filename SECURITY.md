# Security Policy

We take security seriously and appreciate responsible disclosure.

## Supported versions

We actively maintain the latest main branch and released images. Security fixes are backported on a best‑effort basis.

## Reporting a vulnerability

- Open a private security advisory on GitHub or contact the maintainer.
- Please include:
  - A description of the issue and impact
  - Steps to reproduce or a proof of concept
  - Affected versions (image tag/commit)
  - Environment details (Kubernetes version, storage class)
- We will acknowledge within 72 hours and provide a remediation plan and timeline.

## CI security checks

- Container image and Helm IaC are scanned with Trivy in GitHub Actions on every PR and push to main.
- Builds fail on HIGH/CRITICAL issues by policy; low/medium are reported but may not block.
- Dependency updates are checked regularly by CI; consider pinning base image digests in production.

## Data plane hardening

- Backend and agents run as non‑root, read‑only root filesystem, no privilege escalation, drop ALL Linux capabilities.
- File access is limited to mounted PVC paths and protected against path traversal using secure join and symlink containment.
- AccessModes: only RWX (ReadWriteMany) PVCs are supported in multi‑pod scenarios.
- Network exposure: backend exposes UI/API; agents are internal ClusterIP services (proxied by backend). NetworkPolicies recommended.

## Configuration hardening

- Use `securityOverrides` per storage class to set `fsGroup`, `supplementalGroups`, `runAsUser/runAsGroup`, and read‑only when appropriate.
- Restrict watched namespaces/PVCs/storageClasses via include/exclude globs.
- Deploy behind authenticated ingress or service mesh in multi‑tenant clusters.

## Dependency and image security

- Images are scanned with Trivy in CI and should be scanned in your registry on push.
- Base image: distroless static nonroot; no shell/package manager.

## Incident response

- For confirmed vulnerabilities:
  - We will publish a security advisory (CVE if applicable)
  - Release a patched version
  - Provide mitigation guidance
