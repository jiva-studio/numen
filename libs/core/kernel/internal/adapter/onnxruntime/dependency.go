package onnxruntime

import (
	"context"
	"github.com/jiva-studio/numen/libs/core/kernel/port/machineport"
	"time"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/plugin"
	"github.com/jiva-studio/numen/libs/plugin/catalog/manifest"
)

// EnvLibrary is the variable that holds the path of the library a plugin loads.
const EnvLibrary = "NUMEN_ONNX_RUNTIME_LIB"

// progressInterval is the most often the progress of a download is reported.
const progressInterval = 100 * time.Millisecond

// ONNXProvisioner provisions the ONNX Runtime as a runtime dependency.
type ONNXProvisioner struct {
	inner provisioning
	clock machineport.Clock
}

func NewONNXProvisioner(cfg Config) *ONNXProvisioner {
	return &ONNXProvisioner{inner: New(cfg), clock: cfg.Clock}
}

// CanProvision reports whether the requirement is the ONNX Runtime.
func (p *ONNXProvisioner) CanProvision(req manifest.DependencyRequirement) bool {
	return req.Kind == manifest.DependencyKindRuntime && req.ID == plugin.RuntimeONNX
}

// Provision finds or fetches the library for the plugin folder of the requirement.
func (p *ONNXProvisioner) Provision(
	ctx context.Context, req manifest.DependencyRequirement, onProgress func(plugin.DependencyStatus),
) (plugin.ResolvedDependency, error) {
	var last time.Time
	report := func(done, total int64) {
		if onProgress == nil || (done < total && p.clock.Now().Sub(last) < progressInterval) {
			return
		}
		last = p.clock.Now()
		var percent float64
		if total > 0 {
			percent = float64(done) * 100 / float64(total)
		}
		onProgress(plugin.DependencyStatus{
			ID: req.ID, State: plugin.DependencyStateProvisioning, BytesDownloaded: done, BytesTotal: total,
			ProgressPercent: percent, Message: "Downloading " + req.ID,
		})
	}
	library, provider, err := p.inner.ProvisionWithProgress(ctx, req.PluginDir, report)
	if err != nil {
		return plugin.ResolvedDependency{}, err
	}
	resolved := plugin.ResolvedDependency{
		ID: req.ID, PrimaryPath: library,
		EnvVars:  map[string]string{EnvLibrary: library, EnvProvider: provider},
		Metadata: map[string]string{"provider": provider},
	}
	return resolved, nil
}

// provisioning fetches or finds the library with the progress of a download reported.
type provisioning interface {
	ProvisionWithProgress(ctx context.Context, pluginDir string, report func(done, total int64)) (library, provider string, err error)
}
