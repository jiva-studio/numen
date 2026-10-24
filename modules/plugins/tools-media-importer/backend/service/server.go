// Package service is the backend of the media downloader plugin.
package service

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"sync"

	"connectrpc.com/connect"

	"github.com/jiva-studio/numen/modules/libs/plugin-sdk-go/server"
	numenv1 "github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1"
	"github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1/numenv1connect"
	"github.com/jiva-studio/numen/modules/plugins/tools-media-importer/backend/download"
)

type downloadServer struct {
	mu         sync.RWMutex
	downloader *download.Downloader
	cfg        download.Config
}

func (s *downloadServer) getDownloader() *download.Downloader {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.downloader
}

func (s *downloadServer) updateConfig(values map[string]string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if v, ok := values["yt_dlp"]; ok {
		if v != "" {
			s.cfg.Video = download.Tool{Command: []string{v}}
		} else {
			s.cfg.Video = download.Tool{}
		}
	}
	if v, ok := values["ffmpeg"]; ok {
		if v != "" {
			s.cfg.Sound = download.Tool{Command: []string{v}}
		} else {
			s.cfg.Sound = download.Tool{}
		}
	}
	if dl, err := download.New(context.Background(), s.cfg); err == nil {
		s.downloader = dl
	}
}

func (s *downloadServer) CanHandle(
	_ context.Context,
	req *connect.Request[numenv1.CanHandleRequest],
) (*connect.Response[numenv1.CanHandleResponse], error) {
	can := s.getDownloader().CanHandle(req.Msg.GetUrl())
	return connect.NewResponse(&numenv1.CanHandleResponse{
		CanHandle: can,
	}), nil
}

func (s *downloadServer) GetDownloadModel(
	_ context.Context,
	req *connect.Request[numenv1.GetDownloadModelRequest],
) (*connect.Response[numenv1.GetDownloadModelResponse], error) {
	model := s.getDownloader().GetDownloadModel(req.Msg.GetUrl())
	return connect.NewResponse(&numenv1.GetDownloadModelResponse{
		Tool:     model.Tool,
		Version:  model.Version,
		Producer: model.Producer,
	}), nil
}

func (s *downloadServer) GetMetadata(
	ctx context.Context,
	req *connect.Request[numenv1.GetMetadataRequest],
) (*connect.Response[numenv1.GetMetadataResponse], error) {
	meta, err := s.getDownloader().GetMetadata(ctx, req.Msg.GetUrl())
	if err != nil {
		if errors.Is(err, download.ErrNothingDownloaded) {
			return nil, connect.NewError(connect.CodeNotFound, err)
		}
		return nil, err
	}
	return connect.NewResponse(&numenv1.GetMetadataResponse{
		Title:       meta.Title,
		Bytes:       meta.Bytes,
		Length:      int32(meta.Length),
		Language:    meta.Language,
		Captions:    meta.Captions,
		IsAutomatic: meta.IsAutomatic,
	}), nil
}

func (s *downloadServer) GetText(
	ctx context.Context,
	req *connect.Request[numenv1.GetTextRequest],
) (*connect.Response[numenv1.GetTextResponse], error) {
	pref := download.PreferredCaptions{}
	if pc := req.Msg.GetPreferredCaptions(); pc != nil {
		pref.Languages = pc.GetLanguages()
		pref.IsAutomatic = pc.GetIsAutomatic()
	}
	text, err := s.getDownloader().GetText(ctx, req.Msg.GetUrl(), pref)
	if err != nil {
		if errors.Is(err, download.ErrNothingDownloaded) {
			return nil, connect.NewError(connect.CodeNotFound, err)
		}
		return nil, err
	}
	protoCues := make([]*numenv1.Cue, len(text.Cues))
	for i, c := range text.Cues {
		protoCues[i] = &numenv1.Cue{
			Text: c.Text,
			From: int32(c.From),
			To:   int32(c.To),
		}
	}
	return connect.NewResponse(&numenv1.GetTextResponse{
		Producer: text.Producer,
		Cues:     protoCues,
		Prose:    text.Prose,
		Title:    text.Title,
		Length:   int32(text.Length),
	}), nil
}

type streamChunkWriter struct {
	stream *connect.ServerStream[numenv1.DownloadMediaResponse]
}

func (w *streamChunkWriter) Write(p []byte) (int, error) {
	if len(p) == 0 {
		return 0, nil
	}
	err := w.stream.Send(&numenv1.DownloadMediaResponse{
		Chunk: p,
	})
	if err != nil {
		return 0, err
	}
	return len(p), nil
}

func (s *downloadServer) DownloadMedia(
	ctx context.Context,
	req *connect.Request[numenv1.DownloadMediaRequest],
	stream *connect.ServerStream[numenv1.DownloadMediaResponse],
) error {
	writer := &streamChunkWriter{stream: stream}
	copyInfo, err := s.getDownloader().Download(ctx, req.Msg.GetUrl(), writer)
	if err != nil {
		if errors.Is(err, download.ErrNothingDownloaded) {
			return connect.NewError(connect.CodeNotFound, err)
		}
		return err
	}
	// Send final response with media type and extension
	return stream.Send(&numenv1.DownloadMediaResponse{
		MediaType: copyInfo.MediaType,
		Extension: copyInfo.Extension,
	})
}

// Mount registers the download service on s.
func Mount(s *server.Server) error {
	cfg := download.Defaults()
	downloader, err := download.New(context.Background(), cfg)
	if err != nil {
		return fmt.Errorf("initialize downloader: %w", err)
	}
	srv := &downloadServer{downloader: downloader, cfg: cfg}
	s.SetUpdateConfig(srv.updateConfig)
	s.Register(numenv1connect.NewDownloadServiceHandler(srv, s.HandlerOptions()...))
	return nil
}

// NewHandler returns the handler of the plugin's backend for a host that links the plugin in, made
// from the variables a launched process of the plugin is given.
func NewHandler(vars map[string]string) (http.Handler, error) {
	return server.NewLinkedHandler(vars, Mount)
}
