package main

import (
	"context"
	"errors"
	"fmt"
	"os"

	"connectrpc.com/connect"

	numenv1 "github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1"
	"github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1/numenv1connect"
	"github.com/jiva-studio/numen/modules/libs/plugin-sdk-go/server"
	"github.com/jiva-studio/numen/modules/plugins/downloader/backend/download"
)

type downloadServer struct {
	downloader *download.Downloader
}

func (s *downloadServer) CanHandle(
	_ context.Context,
	req *connect.Request[numenv1.CanHandleRequest],
) (*connect.Response[numenv1.CanHandleResponse], error) {
	can := s.downloader.CanHandle(req.Msg.GetUrl())
	return connect.NewResponse(&numenv1.CanHandleResponse{
		CanHandle: can,
	}), nil
}

func (s *downloadServer) GetDownloadModel(
	_ context.Context,
	req *connect.Request[numenv1.GetDownloadModelRequest],
) (*connect.Response[numenv1.GetDownloadModelResponse], error) {
	model := s.downloader.GetDownloadModel(req.Msg.GetUrl())
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
	meta, err := s.downloader.GetMetadata(ctx, req.Msg.GetUrl())
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
	text, err := s.downloader.GetText(ctx, req.Msg.GetUrl(), pref)
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
	copyInfo, err := s.downloader.Download(ctx, req.Msg.GetUrl(), writer)
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

func main() {
	s := server.New()

	ctx := context.Background()
	downloader, err := download.New(ctx, download.Defaults())
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to initialize downloader: %v\n", err)
		os.Exit(1)
	}

	downloadSvc := &downloadServer{downloader: downloader}
	s.Register(numenv1connect.NewDownloadServiceHandler(downloadSvc, s.HandlerOptions()...))

	if err := s.Serve(); err != nil {
		fmt.Fprintf(os.Stderr, "server error: %v\n", err)
		os.Exit(1)
	}
}

