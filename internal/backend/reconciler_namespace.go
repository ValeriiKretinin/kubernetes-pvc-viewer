package backend

import (
	"context"
	"crypto/sha1"
	"encoding/hex"
	"sort"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

// EnsureNamespaceAgent creates/updates a single agent Pod per namespace with multiple PVC mounts
func (r *Reconciler) EnsureNamespaceAgent(ctx context.Context, namespace string, pvcNames []string) error {
	if len(pvcNames) == 0 {
		return nil
	}
	name := NamespaceAgentName(namespace)
	labels := map[string]string{
		"app":                 "pvc-viewer-agent-ns",
		"pvcviewer.k8s.io/ns": namespace,
	}
	// Build volumes and mounts
	volumes := make([]corev1.Volume, 0, len(pvcNames))
	mounts := make([]corev1.VolumeMount, 0, len(pvcNames))
	sort.Strings(pvcNames)
	for _, pvc := range pvcNames {
		vname := "v-" + pvc
		volumes = append(volumes, corev1.Volume{
			Name:         vname,
			VolumeSource: corev1.VolumeSource{PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{ClaimName: pvc}},
		})
		mounts = append(mounts, corev1.VolumeMount{Name: vname, MountPath: "/data/" + pvc})
	}
	// desired spec hash (PVC set only; extend if needed)
	h := sha1.Sum([]byte(stringsJoin(pvcNames, ",")))
	desiredHash := hex.EncodeToString(h[:8])

	// Service
	svc := &corev1.Service{ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: namespace, Labels: labels}, Spec: corev1.ServiceSpec{
		Selector: labels,
		Ports:    []corev1.ServicePort{{Name: "http", Port: 8090, TargetPort: intstr.FromInt(8090)}},
	}}
	if _, err := r.Client.CoreV1().Services(namespace).Get(ctx, name, metav1.GetOptions{}); err == nil {
		_ = r.Client.CoreV1().Services(namespace).Delete(ctx, name, metav1.DeleteOptions{})
	}
	_, _ = r.Client.CoreV1().Services(namespace).Create(ctx, svc, metav1.CreateOptions{})

	// Pod (replace if absent)
	if existing, err := r.Client.CoreV1().Pods(namespace).Get(ctx, name, metav1.GetOptions{}); err == nil {
		if existing.Annotations != nil && existing.Annotations["pvcviewer.k8s.io/spec-hash"] == desiredHash {
			return nil
		}
		// spec changed -> recreate
		_ = r.Client.CoreV1().Pods(namespace).Delete(ctx, name, metav1.DeleteOptions{})
	}
	ann := map[string]string{"pvcviewer.k8s.io/spec-hash": desiredHash}
	pod := &corev1.Pod{ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: namespace, Labels: labels, Annotations: ann}, Spec: corev1.PodSpec{
		Containers: []corev1.Container{{
			Name:         "agent",
			Image:        r.AgentImage,
			Command:      []string{"/bin/agent"},
			Env:          []corev1.EnvVar{{Name: "PVC_VIEWER_DATA_ROOT", Value: "/data"}},
			Ports:        []corev1.ContainerPort{{ContainerPort: 8090}},
			VolumeMounts: mounts,
			SecurityContext: &corev1.SecurityContext{
				RunAsNonRoot: boolPtr(true), RunAsUser: int64Ptr(65532), RunAsGroup: int64Ptr(65532),
				AllowPrivilegeEscalation: boolPtr(false), Capabilities: &corev1.Capabilities{Drop: []corev1.Capability{"ALL"}},
			},
		}},
		Volumes:         volumes,
		SecurityContext: &corev1.PodSecurityContext{RunAsUser: int64Ptr(65532), RunAsGroup: int64Ptr(65532)},
	}}
	_, _ = r.Client.CoreV1().Pods(namespace).Create(ctx, pod, metav1.CreateOptions{})
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
