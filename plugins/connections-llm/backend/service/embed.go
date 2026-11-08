package service

import (
	"context"
	"net/http"

	"github.com/jiva-studio/numen/libs/plugin-sdk-go/connection"
)

const probeText = "width probe"

type embedBody struct {
	Model string   `json:"model"`
	Input []string `json:"input"`
}

type embedAnswer struct {
	Data []struct {
		Index     *int      `json:"index"`
		Embedding []float32 `json:"embedding"`
	} `json:"data"`
}

// Embed asks the embeddings endpoint of the service for one vector for each text.
func (s *Service) Embed(ctx context.Context, call connection.Call, texts []string) ([][]float32, error) {
	if len(texts) == 0 {
		return nil, ctx.Err()
	}
	var answer embedAnswer
	if err := s.send(ctx, call, operationEmbed, http.MethodPost, "/embeddings", embedBody{Model: call.Model, Input: texts}, &answer); err != nil {
		return nil, err
	}
	if len(answer.Data) != len(texts) {
		return nil, connection.NewStopError(connection.ReasonUnknown)
	}
	vectors := make([][]float32, len(texts))
	for position, one := range answer.Data {
		index := position
		if one.Index != nil {
			index = *one.Index
		}
		if index < 0 || index >= len(vectors) || vectors[index] != nil || len(one.Embedding) == 0 {
			return nil, connection.NewStopError(connection.ReasonUnknown)
		}
		vectors[index] = one.Embedding
	}
	s.markServed(call)
	return vectors, nil
}

// ProbeEmbeddingWidth embeds one short text and returns the length of its vector.
func (s *Service) ProbeEmbeddingWidth(ctx context.Context, call connection.Call) (int, error) {
	vectors, err := s.Embed(ctx, call, []string{probeText})
	if err != nil {
		return 0, err
	}
	return len(vectors[0]), nil
}
