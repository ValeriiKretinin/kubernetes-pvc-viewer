package backend

import (
	"context"
	"time"

	"go.uber.org/zap"

	"github.com/valeriikretinin/kubernetes-pvc-viewer/internal/config"
)

// Controller wires config hot-reload to reconciliation loop.
type Controller struct {
	Recon  *Reconciler
	Disc   *Discovery
	Logger *zap.SugaredLogger
}

func (c *Controller) OnConfigChange(ctx context.Context, cfg *config.Config) {
	// Debounce rapid changes
	go func() {
		time.Sleep(200 * time.Millisecond)
		if cfg.Mode.DataPlane == "agent-per-pvc" {
			targets, err := c.Disc.BuildTargets(ctx, cfg)
			if err != nil {
				c.Logger.Warnw("build targets failed", "error", err)
				return
			}
			if err := c.Recon.Reconcile(ctx, targets); err != nil {
				c.Logger.Warnw("reconcile failed", "error", err)
			}
			return
		}
		if cfg.Mode.DataPlane == "agent-per-namespace" {
			c.reconcilePerNamespace(ctx, cfg)
			return
		}
	}()
}

func (c *Controller) reconcilePerNamespace(ctx context.Context, cfg *config.Config) {
	// group matched PVCs by namespace and ensure one agent per namespace mounts all of them
	targets, err := c.Disc.BuildTargets(ctx, cfg)
	if err != nil {
		c.Logger.Warnw("build targets failed", "error", err)
		return
	}
	nsToPvcs := map[string][]string{}
	for _, t := range targets {
		nsToPvcs[t.Namespace] = append(nsToPvcs[t.Namespace], t.PVCName)
	}
	for ns, pvcs := range nsToPvcs {
		if err := c.Recon.EnsureNamespaceAgent(ctx, ns, pvcs); err != nil {
			c.Logger.Warnw("ns agent ensure failed", "ns", ns, "error", err)
		}
	}
}

// StartPeriodic starts a background reconcile ticker.
func (c *Controller) StartPeriodic(ctx context.Context, cfgProvider func() *config.Config, interval time.Duration) {
	if interval <= 0 {
		interval = time.Minute
	}
	go func() {
		t := time.NewTicker(interval)
		defer t.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-t.C:
				cfg := cfgProvider()
				targets, err := c.Disc.BuildTargets(ctx, cfg)
				if err != nil {
					c.Logger.Warnw("periodic build targets failed", "error", err)
					continue
				}
				if err := c.Recon.Reconcile(ctx, targets); err != nil {
					c.Logger.Warnw("periodic reconcile failed", "error", err)
				}
			}
		}
	}()
}
