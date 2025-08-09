package main

import (
	"context"
	"embed"
	iofs "io/fs"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"go.uber.org/zap"

	"github.com/valeriikretinin/kubernetes-pvc-viewer/internal/backend"
	"github.com/valeriikretinin/kubernetes-pvc-viewer/internal/config"
	"github.com/valeriikretinin/kubernetes-pvc-viewer/internal/kube"
)

//go:embed static/*
var embeddedStatic embed.FS

func main() {
	logger, _ := zap.NewProduction()
	defer logger.Sync()
	sugar := logger.Sugar()

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	cfgPath := getenv("PVC_VIEWER_CONFIG", "/config/config.yaml")

	cfgState := config.NewState()
	// Kube client
	clientset, _, err := kube.NewClient()
	if err != nil {
		sugar.Fatalw("kube client", "error", err)
	}
	controller := &backend.Controller{Recon: &backend.Reconciler{Client: clientset, AgentImage: getenv("PVC_VIEWER_AGENT_IMAGE", "ghcr.io/example/pvc-viewer-agent:latest"), Defaults: cfgState.Current().Agents.SecurityDefaults, Overrides: cfgState.Current().Agents.SecurityOverrides}, Disc: &backend.Discovery{Client: clientset}, Logger: sugar}
	if err := config.WatchFile(ctx, cfgPath, func(c *config.Config) {
		cfgState.ApplyNewConfig(c)
		controller.Recon.Defaults = c.Agents.SecurityDefaults
		controller.Recon.Overrides = c.Agents.SecurityOverrides
		controller.OnConfigChange(ctx, c)
	}); err != nil {
		sugar.Fatalw("failed to start config watcher", "error", err)
	}
	// periodic reconcile to self-heal
	controller.StartPeriodic(ctx, cfgState.Current, time.Minute)

	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(60 * time.Second))

	// Health endpoints
	r.Get("/api/v1/healthz", func(w http.ResponseWriter, _ *http.Request) {
		sugar.Infow("healthz")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})
	r.Get("/api/v1/readyz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ready"))
	})

	// Agent proxy
	proxy := backend.NewAgentProxy(clientset)
	// Metrics endpoint
	r.Handle("/metrics", backend.MetricsHandler())

	// API
	r.Route("/api/v1", func(api chi.Router) {
		backend.RegisterReadAPIs(api, clientset, cfgState.Current)
		api.Get("/tree", func(w http.ResponseWriter, r *http.Request) {
			ns := r.URL.Query().Get("ns")
			pvc := r.URL.Query().Get("pvc")
			sugar.Infow("/tree", "ns", ns, "pvc", pvc, "rawPath", r.URL.Query().Get("path"))
			svc, newRaw := computeRouting(cfgState.Current(), ns, pvc, r.URL.Query().Get("path"), r.URL.RawQuery)
			rc := r.Clone(r.Context())
			rc.URL.RawQuery = newRaw
			if err := proxy.Proxy(r.Context(), ns, svc, "/v1/tree", w, rc); err != nil {
				sugar.Warnw("proxy tree failed", "ns", ns, "pvc", pvc, "svc", svc, "error", err)
				http.Error(w, "agent unavailable", http.StatusBadGateway)
				return
			}
		})
		api.Get("/download", func(w http.ResponseWriter, r *http.Request) {
			ns := r.URL.Query().Get("ns")
			pvc := r.URL.Query().Get("pvc")
			sugar.Infow("/download", "ns", ns, "pvc", pvc, "path", r.URL.Query().Get("path"))
			svc, newRaw := computeRouting(cfgState.Current(), ns, pvc, r.URL.Query().Get("path"), r.URL.RawQuery)
			rc := r.Clone(r.Context())
			rc.URL.RawQuery = newRaw
			if err := proxy.Proxy(r.Context(), ns, svc, "/v1/file", w, rc); err != nil {
				sugar.Warnw("proxy download failed", "ns", ns, "pvc", pvc, "svc", svc, "error", err)
				http.Error(w, "agent unavailable", http.StatusBadGateway)
				return
			}
		})
		api.Delete("/file", func(w http.ResponseWriter, r *http.Request) {
			ns := r.URL.Query().Get("ns")
			pvc := r.URL.Query().Get("pvc")
			sugar.Infow("/file DELETE", "ns", ns, "pvc", pvc, "path", r.URL.Query().Get("path"))
			svc, newRaw := computeRouting(cfgState.Current(), ns, pvc, r.URL.Query().Get("path"), r.URL.RawQuery)
			rc := r.Clone(r.Context())
			rc.URL.RawQuery = newRaw
			if err := proxy.Proxy(r.Context(), ns, svc, "/v1/file", w, rc); err != nil {
				sugar.Warnw("proxy delete failed", "ns", ns, "pvc", pvc, "svc", svc, "error", err)
				http.Error(w, "agent unavailable", http.StatusBadGateway)
				return
			}
		})
		api.Post("/upload", func(w http.ResponseWriter, r *http.Request) {
			ns := r.URL.Query().Get("ns")
			pvc := r.URL.Query().Get("pvc")
			sugar.Infow("/upload", "ns", ns, "pvc", pvc, "path", r.URL.Query().Get("path"))
			svc, newRaw := computeRouting(cfgState.Current(), ns, pvc, r.URL.Query().Get("path"), r.URL.RawQuery)
			rc := r.Clone(r.Context())
			rc.URL.RawQuery = newRaw
			if err := proxy.Proxy(r.Context(), ns, svc, "/v1/upload", w, rc); err != nil {
				sugar.Warnw("proxy upload failed", "ns", ns, "pvc", pvc, "svc", svc, "error", err)
				http.Error(w, "agent unavailable", http.StatusBadGateway)
				return
			}
		})
		api.Get("/pvc-status", func(w http.ResponseWriter, r *http.Request) {
			ns := r.URL.Query().Get("ns")
			pvc := r.URL.Query().Get("pvc")
			st, _ := (&backend.StatusService{Client: clientset}).GetStatus(r.Context(), ns, pvc)
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte("\"" + string(st) + "\""))
		})
	})

	// Static UI (embedded). Serve contents of subdir "static" as root.
	staticFS, err := iofs.Sub(embeddedStatic, "static")
	if err != nil {
		sugar.Fatalw("embed FS error", "error", err)
	}
	r.Handle("/*", http.FileServer(http.FS(staticFS)))

	srv := &http.Server{Addr: ":8080", Handler: r}

	go func() {
		sugar.Infow("backend listening", "addr", srv.Addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			sugar.Fatalw("server error", "error", err)
		}
	}()

	<-ctx.Done()
	sugar.Infow("shutting down")
  // Best-effort cleanup of all agents on shutdown (e.g., Helm uninstall)
  go func() {
    bg := context.Background()
    if err := controller.Recon.GCPerPVCAll(bg); err != nil {
      sugar.Warnw("gc per-pvc agents on shutdown failed", "error", err)
    }
    if err := controller.Recon.GCNamespaceAgents(bg, map[string]struct{}{}); err != nil {
      sugar.Warnw("gc ns agents on shutdown failed", "error", err)
    }
  }()
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_ = srv.Shutdown(shutdownCtx)
}

func getenv(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}

func backendServiceName(ns, pvc string) string {
	return backend.AgentName(ns, pvc)
}

// agent name generation is delegated to internal/backend.AgentName

// computeRouting picks target Service and rewrites path for per-namespace agents
func computeRouting(cfg *config.Config, ns, pvc, path, rawQuery string) (svcName string, newRaw string) {
	if cfg != nil && cfg.Mode.DataPlane == "agent-per-namespace" {
		// namespace agent service; ensure path is under /data/<pvc>
		q, _ := url.ParseQuery(rawQuery)
		decoded := path
		if u, err := url.QueryUnescape(decoded); err == nil {
			decoded = u
		}
		if !strings.HasPrefix(decoded, "/") {
			decoded = "/" + decoded
		}
		q.Set("path", "/"+pvc+decoded)
		return backend.NamespaceAgentName(ns), q.Encode()
	}
	return backend.AgentName(ns, pvc), rawQuery
}
