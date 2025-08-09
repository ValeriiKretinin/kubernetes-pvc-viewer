package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"go.uber.org/zap"

	"github.com/valeriikretinin/kubernetes-pvc-viewer/internal/agent"
)

func main() {
	logger, _ := zap.NewProduction()
	defer logger.Sync()
	sugar := logger.Sugar()

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	dataRoot := getenv("PVC_VIEWER_DATA_ROOT", "/data")
	readOnly := getenv("PVC_VIEWER_READ_ONLY", "false") == "true"

	sugar.Infow("agent config", "dataRoot", dataRoot, "readOnly", readOnly)

	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(60 * time.Second))

	r.Get("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})

	srvImpl := agent.NewHTTPServer(dataRoot, readOnly)
	r.Mount("/", srvImpl.Router)

	srv := &http.Server{Addr: ":8090", Handler: r}

	go func() {
		sugar.Infow("agent listening", "addr", srv.Addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			sugar.Fatalw("server error", "error", err)
		}
	}()

	<-ctx.Done()
	sugar.Infow("shutting down")
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
