package backend

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/bmatcuk/doublestar/v4"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// RegisterReadAPIs wires list endpoints into the router.
func RegisterReadAPIs(mux interface {
	Get(string, http.HandlerFunc)
}, client kubernetes.Interface) {
	mux.Get("/namespaces", func(w http.ResponseWriter, r *http.Request) {
		ns, err := client.CoreV1().Namespaces().List(r.Context(), metav1.ListOptions{})
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		names := make([]string, 0, len(ns.Items))
		for _, n := range ns.Items {
			names = append(names, n.Name)
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(names)
	})
	mux.Get("/pvcs", func(w http.ResponseWriter, r *http.Request) {
		ns := r.URL.Query().Get("namespace")
		if ns == "" {
			http.Error(w, "namespace required", http.StatusBadRequest)
			return
		}
		pvcs, err := client.CoreV1().PersistentVolumeClaims(ns).List(context.Background(), metav1.ListOptions{})
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		scGlob := r.URL.Query().Get("storageClass")
		names := make([]string, 0, len(pvcs.Items))
		for _, p := range pvcs.Items {
			if scGlob != "" {
				sc := ""
				if p.Spec.StorageClassName != nil {
					sc = *p.Spec.StorageClassName
				}
				if ok, _ := doublestar.Match(scGlob, sc); !ok {
					continue
				}
			}
			names = append(names, p.Name)
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(names)
	})
}
