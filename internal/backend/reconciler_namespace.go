package backend

import (
	"context"
	"crypto/sha1"
	"encoding/hex"
	"sort"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"

	"github.com/bmatcuk/doublestar/v4"
	"github.com/valeriikretinin/kubernetes-pvc-viewer/internal/config"
)

// EnsureNamespaceAgent groups PVCs by effective security profile and ensures one agent Pod/Service per group.
func (r *Reconciler) EnsureNamespaceAgent(ctx context.Context, namespace string, pvcNames []string) error {
	if len(pvcNames) == 0 {
		return nil
	}

	// Best-effort cleanup of legacy single-agent resources
	legacy := NamespaceAgentName(namespace)
	_ = r.Client.CoreV1().Pods(namespace).Delete(ctx, legacy, metav1.DeleteOptions{})
	_ = r.Client.CoreV1().Services(namespace).Delete(ctx, legacy, metav1.DeleteOptions{})

	type group struct {
		pvcs []string
		sec  config.SecuritySpec
	}
	groups := map[string]*group{}

	// Build groups by security profile
	for _, pvc := range pvcNames {
		sc := ""
		if p, err := r.Client.CoreV1().PersistentVolumeClaims(namespace).Get(ctx, pvc, metav1.GetOptions{}); err == nil {
			if p.Spec.StorageClassName != nil {
				sc = *p.Spec.StorageClassName
			} else if p.Spec.VolumeName != "" {
				if pv, err2 := r.Client.CoreV1().PersistentVolumes().Get(ctx, p.Spec.VolumeName, metav1.GetOptions{}); err2 == nil {
					sc = pv.Spec.StorageClassName
				}
			}
		}
		// compute effective security from defaults + first matching override (glob)
		eff := r.Defaults
		for _, ov := range r.Overrides {
			if ok, _ := doublestar.Match(ov.Match, sc); ok {
				eff = mergeSecurity(eff, ov.SecuritySpec)
				break
			}
		}
		key := ProfileKey(eff)
		if _, ok := groups[key]; !ok {
			groups[key] = &group{pvcs: []string{}, sec: eff}
		}
		groups[key].pvcs = append(groups[key].pvcs, pvc)
	}

	desired := map[string]struct{}{}
	// Ensure each group
	for key, g := range groups {
		sort.Strings(g.pvcs)
		name := NamespaceAgentGroupName(namespace, key)
		desired[name] = struct{}{}

		labels := map[string]string{
			"app":                 "pvc-viewer-agent-ns",
			"pvcviewer.k8s.io/ns": namespace,
			"pvcviewer.k8s.io/gr": key,
		}

		volumes := make([]corev1.Volume, 0, len(g.pvcs))
		mounts := make([]corev1.VolumeMount, 0, len(g.pvcs))
		for _, pvc := range g.pvcs {
			vname := "v-" + pvc
			volumes = append(volumes, corev1.Volume{
				Name:         vname,
				VolumeSource: corev1.VolumeSource{PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{ClaimName: pvc}},
			})
			mounts = append(mounts, corev1.VolumeMount{Name: vname, MountPath: "/data/" + pvc, ReadOnly: g.sec.ReadOnly})
		}

		// Spec hash to detect PVC set changes
		h := sha1.Sum([]byte(stringsJoin(g.pvcs, ",")))
		desiredHash := hex.EncodeToString(h[:8])

		// Service
		svc := &corev1.Service{ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: namespace, Labels: labels}, Spec: corev1.ServiceSpec{
			Selector: labels,
			Ports:    []corev1.ServicePort{{Name: "http", Port: 8090, TargetPort: intstr.FromInt(8090)}},
		}}
		if _, err := r.Client.CoreV1().Services(namespace).Get(ctx, name, metav1.GetOptions{}); err != nil {
			_, _ = r.Client.CoreV1().Services(namespace).Create(ctx, svc, metav1.CreateOptions{})
		}

		// Pod
		recreate := true
		if existing, err := r.Client.CoreV1().Pods(namespace).Get(ctx, name, metav1.GetOptions{}); err == nil {
			if existing.Annotations != nil && existing.Annotations["pvcviewer.k8s.io/spec-hash"] == desiredHash {
				recreate = false
			} else {
				_ = r.Client.CoreV1().Pods(namespace).Delete(ctx, name, metav1.DeleteOptions{})
			}
		}
		if recreate {
			ann := map[string]string{"pvcviewer.k8s.io/spec-hash": desiredHash}
			sec := g.sec
			pod := &corev1.Pod{ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: namespace, Labels: labels, Annotations: ann}, Spec: corev1.PodSpec{
				Containers: []corev1.Container{{
					Name:         "agent",
					Image:        r.AgentImage,
					Command:      []string{"/bin/agent"},
					Env:          []corev1.EnvVar{{Name: "PVC_VIEWER_DATA_ROOT", Value: "/data"}, {Name: "PVC_VIEWER_READ_ONLY", Value: boolString(sec.ReadOnly)}},
					Ports:        []corev1.ContainerPort{{ContainerPort: 8090}},
					VolumeMounts: mounts,
					SecurityContext: &corev1.SecurityContext{
						RunAsNonRoot:             boolPtr(true),
						RunAsUser:                pickInt(sec.RunAsUser, 65532),
						RunAsGroup:               pickInt(sec.RunAsGroup, 65532),
						AllowPrivilegeEscalation: boolPtr(false),
						Capabilities:             &corev1.Capabilities{Drop: []corev1.Capability{"ALL"}},
					},
				}},
				Volumes: volumes,
				SecurityContext: &corev1.PodSecurityContext{
					RunAsUser:          pickInt(sec.RunAsUser, 65532),
					RunAsGroup:         pickInt(sec.RunAsGroup, 65532),
					FSGroup:            sec.FSGroup,
					SupplementalGroups: mergeSupplemental(r.Defaults.SupplementalGroups, sec.SupplementalGroups),
				},
			}}
			if created, err := r.Client.CoreV1().Pods(namespace).Create(ctx, pod, metav1.CreateOptions{}); err == nil {
				if r.Logger != nil {
					r.Logger.Infow("ns agent group ensured", "namespace", namespace, "name", created.Name, "pvcs", g.pvcs)
				}
			}
		}
	}

	// GC pods/services not in desired
	podList, _ := r.Client.CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{LabelSelector: "app=pvc-viewer-agent-ns"})
	for _, p := range podList.Items {
		if _, ok := desired[p.Name]; !ok {
			_ = r.Client.CoreV1().Pods(namespace).Delete(ctx, p.Name, metav1.DeleteOptions{})
			_ = r.Client.CoreV1().Services(namespace).Delete(ctx, p.Name, metav1.DeleteOptions{})
		}
	}
	return nil
}

