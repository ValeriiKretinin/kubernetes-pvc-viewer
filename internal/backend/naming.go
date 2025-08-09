package backend

import (
	"crypto/sha1"
	"encoding/hex"
)

// NamespaceAgentName returns a deterministic name for a namespace-scoped agent
func NamespaceAgentName(ns string) string {
	h := sha1.Sum([]byte(ns))
	return "pvc-viewer-agent-ns-" + hex.EncodeToString(h[:8])
}
