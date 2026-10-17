package plugin

import (
	"context"
	"errors"
	"fmt"

	"connectrpc.com/connect"

	v1 "github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1"
	"github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1/numenv1connect"

	"github.com/jiva-studio/numen/modules/libs/core/domain"
	"github.com/jiva-studio/numen/modules/libs/core/port"
)

var (
	errNoProofreaderPlugin = errors.New("no active proofreading plugin found")
)

// ProofreaderPluginHost provides plugin resolution and activation for proofreading.
type ProofreaderPluginHost interface {
	port.PluginResolver
	port.PluginActivator
}

// ProofreaderBridge connects port.Proofreader calls to an out-of-process proofreading plugin via Connect-RPC.
type ProofreaderBridge struct {
	host     ProofreaderPluginHost
	pluginID string
	name     string
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
	}, nil
}

// GetName is what made a correction, recorded beside it.
func (b *ProofreaderBridge) GetName() string {
	return b.name
}

// ProofreadScan hands the pages over and answers with what the plugin corrected.
func (b *ProofreaderBridge) ProofreadScan(
	ctx context.Context, pages []domain.ProofreadPage, maxEditDistance float64,
) (domain.ScanProofreading, error) {
	groups := make([]*v1.ProofreadGroup, len(pages))
	for i, page := range pages {
		groups[i] = &v1.ProofreadGroup{Number: int32(page.Number), Lines: writeWireLines(page.Lines)}
	}
	res, err := b.proofreadText(ctx, &v1.ProofreadTextRequest{
		Kind:   v1.ProofreadKind_PROOFREAD_KIND_SCAN,
		Groups: groups,
		Options: &v1.ProofreadTextRequest_Scan{
			Scan: &v1.ProofreadScanOptions{MaxEditDistance: maxEditDistance},
		},
	})
	if err != nil {
		return domain.ScanProofreading{}, err
	}
	return domain.ScanProofreading{
		Corrections: readWireCorrections(res.GetCorrections()),
		Uncorrected: int(res.GetUncorrected()),
	}, nil
}

// ProofreadSpeech hands the lines of the transcript over and answers with what
// the plugin corrected in the window.
func (b *ProofreaderBridge) ProofreadSpeech(
	ctx context.Context, lines []domain.ProofreadLine, window domain.SpeechWindow,
) (domain.SpeechProofreading, error) {
	res, err := b.proofreadText(ctx, &v1.ProofreadTextRequest{
		Kind:   v1.ProofreadKind_PROOFREAD_KIND_SPEECH,
		Groups: []*v1.ProofreadGroup{{Lines: writeWireLines(lines)}},
		Options: &v1.ProofreadTextRequest_Speech{
			Speech: &v1.ProofreadSpeechOptions{
				Size:      int32(window.Size),
				Overlap:   int32(window.Overlap),
				FromLine:  int32(window.From),
				LineLimit: int32(window.Limit),
			},
		},
	})
	if err != nil {
		return domain.SpeechProofreading{}, err
	}
	return domain.SpeechProofreading{
		Corrections: readWireCorrections(res.GetCorrections()),
		Uncorrected: int(res.GetUncorrected()),
		Next:        int(res.GetNextLine()),
		IsDone:      res.GetIsDone(),
	}, nil
}

// proofreadText sends one request to the plugin, holding its reference for the
// length of the call.
func (b *ProofreaderBridge) proofreadText(
	ctx context.Context, request *v1.ProofreadTextRequest,
) (*v1.ProofreadTextResponse, error) {
	client, lease, err := openPlugin(ctx, b.host, b.pluginID, numenv1connect.NewProofreadingServiceClient)
	if err != nil {
		return nil, err
	}
	defer func() { _ = lease.Close() }()

	res, err := client.ProofreadText(ctx, connect.NewRequest(request))
	if err != nil {
		return nil, readProofreadError(ctx, err)
	}
	return res.Msg, nil
}

// Close releases nothing: a proofread holds its reference on the plugin for the length of the call.
func (b *ProofreaderBridge) Close() error {
	return nil
}

// readProofreadError sorts the failure of a call to the proofreading plugin: the
// end of the caller's context is the caller's own, a plugin that cannot be
// reached or has no model to ask is port.ErrUnavailable, and anything else it
// said is kept as it is.
func readProofreadError(ctx context.Context, err error) error {
	if ctxErr := ctx.Err(); ctxErr != nil {
		return ctxErr
	}
	var failure *connect.Error
	if !errors.As(err, &failure) {
		return fmt.Errorf("%w: %w", port.ErrUnavailable, err)
	}
	switch failure.Code() {
	case connect.CodeUnavailable, connect.CodeUnimplemented:
		return fmt.Errorf("%w: %w", port.ErrUnavailable, err)
	default:
		return fmt.Errorf("proofread rpc: %w", err)
	}
}

func writeWireLines(lines []domain.ProofreadLine) []*v1.ProofreadLine {
	out := make([]*v1.ProofreadLine, len(lines))
	for i, line := range lines {
		out[i] = &v1.ProofreadLine{Number: int32(line.Number), Text: line.Text}
	}
	return out
}

func readWireCorrections(wire []*v1.ProofreadCorrection) []domain.ProofreadCorrection {
	out := make([]domain.ProofreadCorrection, len(wire))
	for i, correction := range wire {
		out[i] = domain.ProofreadCorrection{
			Number: int(correction.GetNumber()),
			Last:   int(correction.GetLast()),
			Text:   correction.GetText(),
		}
	}
	return out
}
