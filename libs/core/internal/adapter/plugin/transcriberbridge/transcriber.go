package transcriberbridge

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"connectrpc.com/connect"

	v1 "github.com/jiva-studio/numen/libs/protocol/gen/numen/v1"
	"github.com/jiva-studio/numen/libs/protocol/gen/numen/v1/numenv1connect"

	"github.com/jiva-studio/numen/libs/core/internal/adapter/plugin/bridge"
	"github.com/jiva-studio/numen/libs/core/internal/adapter/plugin/plugindir"
	"github.com/jiva-studio/numen/libs/core/internal/adapter/plugin/security"
	"github.com/jiva-studio/numen/libs/core/internal/adapter/plugin/transport"
	"github.com/jiva-studio/numen/libs/core/port/pluginport"
	"github.com/jiva-studio/numen/libs/core/port/transcriptionport"
)

// Host provides resolution, activation, reference tracking, permissions, and path validation.
type Host interface {
	pluginport.PluginResolver
	pluginport.PluginActivator
	pluginport.PluginAcquirer
	pluginport.PluginAuthorizer
	pluginport.PluginPathValidator
}

// Transcriber connects port.Transcriber calls to an out-of-process transcription plugin via Connect-RPC.
type Transcriber struct {
	host     Host
	pluginID string
	service  bridge.Service[numenv1connect.TranscriptionServiceClient]
	lease    *bridge.Lease
	model    transcriptionport.TranscriptionModel
}

// Open activates or connects to the transcription plugin and returns a port.Transcriber.
func Open(ctx context.Context, host Host) (*Transcriber, error) {
	plugin, err := host.ResolveCapability(bridge.CapabilityTranscription)
	if err != nil {
		return nil, err
	}

	client, lease, err := bridge.Open(ctx, host, plugin.Manifest.ID, numenv1connect.NewTranscriptionServiceClient)
	if err != nil {
		return nil, err
	}

	modelCtx, cancel := context.WithTimeout(ctx, transport.ModelInfoTimeout)
	defer cancel()
	resp, err := client.GetTranscriptionModel(modelCtx, connect.NewRequest(&v1.GetTranscriptionModelRequest{}))
	if err != nil {
		_ = lease.Close()
		return nil, fmt.Errorf("get transcription model %s: %w", plugin.Manifest.ID, err)
	}

	model := transcriptionport.TranscriptionModel{
		Model:     resp.Msg.GetModel(),
		Segmenter: resp.Msg.GetSegmenter(),
		Cutting:   resp.Msg.GetCutting(),
		From:      resp.Msg.GetFrom(),
	}

	return &Transcriber{
		host:     host,
		pluginID: plugin.Manifest.ID,
		service:  bridge.Service[numenv1connect.TranscriptionServiceClient]{Host: host, PluginID: plugin.Manifest.ID, NewClient: numenv1connect.NewTranscriptionServiceClient},
		lease:    lease,
		model:    model,
	}, nil
}

// GetTranscriptionModel returns the model the transcription plugin answered with when the bridge was opened.
func (b *Transcriber) GetTranscriptionModel() transcriptionport.TranscriptionModel {
	return b.model
}

// Open sends recording raw bytes to the plugin to calculate duration and prepare segmentation.
func (b *Transcriber) Open(ctx context.Context, raw []byte) (transcriptionport.Recording, error) {
	return b.OpenSource(ctx, raw)
}

// OpenPath sends recording file path to the plugin to calculate duration and prepare segmentation.
func (b *Transcriber) OpenPath(ctx context.Context, path string) (transcriptionport.Recording, error) {
	return b.OpenSource(ctx, path)
}

