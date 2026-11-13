package providerbridge

import (
	"context"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/modelcall"
	"github.com/jiva-studio/numen/libs/core/kernel/port/connectionport"
)

// CallReportingSource opens the providers of source and reports each call through a connection that a
// provider answered.
type CallReportingSource struct {
	source   connectionport.ProviderSource
	onAnswer func(connectionID string)
}

// NewCallReportingSource returns the source over source that passes the connection of every answered call
// to onAnswer.
func NewCallReportingSource(source connectionport.ProviderSource, onAnswer func(connectionID string)) *CallReportingSource {
	return &CallReportingSource{source: source, onAnswer: onAnswer}
}

// OpenProvider opens the provider of the plugin through the source it wraps.
func (s *CallReportingSource) OpenProvider(ctx context.Context, pluginID string) (connectionport.Provider, func(), error) {
	provider, release, err := s.source.OpenProvider(ctx, pluginID)
	if err != nil {
		return nil, nil, err
	}
	return &callReportingProvider{Provider: provider, onAnswer: s.onAnswer}, release, nil
}

type callReportingProvider struct {
	connectionport.Provider
	onAnswer func(connectionID string)
}

func (p *callReportingProvider) Call(
	ctx context.Context, operation string, call modelcall.ProviderCall, payload []byte, send modelcall.Send,
) ([]byte, error) {
	done, err := p.Provider.Call(ctx, operation, call, payload, send)
	if err == nil {
		p.onAnswer(call.ConnectionID)
	}
	return done, err
}
