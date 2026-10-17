package plugin

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"connectrpc.com/connect"

	v1 "github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1"
	"github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1/numenv1connect"

	"github.com/jiva-studio/numen/modules/libs/core/domain"
	"github.com/jiva-studio/numen/modules/libs/core/port"
)

var (
	errNoProofreaderPlugin  = errors.New("no active proofreading plugin found")
	errUnknownProofreadKind = errors.New("unknown kind of text to proofread")
)

// ProofreaderPluginHost provides plugin resolution, activation, and permission checking for proofreading.
type ProofreaderPluginHost interface {
	port.PluginResolver
	port.PluginActivator
	port.PluginAuthorizer
}

// ProofreaderBridge connects port.Proofreader calls to an out-of-process proofreading plugin via Connect-RPC.
type ProofreaderBridge struct {
	host     ProofreaderPluginHost
	pluginID string
	name     string
	mu       sync.Mutex
	acquired bool
}

// OpenProofreaderBridge activates or connects to the proofreading plugin and returns a port.Proofreader.
func OpenProofreaderBridge(host ProofreaderPluginHost) (*ProofreaderBridge, error) {
	plugin, ok := host.FindPluginByCapability("proofreading")
	if !ok {
		return nil, errNoProofreaderPlugin
	}

	return &ProofreaderBridge{
		host:     host,
		pluginID: plugin.Manifest.ID,
		name:     plugin.Manifest.Name,
		acquired: false,
	}, nil
}

// GetName is what made a correction, recorded beside it.
func (b *ProofreaderBridge) GetName() string {
	return b.name
}

// Proofread hands over the lines and answers with what the plugin put right.
func (b *ProofreaderBridge) Proofread(
	ctx context.Context, request domain.ProofreadRequest,
) (domain.ProofreadAnswer, error) {
	kind, err := getWireKind(request.Kind)
	if err != nil {
		return domain.ProofreadAnswer{}, err
	}
	if len(request.Groups) == 0 {
		return domain.ProofreadAnswer{IsDone: true}, nil
	}

	if b.host != nil {
		if err := b.host.CheckPluginPermission(b.pluginID, PermissionVaultRead); err != nil {
			return domain.ProofreadAnswer{}, err
		}
	}

	endpoint, err := b.host.EnsureActive(ctx, b.pluginID)
	if err != nil {
		return domain.ProofreadAnswer{}, fmt.Errorf("ensure active %s: %w", b.pluginID, err)
	}
	defer b.host.Release(b.pluginID)

	httpClient := EndpointClient(endpoint)
	interceptor := connect.WithInterceptors(authInterceptor(endpoint.Token))
	client := numenv1connect.NewProofreadingServiceClient(httpClient, endpoint.BaseURL(), interceptor)

	groups := make([]*v1.ProofreadGroup, len(request.Groups))
	for i, group := range request.Groups {
		lines := make([]*v1.ProofreadLine, len(group.Lines))
		for j, line := range group.Lines {
			lines[j] = &v1.ProofreadLine{Number: int32(line.Number), Text: line.Text}
		}
		groups[i] = &v1.ProofreadGroup{Number: int32(group.Number), Lines: lines}
	}

	resp, err := client.ProofreadText(ctx, connect.NewRequest(&v1.ProofreadTextRequest{
		Kind:            kind,
		Groups:          groups,
		Size:            int32(request.Size),
		Overlap:         int32(request.Overlap),
		FromLine:        int32(request.FromLine),
		LineLimit:       int32(request.LineLimit),
		MaxEditDistance: request.MaxEditDistance,
	}))
	if err != nil {
		return domain.ProofreadAnswer{}, fmt.Errorf("proofread rpc: %w", err)
	}

	corrections := make([]domain.ProofreadCorrection, len(resp.Msg.GetCorrections()))
	for i, one := range resp.Msg.GetCorrections() {
		corrections[i] = domain.ProofreadCorrection{
			Number: int(one.GetNumber()),
			Last:   int(one.GetLast()),
			Text:   one.GetText(),
		}
	}
	return domain.ProofreadAnswer{
		Corrections: corrections,
		Uncorrected: int(resp.Msg.GetUncorrected()),
		NextLine:    int(resp.Msg.GetNextLine()),
		IsDone:      resp.Msg.GetIsDone(),
	}, nil
}

// getWireKind returns the kind as the schema names it.
func getWireKind(kind domain.ProofreadKind) (v1.ProofreadKind, error) {
	switch kind {
	case domain.ProofreadScan:
		return v1.ProofreadKind_PROOFREAD_KIND_SCAN, nil
	case domain.ProofreadSpeech:
		return v1.ProofreadKind_PROOFREAD_KIND_SPEECH, nil
	}
	return v1.ProofreadKind_PROOFREAD_KIND_UNSPECIFIED, errUnknownProofreadKind
}

// Close releases the active reference count for the proofreading plugin in the host supervisor.
func (b *ProofreaderBridge) Close() error {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.acquired && b.host != nil {
		b.host.Release(b.pluginID)
		b.acquired = false
	}
	return nil
}
