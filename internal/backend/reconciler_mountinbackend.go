package backend

// Placeholder for mode "mount-in-backend": it requires backend Pod to mount multiple PVCs.
// This mode is wired via Helm values (mode.dataPlane) and would be implemented by rendering
// volumes/volumeMounts into the backend Deployment, then triggering rollout on config changes.
