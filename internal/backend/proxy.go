package backend

import (
	"context"
	"io"
	"net/http"
	"net/url"
	"time"

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
	// Build URL to service DNS to avoid flakiness with manual endpoints resolution
	// svc.ns.svc resolves to ClusterIP with kube-proxy handling load balancing
	host := svcName + "." + ns + ".svc:8090"
	u := url.URL{Scheme: "http", Host: host, Path: path, RawQuery: r.URL.RawQuery}
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

// no additional helpers
