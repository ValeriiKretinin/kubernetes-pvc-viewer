package auth

import (
	"net/http"
	"strings"
)

// Simple RBAC based on allowlists of namespace/PVC glob patterns
type Rule struct {
	Namespaces []string
	PVCs       []string
}

type RBAC struct{ Rules []Rule }

type Matcher interface{ Match(string) bool }

func (r *RBAC) Allowed(ns, pvc string, nsMatch func([]string) bool, pvcMatch func([]string) bool) bool {
	if len(r.Rules) == 0 {
		return true
	}
	for _, rule := range r.Rules {
		if nsMatch(rule.Namespaces) && pvcMatch(rule.PVCs) {
			return true
		}
	}
	return false
}

// Middleware extracts Bearer or Basic (username) and attaches subject to context (simplified)
func Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authz := r.Header.Get("Authorization")
		if authz == "" {
			next.ServeHTTP(w, r)
			return
		}
		_ = strings.TrimSpace(authz) // placeholder; in real impl validate token
		next.ServeHTTP(w, r)
	})
}

