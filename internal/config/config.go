package config

import (
	"context"
	"io/ioutil"
	"path/filepath"
	"strings"
	"sync/atomic"
	"time"

	"github.com/fsnotify/fsnotify"
	"go.uber.org/zap"
	"gopkg.in/yaml.v3"
)

type WatchSet struct {
	Include []string `yaml:"include"`
	Exclude []string `yaml:"exclude"`
}

type SecuritySpec struct {
	RunAsUser          *int64  `yaml:"runAsUser"`
	RunAsGroup         *int64  `yaml:"runAsGroup"`
	FSGroup            *int64  `yaml:"fsGroup"`
	SupplementalGroups []int64 `yaml:"supplementalGroups"`
	ReadOnly           bool    `yaml:"readOnly"`
}

type OverrideSpec struct {
	Match        string `yaml:"match"`
	SecuritySpec `yaml:",inline"`
}

type Config struct {
	Watch struct {
		Namespaces     WatchSet `yaml:"namespaces"`
		Pvcs           WatchSet `yaml:"pvcs"`
		StorageClasses WatchSet `yaml:"storageClasses"`
	} `yaml:"watch"`
	Mode struct {
		DataPlane string `yaml:"dataPlane"`
	} `yaml:"mode"`
	Agents   struct {
		SecurityDefaults  SecuritySpec   `yaml:"securityDefaults"`
		SecurityOverrides []OverrideSpec `yaml:"securityOverrides"`
	} `yaml:"agents"`
}

type State struct {
	cfg atomic.Value // *Config
}

func NewState() *State { s := &State{}; s.cfg.Store(&Config{}); return s }

func (s *State) Current() *Config { return s.cfg.Load().(*Config) }

func (s *State) ApplyNewConfig(c *Config) { s.cfg.Store(c) }

func Load(path string) (*Config, error) {
	b, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var c Config
	if err := yaml.Unmarshal(b, &c); err != nil {
		return nil, err
	}
	return &c, nil
}

func WatchFile(ctx context.Context, path string, onChange func(*Config)) error {
	logger, _ := zap.NewProduction()
	defer logger.Sync()
	sugar := logger.Sugar()

	if cfg, err := Load(path); err == nil {
		onChange(cfg)
	} else {
		sugar.Warnw("failed to load initial config", "error", err)
	}

	w, err := fsnotify.NewWatcher()
	if err != nil {
		return err
	}

	dir := filepath.Dir(path)
	if err := w.Add(dir); err != nil {
		return err
	}

	go func() {
		defer w.Close()
		for {
			select {
			case ev := <-w.Events:
				if strings.HasSuffix(ev.Name, filepath.Base(path)) {
					time.Sleep(200 * time.Millisecond)
					if cfg, err := Load(path); err == nil {
						onChange(cfg)
					} else {
						sugar.Warnw("reload failed", "error", err)
					}
				}
			case err := <-w.Errors:
				sugar.Warnw("watch error", "error", err)
			case <-ctx.Done():
				return
			}
		}
	}()
	return nil
}

// end
