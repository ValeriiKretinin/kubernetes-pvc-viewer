package backend

import (
	"context"
	"io"
	"net/http"
	"net/url"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// Simple proxy that finds agent Pod IP via Endpoints and forwards request.
type AgentProxy struct {
	Client kubernetes.Interface
	HTTP   *http.Client
}

func NewAgentProxy(c kubernetes.Interface) *AgentProxy {
	return &AgentProxy{Client: c, HTTP: &http.Client{Timeout: 120 * time.Second}}
}

func (p *AgentProxy) Proxy(ctx context.Context, ns, svcName string, path string, w http.ResponseWriter, r *http.Request) error {
	// Lookup endpoints
	eps, err := p.Client.CoreV1().Endpoints(ns).Get(ctx, svcName, metav1.GetOptions{})
	if err != nil {
		return err
	}
	var ip string
	for _, ss := range eps.Subsets {
		for _, addr := range ss.Addresses {
			ip = addr.IP
			break
		}
		if ip != "" {
			break
		}
	}
	if ip == "" {
		return context.DeadlineExceeded
	}

	// Build URL
	u := url.URL{Scheme: "http", Host: ip + ":8090", Path: path, RawQuery: r.URL.RawQuery}
	req, err := http.NewRequestWithContext(ctx, r.Method, u.String(), r.Body)
	if err != nil {
		return err
	}
	req.Header = r.Header.Clone()
	resp, err := p.HTTP.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// Copy headers/status/body
	for k, vv := range resp.Header {
		for _, v := range vv {
			w.Header().Add(k, v)
		}
	}
	w.WriteHeader(resp.StatusCode)
	_, _ = io.Copy(w, resp.Body)
	return nil
}

// Helper to create empty Endpoints for new Service (not used now, for completeness)
func newEmptyEndpoints(ns, name string) *corev1.Endpoints {
	return &corev1.Endpoints{ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: ns}}
}
