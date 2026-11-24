package service

import (
	"context"
	"errors"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/domain"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/api"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/usecase/lifecycle"
)

var errNoLifecycle = errors.New("the sync does not follow the screen")

// idleStream is the stream of a server that sends no hints: it holds until the sync stops.
type idleStream struct{}

func (idleStream) Watch(ctx context.Context, _ func(domain.VaultID)) error {
	<-ctx.Done()
	return ctx.Err()
}

// SetLifecycle sets what Suspend and Resume start and stop.
func (b *Backend) SetLifecycle(life *lifecycle.Lifecycle) { b.life = life }

// Suspend stops the sync as the application leaves the screen, after one last pass.
func (b *Backend) Suspend(ctx context.Context, _ api.SuspendRequest) (api.SuspendResponse, error) {
	if b.life == nil {
		return api.SuspendResponse{}, errNoLifecycle
	}
	return api.SuspendResponse{}, b.life.Suspend(ctx)
}

// Resume starts the sync as the application comes to the screen.
func (b *Backend) Resume(ctx context.Context, _ api.ResumeRequest) (api.ResumeResponse, error) {
	if b.life == nil {
		return api.ResumeResponse{}, errNoLifecycle
	}
	return api.ResumeResponse{}, b.life.Resume(ctx)
}
