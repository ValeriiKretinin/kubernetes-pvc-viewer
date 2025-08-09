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

// NamespaceAgentGroupName returns deterministic name for a namespace agent bound to a security profile
func NamespaceAgentGroupName(ns, profileHash string) string {
	// profileHash is expected to be short hex (8-12 chars)
	if profileHash == "" {
		return NamespaceAgentName(ns)
	}
	base := sha1.Sum([]byte(ns))
	return "pvc-viewer-agent-ns-" + hex.EncodeToString(base[:4]) + "-" + profileHash
}
