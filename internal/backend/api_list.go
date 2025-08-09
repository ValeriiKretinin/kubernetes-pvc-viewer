package backend

import (
	"encoding/json"
	"net/http"
	"sort"

	"k8s.io/client-go/kubernetes"

	"github.com/valeriikretinin/kubernetes-pvc-viewer/internal/config"
)

// RegisterReadAPIs wires list endpoints into the router.
// It filters namespaces/PVCs using the same matching logic as reconciliation,
// so the UI only shows items that the data plane is actually serving.
func RegisterReadAPIs(mux interface {
	Get(string, http.HandlerFunc)
}, client kubernetes.Interface, cfgProvider func() *config.Config) {
	mux.Get("/namespaces", func(w http.ResponseWriter, r *http.Request) {
		cfg := cfgProvider()
		// Build eligible targets and return unique namespaces
		d := &Discovery{Client: client}
		targets, err := d.BuildTargets(r.Context(), cfg)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		set := map[string]struct{}{}
		for _, t := range targets {
			set[t.Namespace] = struct{}{}
		}
		names := make([]string, 0, len(set))
		for n := range set {
			names = append(names, n)
		}
		sort.Strings(names)
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(names)
	})
	mux.Get("/pvcs", func(w http.ResponseWriter, r *http.Request) {
		ns := r.URL.Query().Get("namespace")
		if ns == "" {
			http.Error(w, "namespace required", http.StatusBadRequest)
			return
		}
		cfg := cfgProvider()
		d := &Discovery{Client: client}
		targets, err := d.BuildTargets(r.Context(), cfg)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		names := make([]string, 0)
		for _, t := range targets {
			if t.Namespace == ns {
				names = append(names, t.PVCName)
			}
		}
		sort.Strings(names)
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(names)
	})
}
