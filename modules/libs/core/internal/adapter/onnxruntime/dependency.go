package onnxruntime

import (
	"context"
	"sync"
	"time"

	"github.com/jiva-studio/numen/modules/libs/core/port"
)

// EnvLibrary is the variable that holds the path of the library a plugin loads.
const EnvLibrary = "NUMEN_ONNX_RUNTIME_LIB"

// progressInterval is the most often the progress of a download is reported.
const progressInterval = 100 * time.Millisecond

// ONNXProvisioner provisions the ONNX Runtime as a runtime dependency.
type ONNXProvisioner struct {
	inner *Provisioner

	mu    sync.Mutex
	ready map[string]port.ResolvedDependency
}

// NewONNXProvisioner returns an ONNXProvisioner for the configuration.
func NewONNXProvisioner(cfg Config) *ONNXProvisioner {
	return &ONNXProvisioner{inner: New(cfg), ready: make(map[string]port.ResolvedDependency)}
}

// CanProvision reports whether the requirement is the ONNX Runtime.
func (p *ONNXProvisioner) CanProvision(req port.DependencyRequirement) bool {
	return req.Kind == port.DependencyKindRuntime && req.ID == port.RuntimeONNX
}

// Provision finds or fetches the library for the plugin folder of the requirement.
func (p *ONNXProvisioner) Provision(
	ctx context.Context, req port.DependencyRequirement, onProgress func(port.DependencyStatus),
) (port.ResolvedDependency, error) {
	var last time.Time
	report := func(done, total int64) {
		if onProgress == nil || (done < total && time.Since(last) < progressInterval) {
			return
		}
		last = time.Now()
		var percent float64
		if total > 0 {
			percent = float64(done) * 100 / float64(total)
		}
		onProgress(port.DependencyStatus{
			ID: req.ID, State: port.DependencyStateProvisioning, BytesDownloaded: done, BytesTotal: total,
			ProgressPercent: percent, Message: "Downloading " + req.ID,
		})
	}
	library, provider, err := p.inner.ProvisionWithProgress(ctx, req.PluginDir, report)
	if err != nil {
		return port.ResolvedDependency{}, err
	}
	resolved := port.ResolvedDependency{
		ID: req.ID, PrimaryPath: library,
		EnvVars:  map[string]string{EnvLibrary: library, EnvProvider: provider},
		Metadata: map[string]string{"provider": provider},
	}
	p.mu.Lock()
	p.ready[req.PluginDir] = resolved
	p.mu.Unlock()
	return resolved, nil
}

// Status answers ready for a plugin folder the library was provisioned for, and
// missing otherwise.
func (p *ONNXProvisioner) Status(_ context.Context, req port.DependencyRequirement) (port.DependencyStatus, error) {
	p.mu.Lock()
	resolved, ok := p.ready[req.PluginDir]
	p.mu.Unlock()
	if !ok {
		return port.DependencyStatus{ID: req.ID, State: port.DependencyStateMissing}, nil
	}
	return port.DependencyStatus{ID: req.ID, State: port.DependencyStateReady, ProgressPercent: 100, Resolved: &resolved}, nil
}
