package backend

import (
	"context"
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"strings"
	"sync/atomic"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/kubernetes"

	"github.com/bmatcuk/doublestar/v4"
	"github.com/valeriikretinin/kubernetes-pvc-viewer/internal/config"
	"go.uber.org/zap"
)

type Target struct {
	Namespace    string
	PVCName      string
	StorageClass string
}

type Reconciler struct {
	Client     kubernetes.Interface
	AgentImage string
	Defaults   config.SecuritySpec
	Overrides  []config.OverrideSpec
	Disabled   atomic.Bool
	Logger     *zap.SugaredLogger
}

func (r *Reconciler) Reconcile(ctx context.Context, targets []Target) error {
	if r.Disabled.Load() {
		return nil
	}
	desired := map[string]Target{}
	for _, t := range targets {
		desired[key(t)] = t
	}
	// List existing agent Pods by label
	sel := labels.SelectorFromSet(labels.Set{"app": "pvc-viewer-agent"})
	pods, err := r.Client.CoreV1().Pods("").List(ctx, metav1.ListOptions{LabelSelector: sel.String()})
	if err != nil {
		return err
	}

	existing := map[string]struct{}{}
	for _, p := range pods.Items {
		ns := p.Labels["pvcviewer.k8s.io/ns"]
		pvc := p.Labels["pvcviewer.k8s.io/pvc"]
		if ns != "" && pvc != "" {
			existing[key(Target{Namespace: ns, PVCName: pvc})] = struct{}{}
		}
	}

	// Ensure desired
	for _, t := range targets {
		if err := r.ensureAgent(ctx, t); err != nil {
			return err
		}
	}

	// GC
	for k := range existing {
		if _, ok := desired[k]; !ok {
			// parse key
			parts := strings.SplitN(k, "/", 2)
			if len(parts) != 2 {
				continue
			}
			_ = r.Client.CoreV1().Pods(parts[0]).Delete(ctx, AgentName(parts[0], parts[1]), metav1.DeleteOptions{})
			_ = r.Client.CoreV1().Services(parts[0]).Delete(ctx, AgentName(parts[0], parts[1]), metav1.DeleteOptions{})
		}
	}
	return nil
}

func (r *Reconciler) ensureAgent(ctx context.Context, t Target) error {
	if r.Disabled.Load() {
		return nil
	}
	name := AgentName(t.Namespace, t.PVCName)
	labels := map[string]string{
		"app":                  "pvc-viewer-agent",
		"pvcviewer.k8s.io/ns":  t.Namespace,
		"pvcviewer.k8s.io/pvc": t.PVCName,
	}
	// Ensure Service (ClusterIP)
	_, _ = r.Client.CoreV1().Services(t.Namespace).Get(ctx, name, metav1.GetOptions{})
	svc := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: t.Namespace, Labels: labels},
		Spec: corev1.ServiceSpec{
			Selector: labels,
			Ports: []corev1.ServicePort{{
				Name: "http", Port: 8090, TargetPort: intstr.FromInt(8090), Protocol: corev1.ProtocolTCP,
			}},
		},
	}
	if _, err := r.Client.CoreV1().Services(t.Namespace).Create(ctx, svc, metav1.CreateOptions{}); err == nil {
		if r.Logger != nil {
			r.Logger.Infow("agent service ensured", "ns", t.Namespace, "svc", name)
		}
	}

	// Ensure Pod
	if _, err := r.Client.CoreV1().Pods(t.Namespace).Get(ctx, name, metav1.GetOptions{}); err == nil {
		return nil
	}
	// Resolve security for this storageClass
	sec := r.resolveSecurityForStorageClass(t.PVCName, t.StorageClass)

	ro := sec.ReadOnly
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: t.Namespace, Labels: labels},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{{
				Name:         "agent",
				Image:        r.AgentImage,
				Command:      []string{"/bin/agent"},
				Env:          []corev1.EnvVar{{Name: "PVC_VIEWER_DATA_ROOT", Value: "/data"}, {Name: "PVC_VIEWER_READ_ONLY", Value: boolString(ro)}},
				Ports:        []corev1.ContainerPort{{ContainerPort: 8090}},
				VolumeMounts: []corev1.VolumeMount{{Name: "data", MountPath: "/data", ReadOnly: ro}},
				SecurityContext: &corev1.SecurityContext{
					RunAsNonRoot:             boolPtr(true),
					RunAsUser:                pickInt(sec.RunAsUser, 65532),
					RunAsGroup:               pickInt(sec.RunAsGroup, 65532),
					AllowPrivilegeEscalation: boolPtr(false),
					Capabilities:             &corev1.Capabilities{Drop: []corev1.Capability{"ALL"}},
				},
			}},
			Volumes: []corev1.Volume{{
				Name:         "data",
				VolumeSource: corev1.VolumeSource{PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{ClaimName: t.PVCName}},
			}},
			SecurityContext: &corev1.PodSecurityContext{
				RunAsUser:          pickInt(sec.RunAsUser, 65532),
				RunAsGroup:         pickInt(sec.RunAsGroup, 65532),
				FSGroup:            sec.FSGroup,
				SupplementalGroups: mergeSupplemental(r.Defaults.SupplementalGroups, sec.SupplementalGroups),
			},
		},
	}
	if _, err := r.Client.CoreV1().Pods(t.Namespace).Create(ctx, pod, metav1.CreateOptions{}); err == nil {
		if r.Logger != nil {
			r.Logger.Infow("agent pod ensured", "ns", t.Namespace, "pod", name, "pvc", t.PVCName)
		}
	}
	return nil
}

