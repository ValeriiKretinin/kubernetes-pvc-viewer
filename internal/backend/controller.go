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
		targets, err := c.Disc.BuildTargets(ctx, cfg)
		if err != nil {
			c.Logger.Warnw("build targets failed", "error", err)
			return
		}
		if err := c.Recon.Reconcile(ctx, targets); err != nil {
			c.Logger.Warnw("reconcile failed", "error", err)
		}
	}()
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
