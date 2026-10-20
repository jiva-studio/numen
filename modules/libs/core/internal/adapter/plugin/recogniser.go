package plugin

import (
	"bytes"
	"context"
	"fmt"
	"image"
	"image/png"

	"connectrpc.com/connect"

	v1 "github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1"
	"github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1/numenv1connect"

	"github.com/jiva-studio/numen/modules/libs/core/port"
	"github.com/jiva-studio/numen/modules/libs/notes/domain"
)

// RecogniserPluginHost provides plugin resolution, activation, and permission checking for OCR.
type RecogniserPluginHost interface {
	port.PluginResolver
	port.PluginActivator
	port.PluginAuthorizer
}

// RecogniserBridge connects port.Recogniser calls to an out-of-process OCR plugin via Connect-RPC.
type RecogniserBridge struct {
	host     RecogniserPluginHost
	pluginID string
	service  pluginService[numenv1connect.RecognitionServiceClient]
	lease    *pluginLease
	model    port.RecognitionModel
}

// OpenRecogniserBridge activates or connects to the OCR plugin and returns a port.Recogniser.
func OpenRecogniserBridge(ctx context.Context, host RecogniserPluginHost) (*RecogniserBridge, error) {
	plugin, err := host.ResolveCapability(CapabilityRecognition)
	if err != nil {
		return nil, err
	}

	client, lease, err := openPlugin(ctx, host, plugin.Manifest.ID, numenv1connect.NewRecognitionServiceClient)
	if err != nil {
		return nil, err
	}

	modelCtx, cancel := context.WithTimeout(ctx, modelInfoTimeout)
	defer cancel()
	resp, err := client.GetRecognitionModel(modelCtx, connect.NewRequest(&v1.GetRecognitionModelRequest{}))
	if err != nil {
		_ = lease.Close()
		return nil, fmt.Errorf("get recognition model %s: %w", plugin.Manifest.ID, err)
	}

	model := port.RecognitionModel{
		Layout:     resp.Msg.GetLayout(),
		Recogniser: resp.Msg.GetRecogniser(),
		DPI:        int(resp.Msg.GetDpi()),
		From:       resp.Msg.GetFrom(),
	}

	return &RecogniserBridge{
		host:     host,
		pluginID: plugin.Manifest.ID,
		service:  pluginService[numenv1connect.RecognitionServiceClient]{host: host, pluginID: plugin.Manifest.ID, newClient: numenv1connect.NewRecognitionServiceClient},
		lease:    lease,
		model:    model,
	}, nil
}

// GetRecognitionModel returns the model the OCR plugin answered with when the bridge was opened.
func (b *RecogniserBridge) GetRecognitionModel() port.RecognitionModel {
	return b.model
}

// Recognise sends page image bytes to the OCR plugin process and decodes returned blocks.
func (b *RecogniserBridge) Recognise(ctx context.Context, page image.Image) ([]domain.OCRBlock, error) {
	if b.host != nil {
		if err := b.host.CheckPluginPermission(b.pluginID, PermissionVaultRead); err != nil {
			return nil, err
		}
	}

	var buf bytes.Buffer
	enc := png.Encoder{CompressionLevel: png.BestSpeed}
	if err := enc.Encode(&buf, page); err != nil {
		return nil, fmt.Errorf("encode page png: %w", err)
	}

	req := connect.NewRequest(&v1.RecognisePageRequest{
		ImageData: buf.Bytes(),
	})
	client, call, err := b.service.open(ctx)
	if err != nil {
		return nil, err
	}
	defer func() { _ = call.Close() }()

	resp, err := client.RecognisePage(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("rpc recognise page: %w", err)
	}

	return convertBlocks(resp.Msg.GetBlocks()), nil
}

// Close releases the active reference count for the OCR plugin in the host supervisor.
func (b *RecogniserBridge) Close() error {
	return b.lease.Close()
}

func convertBlocks(protoBlocks []*v1.OcrBlock) []domain.OCRBlock {
	blocks := make([]domain.OCRBlock, 0, len(protoBlocks))
	for _, pb := range protoBlocks {
		boxes := make([]domain.OCRBox, 0, len(pb.GetBoxes()))
		for _, bx := range pb.GetBoxes() {
			span := domain.ByteSpan{From: 0, To: 0}
			if s := bx.GetSpan(); s != nil {
				span = domain.ByteSpan{From: int(s.GetFrom()), To: int(s.GetTo())}
			}
			boxes = append(boxes, domain.OCRBox{
				Rect: image.Rect(int(bx.GetMinX()), int(bx.GetMinY()), int(bx.GetMaxX()), int(bx.GetMaxY())),
				Span: span,
			})
		}

		blocks = append(blocks, domain.OCRBlock{
			Label:     pb.GetLabel(),
			Text:      pb.GetText(),
			IsHeading: pb.GetIsHeading(),
			Depth:     int(pb.GetDepth()),
			Boxes:     boxes,
		})
	}
	return blocks
}

func authInterceptor(token string) connect.UnaryInterceptorFunc {
	return func(next connect.UnaryFunc) connect.UnaryFunc {
		return func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
			if token != "" {
				req.Header().Set("Authorization", "Bearer "+token)
			}
			return next(ctx, req)
		}
	}
}
