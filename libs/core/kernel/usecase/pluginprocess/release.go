package pluginprocess

import "context"

// Stopper stops the process of a plugin that has finished its work.
type Stopper interface {
	// ReleaseProcess stops the process of the plugin at once where it holds no lease, and else once it
	// holds none. A lease taken afterwards cancels the release.
	ReleaseProcess(ctx context.Context, pluginID string)
}

// ReleaseProcess stops the process of a plugin whose frontend says its backend has finished its work.
type ReleaseProcess struct {
	stopper Stopper
}

// NewReleaseProcess returns the use case over the stopper of the processes.
func NewReleaseProcess(stopper Stopper) *ReleaseProcess {
	return &ReleaseProcess{stopper: stopper}
}

// Execute releases the process of the plugin. A plugin with no process is already released.
func (u *ReleaseProcess) Execute(ctx context.Context, pluginID string) {
	u.stopper.ReleaseProcess(ctx, pluginID)
}
