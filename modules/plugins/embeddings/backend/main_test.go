package main

import (
	"context"
	"testing"

	"connectrpc.com/connect"
	numenv1 "github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1"
)

func TestEmbeddingServerGetModel(t *testing.T) {
	srv := newEmbeddingServer()
	resp, err := srv.GetEmbeddingModel(context.Background(), connect.NewRequest(&numenv1.GetEmbeddingModelRequest{}))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	msg := resp.Msg
	if msg.GetName() != "bge-m3" {
		t.Errorf("expected name bge-m3, got %s", msg.GetName())
	}
	if msg.GetDimensions() != 1024 {
		t.Errorf("expected 1024 dimensions, got %d", msg.GetDimensions())
	}
	if msg.GetMaxTokens() != 8192 {
		t.Errorf("expected 8192 max tokens, got %d", msg.GetMaxTokens())
	}
	if msg.GetPooling() != "head" {
		t.Errorf("expected head pooling, got %s", msg.GetPooling())
	}
}

func TestEmbeddingServerEnvOverrides(t *testing.T) {
	t.Setenv("NUMEN_EMBEDDING_MODEL", "custom-model")
	t.Setenv("NUMEN_EMBEDDING_DIMENSIONS", "512")
	t.Setenv("NUMEN_EMBEDDING_MAX_TOKENS", "1024")
	t.Setenv("NUMEN_EMBEDDING_POOLING", "mean")

	srv := newEmbeddingServer()
	resp, err := srv.GetEmbeddingModel(context.Background(), connect.NewRequest(&numenv1.GetEmbeddingModelRequest{}))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	msg := resp.Msg
	if msg.GetName() != "custom-model" {
		t.Errorf("expected custom-model, got %s", msg.GetName())
	}
	if msg.GetDimensions() != 512 {
		t.Errorf("expected 512 dimensions, got %d", msg.GetDimensions())
	}
	if msg.GetMaxTokens() != 1024 {
		t.Errorf("expected 1024 max tokens, got %d", msg.GetMaxTokens())
	}
	if msg.GetPooling() != "mean" {
		t.Errorf("expected mean pooling, got %s", msg.GetPooling())
	}
}