// OpenSource sends recording raw bytes or file path to the plugin to calculate duration and prepare segmentation.
func (b *Transcriber) OpenSource(ctx context.Context, source any) (transcriptionport.Recording, error) {
	if b.host != nil {
		if err := b.host.CheckPluginPermission(b.pluginID, plugindir.PermissionVaultRead); err != nil {
			return nil, err
		}
	}

	var req *connect.Request[v1.OpenRecordingRequest]

	switch s := source.(type) {
	case string:
		validated, err := b.resolveNamedPath(s)
		if err != nil {
			return nil, err
		}
		req = connect.NewRequest(&v1.OpenRecordingRequest{
			Path: validated,
		})
	case interface{ Path() string }:
		validated, err := b.resolveNamedPath(s.Path())
		if err != nil {
			return nil, err
		}
		req = connect.NewRequest(&v1.OpenRecordingRequest{
			Path: validated,
		})
	case interface{ Name() string }:
		validated, err := b.resolveNamedPath(s.Name())
		if err != nil {
			return nil, err
		}
		req = connect.NewRequest(&v1.OpenRecordingRequest{
			Path: validated,
		})
	case []byte:
		req = connect.NewRequest(&v1.OpenRecordingRequest{
			AudioData: s,
		})
	default:
		return nil, errors.New("unsupported audio source type")
	}

	client, call, err := b.service.Open(ctx)
	if err != nil {
		return nil, err
	}
	defer func() { _ = call.Close() }()

	resp, err := client.OpenRecording(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("rpc open recording: %w", err)
	}

	if b.host != nil {
		b.host.Acquire(b.pluginID)
	}

	return &pluginRecording{
		client:      client,
		recordingID: resp.Msg.GetRecordingId(),
		durationMs:  int(resp.Msg.GetDurationMs()),
		host:        b.host,
		pluginID:    b.pluginID,
	}, nil
}

func (b *Transcriber) resolveNamedPath(rawPath string) (string, error) {
	if b.host == nil {
		return "", security.ErrEmptyVaultRoot
	}
	return b.host.ValidatePluginPath(b.pluginID, rawPath)
}

// Transcribe decodes one audio segment into text words.
func (b *Transcriber) Transcribe(ctx context.Context, audio transcriptionport.Audio) (string, error) {
	if b.host != nil {
		if err := b.host.CheckPluginPermission(b.pluginID, plugindir.PermissionVaultRead); err != nil {
			return "", err
		}
	}

	req := connect.NewRequest(&v1.TranscribeRequest{
		Samples: audio.Samples,
		FromMs:  int32(audio.From),
		ToMs:    int32(audio.To),
	})
	client, call, err := b.service.Open(ctx)
	if err != nil {
		return "", err
	}
	defer func() { _ = call.Close() }()

	resp, err := client.Transcribe(ctx, req)
	if err != nil {
		return "", fmt.Errorf("rpc transcribe: %w", err)
	}
	return resp.Msg.GetText(), nil
}

// Close releases the active reference count for the transcription plugin in the host supervisor.
func (b *Transcriber) Close() error {
	return b.lease.Close()
}

type pluginRecording struct {
	mu          sync.Mutex
	client      numenv1connect.TranscriptionServiceClient
	recordingID string
	durationMs  int
	host        pluginport.PluginActivator
	pluginID    string
}

func (r *pluginRecording) GetDuration() int {
	return r.durationMs
}

func (r *pluginRecording) GetSegments(ctx context.Context, from, count int) ([]transcriptionport.Audio, error) {
	req := connect.NewRequest(&v1.GetSegmentsRequest{
		RecordingId: r.recordingID,
		FromMs:      int32(from),
		Count:       int32(count),
	})
	resp, err := r.client.GetSegments(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("rpc get segments: %w", err)
	}

	segments := make([]transcriptionport.Audio, 0, len(resp.Msg.GetSegments()))
	for _, s := range resp.Msg.GetSegments() {
		segments = append(segments, transcriptionport.Audio{
			Samples: s.GetSamples(),
			From:    int(s.GetFromMs()),
			To:      int(s.GetToMs()),
		})
	}
	return segments, nil
}

func (r *pluginRecording) Close() error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.recordingID == "" {
		return nil
	}
	if r.client != nil {
		ctx := context.Background()
		req := connect.NewRequest(&v1.CloseRecordingRequest{
			RecordingId: r.recordingID,
		})
		_, _ = r.client.CloseRecording(ctx, req)
	}
	r.recordingID = ""
	if r.host != nil && r.pluginID != "" {
		r.host.Release(r.pluginID)
		r.host = nil
	}
	return nil
}