func stringsJoin(in []string, sep string) string {
	if len(in) == 0 {
		return ""
	}
	out := in[0]
	for i := 1; i < len(in); i++ {
		out += sep + in[i]
	}
	return out
}

// GCPerPVCAll deletes all per-PVC agents and their services
func (r *Reconciler) GCPerPVCAll(ctx context.Context) error {
	// pods with app=pvc-viewer-agent
	pods, err := r.Client.CoreV1().Pods("").List(ctx, metav1.ListOptions{LabelSelector: "app=pvc-viewer-agent"})
	if err != nil {
		return err
	}
	for _, p := range pods.Items {
		ns := p.Namespace
		name := p.Name
		_ = r.Client.CoreV1().Pods(ns).Delete(ctx, name, metav1.DeleteOptions{})
		_ = r.Client.CoreV1().Services(ns).Delete(ctx, name, metav1.DeleteOptions{})
	}
	return nil
}

// GCNamespaceAgents deletes namespace agents not in keep set
func (r *Reconciler) GCNamespaceAgents(ctx context.Context, keepNamespaces map[string]struct{}) error {
	pods, err := r.Client.CoreV1().Pods("").List(ctx, metav1.ListOptions{LabelSelector: "app=pvc-viewer-agent-ns"})
	if err != nil {
		return err
	}
	for _, p := range pods.Items {
		if _, ok := keepNamespaces[p.Namespace]; ok {
			continue
		}
		ns := p.Namespace
		name := p.Name
		_ = r.Client.CoreV1().Pods(ns).Delete(ctx, name, metav1.DeleteOptions{})
		_ = r.Client.CoreV1().Services(ns).Delete(ctx, name, metav1.DeleteOptions{})
	}
	return nil
}
