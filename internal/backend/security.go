package backend

import (
	corev1 "k8s.io/api/core/v1"

	"github.com/bmatcuk/doublestar/v4"
)

// buildPodSecurityContext builds Pod-level security context using defaults and any matching overrides by storage class.
func buildPodSecurityContext(r *Reconciler, t Target) *corev1.PodSecurityContext {
	spec := r.Defaults
	// apply first matching override by StorageClass
	for _, ov := range r.Overrides {
		if ok, _ := doublestar.Match(ov.Match, t.StorageClass); ok {
			if ov.RunAsUser != nil {
				spec.RunAsUser = ov.RunAsUser
			}
			if ov.RunAsGroup != nil {
				spec.RunAsGroup = ov.RunAsGroup
			}
			if ov.FSGroup != nil {
				spec.FSGroup = ov.FSGroup
			}
			if len(ov.SupplementalGroups) > 0 {
				spec.SupplementalGroups = ov.SupplementalGroups
			}
			spec.ReadOnly = ov.ReadOnly
			break
		}
	}

	psc := &corev1.PodSecurityContext{}
	if spec.RunAsUser != nil {
		psc.RunAsUser = spec.RunAsUser
	}
	if spec.RunAsGroup != nil {
		psc.RunAsGroup = spec.RunAsGroup
	}
	if spec.FSGroup != nil {
		psc.FSGroup = spec.FSGroup
	}
	if len(spec.SupplementalGroups) > 0 {
		psc.SupplementalGroups = spec.SupplementalGroups
	}
	return psc
}
