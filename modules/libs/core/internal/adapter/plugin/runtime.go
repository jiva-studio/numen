package plugin

import (
	"context"
	"errors"
	"fmt"

	"github.com/jiva-studio/numen/modules/libs/core/port"
)

// ErrRuntimeUnavailable indicates that a plugin declares a runtime the host cannot provision.
var ErrRuntimeUnavailable = errors.New("runtime unavailable")

// RuntimeProvisioner makes the ONNX Runtime library ready on this machine. It
// returns the absolute path of the library and the name of the execution
// provider the plugins open their sessions on.
type RuntimeProvisioner interface {
	Provision(ctx context.Context) (library, provider string, err error)
}

// SetRuntimeProvisioner sets what provisions the ONNX Runtime for the plugins that declare it.
func (h *Host) SetRuntimeProvisioner(provisioner RuntimeProvisioner) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.onnxRuntime = provisioner
}

// getRuntimeVars provisions the runtimes a plugin declares and returns the
// environment variables that name them. A runtime that cannot be provisioned is
// an error: the plugin is not launched without it.
func (h *Host) getRuntimeVars(ctx context.Context, runtimes []string) ([]string, error) {
	var vars []string
	for _, name := range runtimes {
		if name != port.RuntimeONNX {
			return nil, fmt.Errorf("%w: %s", ErrRuntimeUnavailable, name)
		}
		h.mu.RLock()
		provisioner := h.onnxRuntime
		h.mu.RUnlock()
		if provisioner == nil {
			return nil, fmt.Errorf("%w: nothing provisions %s", ErrRuntimeUnavailable, name)
		}
		library, provider, err := provisioner.Provision(ctx)
		if err != nil {
			return nil, fmt.Errorf("provision %s: %w", name, err)
		}
		vars = append(vars, EnvONNXRuntimeLib+"="+library, EnvONNXProvider+"="+provider)
	}
	return vars, nil
}
