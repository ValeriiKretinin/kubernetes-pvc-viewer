package backend

import (
	"context"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	"github.com/valeriikretinin/kubernetes-pvc-viewer/internal/config"
	"github.com/valeriikretinin/kubernetes-pvc-viewer/internal/matcher"
)

type Discovery struct {
	Client kubernetes.Interface
}

// BuildTargets lists PVCs cluster-wide and applies matchers from cfg. If include lists are empty, returns empty.
func (d *Discovery) BuildTargets(ctx context.Context, cfg *config.Config) ([]Target, error) {
	// log inputs to aid troubleshooting
	_ = cfg
	nsMatch := matcher.New(cfg.Watch.Namespaces.Include, cfg.Watch.Namespaces.Exclude)
	pvcMatch := matcher.New(cfg.Watch.Pvcs.Include, cfg.Watch.Pvcs.Exclude)
	scMatch := matcher.New(cfg.Watch.StorageClasses.Include, cfg.Watch.StorageClasses.Exclude)

	// If any include is empty -> treat as nothing per spec
	if len(cfg.Watch.Namespaces.Include) == 0 || len(cfg.Watch.Pvcs.Include) == 0 || len(cfg.Watch.StorageClasses.Include) == 0 {
		return []Target{}, nil
	}

	nsl, err := d.Client.CoreV1().Namespaces().List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	out := []Target{}
	for _, ns := range nsl.Items {
		if !nsMatch.Match(ns.Name) {
			continue
		}
		pvcs, err := d.Client.CoreV1().PersistentVolumeClaims(ns.Name).List(ctx, metav1.ListOptions{})
		if err != nil {
			return nil, err
		}
		for _, pvc := range pvcs.Items {
			if !pvcMatch.Match(pvc.Name) {
				continue
			}
			if !cfg.AllowRWO && !hasRWM(pvc) {
				continue
			}
			sc := ""
			if pvc.Spec.StorageClassName != nil {
				sc = *pvc.Spec.StorageClassName
			}
			// Fallback: some PVCs have nil StorageClassName; resolve from bound PV if possible
			if sc == "" && pvc.Spec.VolumeName != "" {
				if pv, err := d.Client.CoreV1().PersistentVolumes().Get(ctx, pvc.Spec.VolumeName, metav1.GetOptions{}); err == nil {
					sc = pv.Spec.StorageClassName
				}
			}
			if sc == "" || !scMatch.Match(sc) {
				continue
			}
			out = append(out, Target{Namespace: ns.Name, PVCName: pvc.Name, StorageClass: sc})
		}
	}
	return out, nil
}

func hasRWM(p corev1.PersistentVolumeClaim) bool {
	for _, m := range p.Spec.AccessModes {
		if m == corev1.ReadWriteMany || m == corev1.ReadWriteOncePod { // allow RWO-Pod too
			return true
		}
	}
	return false
}
