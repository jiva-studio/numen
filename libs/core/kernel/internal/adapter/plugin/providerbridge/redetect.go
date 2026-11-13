package providerbridge

import (
	"context"
	"errors"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/connection"
	"github.com/jiva-studio/numen/libs/core/kernel/domain/modelcall"
	"github.com/jiva-studio/numen/libs/core/kernel/port/connectionport"
)

// RedetectingSource opens the providers of source. A call through a detected connection that finds its
// service unavailable makes the detection ask the plugin again in the background.
type RedetectingSource struct {
	source    connectionport.ProviderSource
	detection *Detection
}

// NewRedetectingSource returns the source over source that reports unavailable detected connections to
// detection.
func NewRedetectingSource(source connectionport.ProviderSource, detection *Detection) *RedetectingSource {
	return &RedetectingSource{source: source, detection: detection}
}

// OpenProvider opens the provider of the plugin through the source it wraps.
func (s *RedetectingSource) OpenProvider(ctx context.Context, pluginID string) (connectionport.Provider, func(), error) {
	provider, release, err := s.source.OpenProvider(ctx, pluginID)
	if err != nil {
		return nil, nil, err
	}
	return &redetectingProvider{Provider: provider, pluginID: pluginID, detection: s.detection}, release, nil
}

type redetectingProvider struct {
	connectionport.Provider
	pluginID  string
	detection *Detection
}

func (p *redetectingProvider) Call(
	ctx context.Context, operation string, call modelcall.ProviderCall, payload []byte, send modelcall.Send,
) ([]byte, error) {
	var sendErr error
	done, err := p.Provider.Call(ctx, operation, call, payload, func(data []byte) error {
		sendErr = send(data)
		return sendErr
	})
	if err != nil && sendErr == nil && ctx.Err() == nil {
		p.redetectIfUnavailable(call.ConnectionID, err)
	}
	return done, err
}

func (p *redetectingProvider) redetectIfUnavailable(connectionID string, err error) {
	if connection.IsDetected(connectionID) && isUnavailable(err) {
		p.detection.DetectInBackground(p.pluginID)
	}
}

func isUnavailable(err error) bool {
	var tryNext *connection.TryNextError
	var stopError *connection.StopError
	switch {
	case errors.As(err, &tryNext):
		return tryNext.Reason == connection.FailureReasonUnavailable
	case errors.As(err, &stopError):
		return false
	}
	return true
}