func key(t Target) string { return fmt.Sprintf("%s/%s", t.Namespace, t.PVCName) }

// AgentName returns deterministic name for agent Pod/Service
func AgentName(ns, pvc string) string {
	h := sha1.Sum([]byte(ns + ":" + pvc))
	return "pvc-viewer-agent-" + hex.EncodeToString(h[:8])
}

func boolPtr(b bool) *bool    { return &b }
func int64Ptr(i int64) *int64 { return &i }

func boolString(b bool) string {
	if b {
		return "true"
	}
	return "false"
}

// resolveSecurityForStorageClass merges defaults with first matching override
func (r *Reconciler) resolveSecurityForStorageClass(pvcName, sc string) config.SecuritySpec {
	out := r.Defaults
	for _, o := range r.Overrides {
		// pvc-specific override takes precedence
		if o.PvcMatch != "" {
			if ok, _ := doublestar.Match(o.PvcMatch, pvcName); ok {
				if o.RunAsUser != nil {
					out.RunAsUser = o.RunAsUser
				}
				if o.RunAsGroup != nil {
					out.RunAsGroup = o.RunAsGroup
				}
				if o.FSGroup != nil {
					out.FSGroup = o.FSGroup
				}
				if len(o.SupplementalGroups) > 0 {
					out.SupplementalGroups = o.SupplementalGroups
				}
				out.ReadOnly = out.ReadOnly || o.ReadOnly
				break
			}
			continue
		}
		// simple exact match on storageClass
		if o.Match == sc {
			if o.RunAsUser != nil {
				out.RunAsUser = o.RunAsUser
			}
			if o.RunAsGroup != nil {
				out.RunAsGroup = o.RunAsGroup
			}
			if o.FSGroup != nil {
				out.FSGroup = o.FSGroup
			}
			if len(o.SupplementalGroups) > 0 {
				out.SupplementalGroups = o.SupplementalGroups
			}
			out.ReadOnly = out.ReadOnly || o.ReadOnly
			break
		}
	}
	return out
}

func pickInt(v *int64, def int64) *int64 {
	if v != nil {
		return v
	}
	return &def
}

func mergeSupplemental(a, b []int64) []int64 {
	if len(b) == 0 {
		return a
	}
	m := map[int64]struct{}{}
	out := make([]int64, 0, len(a)+len(b))
	for _, x := range a {
		if _, ok := m[x]; !ok {
			m[x] = struct{}{}
			out = append(out, x)
		}
	}
	for _, x := range b {
		if _, ok := m[x]; !ok {
			m[x] = struct{}{}
			out = append(out, x)
		}
	}
	return out
}

func valueOrNil(p *int64) interface{} {
	if p == nil {
		return nil
	}
	return *p
}

// DiscoverTargets is a placeholder: in real impl we would list PVCs and apply matchers.
func (r *Reconciler) DiscoverTargets(ctx context.Context) ([]Target, error) {
	// TODO: implement
	return []Target{}, nil
}
