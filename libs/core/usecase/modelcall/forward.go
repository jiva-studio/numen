package modelcall

import (
	"context"

	"github.com/jiva-studio/numen/libs/core/port/connectionport"
)

// ForwardedCalls sends each call to the calls get returns when the call is made. Where get returns nil
// every call finds no connection.
type ForwardedCalls struct {
	get func() connectionport.ModelCalls
}

// NewForwardedCalls returns the calls that forward to what get returns.
func NewForwardedCalls(get func() connectionport.ModelCalls) *ForwardedCalls {
	return &ForwardedCalls{get: get}
}

// Complete forwards to the calls of the moment.
func (f *ForwardedCalls) Complete(
	ctx context.Context, callerPluginID string, request connectionport.CompleteRequest, priority connectionport.Priority,
) (connectionport.CompleteResult, error) {
	calls := f.get()
	if calls == nil {
		return connectionport.CompleteResult{}, connectionport.ErrNoConnection
	}
	return calls.Complete(ctx, callerPluginID, request, priority)
}

// Embed forwards to the calls of the moment.
func (f *ForwardedCalls) Embed(
	ctx context.Context, callerPluginID string, texts []string, priority connectionport.Priority,
) ([][]float32, error) {
	calls := f.get()
	if calls == nil {
		return nil, connectionport.ErrNoConnection
	}
	return calls.Embed(ctx, callerPluginID, texts, priority)
}

// GetEmbeddingProfile forwards to the calls of the moment.
func (f *ForwardedCalls) GetEmbeddingProfile(ctx context.Context, callerPluginID string) (connectionport.EmbeddingProfile, error) {
	calls := f.get()
	if calls == nil {
		return connectionport.EmbeddingProfile{}, connectionport.ErrNoConnection
	}
	return calls.GetEmbeddingProfile(ctx, callerPluginID)
}

// Chat forwards to the calls of the moment.
func (f *ForwardedCalls) Chat(
	ctx context.Context, callerPluginID string, request connectionport.ChatRequest, priority connectionport.Priority,
	send func(connectionport.ChatEvent) error,
) error {
	calls := f.get()
	if calls == nil {
		return connectionport.ErrNoConnection
	}
	return calls.Chat(ctx, callerPluginID, request, priority, send)
}
