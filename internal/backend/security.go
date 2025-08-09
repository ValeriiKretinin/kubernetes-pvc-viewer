package backend

import (
	"crypto/sha1"
	"encoding/hex"
	"sort"
	"strconv"

	corev1 "k8s.io/api/core/v1"

	"github.com/bmatcuk/doublestar/v4"

	"github.com/valeriikretinin/kubernetes-pvc-viewer/internal/config"
)

// buildPodSecurityContext builds Pod-level security context using defaults and any matching overrides by storage class.
func buildPodSecurityContext(r *Reconciler, t Target) *corev1.PodSecurityContext {
	spec := r.Defaults
	// apply first matching override by StorageClass
	for _, ov := range r.Overrides {
		if ok, _ := doublestar.Match(ov.Match, t.StorageClass); ok {
			spec = mergeSecurity(spec, ov.SecuritySpec)
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

// BuildSecuritySpec computes effective security spec for given storageClass using config (no kube client needed)
func BuildSecuritySpec(cfg *config.Config, pvcName string, storageClass string) config.SecuritySpec {
	out := cfg.Agents.SecurityDefaults
	for _, o := range cfg.Agents.SecurityOverrides {
		// pvcMatch takes precedence if provided
		if o.PvcMatch != "" {
			if ok, _ := doublestar.Match(o.PvcMatch, pvcName); ok {
				out = mergeSecurity(out, o.SecuritySpec)
				break
			}
			continue
		}
		if ok, _ := doublestar.Match(o.Match, storageClass); ok {
			out = mergeSecurity(out, o.SecuritySpec)
			break
		}
	}
	return out
}

// ProfileKey returns stable short key for a security spec used to derive group hash
func ProfileKey(s config.SecuritySpec) string {
	ru, rg, fg := int64(0), int64(0), int64(0)
	if s.RunAsUser != nil {
		ru = *s.RunAsUser
	}
	if s.RunAsGroup != nil {
		rg = *s.RunAsGroup
	}
	if s.FSGroup != nil {
		fg = *s.FSGroup
	}
	supp := append([]int64{}, s.SupplementalGroups...)
	sort.Slice(supp, func(i, j int) bool { return supp[i] < supp[j] })
	buf := strconv.FormatInt(ru, 10) + "|" + strconv.FormatInt(rg, 10) + "|" + strconv.FormatInt(fg, 10) + "|" + strconv.FormatBool(s.ReadOnly) + "|"
	for _, g := range supp {
		buf += strconv.FormatInt(g, 10) + ","
	}
	sum := sha1.Sum([]byte(buf))
	return hex.EncodeToString(sum[:8])
}

// mergeSecurity merges override into base
func mergeSecurity(base config.SecuritySpec, o config.SecuritySpec) config.SecuritySpec {
	if o.RunAsUser != nil {
		base.RunAsUser = o.RunAsUser
	}
	if o.RunAsGroup != nil {
		base.RunAsGroup = o.RunAsGroup
	}
	if o.FSGroup != nil {
		base.FSGroup = o.FSGroup
	}
	if len(o.SupplementalGroups) > 0 {
		base.SupplementalGroups = o.SupplementalGroups
	}
	if o.ReadOnly {
		base.ReadOnly = true
	}
	return base
}
