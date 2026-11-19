// Package service registers the audio recorder on the plugin server.
package service

import (
	"bytes"
	"context"
	"errors"
	"fmt"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/domain"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/server"
	"github.com/jiva-studio/numen/plugins/views-media-viewer/backend/api"
	"github.com/jiva-studio/numen/plugins/views-media-viewer/backend/recording"
)

// Mount binds the recorder to the vault file writer.
func Mount(s *server.Server) error {
	files, err := host.NewClientFromEnv()
	if err != nil {
		return err
	}
	methods := s.NewAPI(api.Service)
	server.Handle(methods, api.SaveRecording, func(ctx context.Context, req api.SaveRequest) (api.SaveResponse, error) {
		return saveRecording(ctx, files.With(domain.VaultID(req.VaultID)), req)
	})
	return methods.Err()
}

type recordingFiles interface {
	Read(ctx context.Context, path string) ([]byte, domain.FileStat, error)
	Create(ctx context.Context, path string, content []byte) error
}

func saveRecording(ctx context.Context, files recordingFiles, req api.SaveRequest) (api.SaveResponse, error) {
	if len(req.Content) == 0 || req.VaultID == "" {
		return api.SaveResponse{}, server.NewAPIError(server.CodeInvalidArgument, "A recording needs a vault and captured audio")
	}
	name, err := recording.GetPath(req.Folder, req.MediaType, req.StartedAt)
	if err != nil {
		return api.SaveResponse{}, server.NewAPIError(server.CodeInvalidArgument, err.Error())
	}
	err = files.Create(ctx, name, req.Content)
	if errors.Is(err, host.ErrAlreadyExists) {
		data, _, readErr := files.Read(ctx, name)
		if readErr == nil && bytes.Equal(data, req.Content) {
			return api.SaveResponse{Path: name}, nil
		}
	}
	if err != nil {
		return api.SaveResponse{}, fmt.Errorf("save recording: %w", err)
	}
	return api.SaveResponse{Path: name}, nil
}
