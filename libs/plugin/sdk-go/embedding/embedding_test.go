package embedding

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	"connectrpc.com/connect"
	v1 "github.com/jiva-studio/numen/libs/core/protocol/gen/numen/v1"
	"github.com/jiva-studio/numen/libs/core/protocol/gen/numen/v1/numenv1connect"
)

type fakeHandler struct {
	info  ModelInfo
	roles *[]Role
	err   error
}

func (f fakeHandler) GetModelInfo(context.Context) (ModelInfo, error) {
	if f.err != nil {
		return ModelInfo{}, f.err
	}
	return f.info, nil
}

func (f fakeHandler) EmbedTexts(_ context.Context, role Role, texts []string) ([][]float32, error) {
	if f.err != nil {
		return nil, f.err
	}
	if f.roles != nil {
		*f.roles = append(*f.roles, role)
	}
	vectors := make([][]float32, len(texts))
	for i := range texts {
		vectors[i] = []float32{float32(i), float32(len(texts[i]))}
	}
	return vectors, nil
}

func TestClient_RoundTripsModelInfoAndVectorsThroughTheHandler(t *testing.T) {
	want := ModelInfo{Name: "small", Dimensions: 2, MaxTokens: 512, Pooling: "mean", From: "hub", Concurrency: 3, FloorHint: 0.25, MinChunkLength: 40, Handling: "1a2b3c4d"}
	mux := http.NewServeMux()
	mux.Handle(NewHandler(fakeHandler{info: want}))
	web := httptest.NewServer(mux)
	defer web.Close()
	client := NewClient(web.Client(), web.URL)

	info, err := client.GetModelInfo(t.Context())
	if err != nil || info != want {
		t.Fatalf("GetModelInfo = %+v, %v; want %+v", info, err, want)
	}
	vectors, err := client.EmbedTexts(t.Context(), RoleDocument, []string{"a", "bcd"})
	if err != nil || !reflect.DeepEqual(vectors, [][]float32{{0, 1}, {1, 3}}) {
		t.Fatalf("EmbedTexts = %v, %v", vectors, err)
	}
}

func TestClient_CarriesTheRoleOfTheTextsToTheHandler(t *testing.T) {
	var roles []Role
	mux := http.NewServeMux()
	mux.Handle(NewHandler(fakeHandler{roles: &roles}))
	web := httptest.NewServer(mux)
	defer web.Close()
	client := NewClient(web.Client(), web.URL)

	for _, role := range []Role{RoleQuery, RoleDocument} {
		if _, err := client.EmbedTexts(t.Context(), role, []string{"words"}); err != nil {
			t.Fatalf("EmbedTexts(%v): %v", role, err)
		}
	}
	if want := []Role{RoleQuery, RoleDocument}; !reflect.DeepEqual(roles, want) {
		t.Errorf("handler got roles %v, want %v", roles, want)
	}
}

func TestReadRoleOfARequestWithNoRoleIsDocument(t *testing.T) {
	if got := readRole(v1.EmbedRole_EMBED_ROLE_UNSPECIFIED); got != RoleDocument {
		t.Errorf("readRole(unspecified) = %v, want RoleDocument", got)
	}
	if got := writeRole(RoleDocument); got != v1.EmbedRole_EMBED_ROLE_DOCUMENT {
		t.Errorf("writeRole(RoleDocument) = %v, want EMBED_ROLE_DOCUMENT", got)
	}
}

func TestClient_PropagatesHandlerErrors(t *testing.T) {
	wantErr := errors.New("model failed")
	ad := &adapter{h: fakeHandler{err: wantErr}}
	if _, err := ad.GetEmbeddingModel(t.Context(), connect.NewRequest(&v1.GetEmbeddingModelRequest{})); !errors.Is(err, wantErr) {
		t.Fatalf("adapter.GetEmbeddingModel err = %v, want %v", err, wantErr)
	}
	if _, err := ad.EmbedBatch(t.Context(), connect.NewRequest(&v1.EmbedBatchRequest{})); !errors.Is(err, wantErr) {
		t.Fatalf("adapter.EmbedBatch err = %v, want %v", err, wantErr)
	}

	mux := http.NewServeMux()
	mux.Handle(NewHandler(fakeHandler{err: wantErr}))
	web := httptest.NewServer(mux)
	defer web.Close()
	client := NewClient(web.Client(), web.URL)

	if _, err := client.GetModelInfo(t.Context()); err == nil {
		t.Fatal("expected error from GetModelInfo")
	}
	if _, err := client.EmbedTexts(t.Context(), RoleDocument, []string{"a"}); err == nil {
		t.Fatal("expected error from EmbedTexts")
	}
}

func TestClient_RoundTripsTheChunkTokensOfTheModelAndZeroWhenNoneIsNamed(t *testing.T) {
	for _, want := range []int32{200, 0} {
		mux := http.NewServeMux()
		mux.Handle(NewHandler(fakeHandler{info: ModelInfo{Name: "small", Dimensions: 2, MaxTokens: 512, ChunkTokens: want}}))
		web := httptest.NewServer(mux)
		client := NewClient(web.Client(), web.URL)

		info, err := client.GetModelInfo(t.Context())
		web.Close()
		if err != nil || info.ChunkTokens != want {
			t.Errorf("GetModelInfo().ChunkTokens = %d, %v; want %d", info.ChunkTokens, err, want)
		}
	}
}

func TestWriteModelInfoWritesChunkTokensToTheWire(t *testing.T) {
	if got := writeModelInfo(ModelInfo{ChunkTokens: 200}).GetChunkTokens(); got != 200 {
		t.Errorf("wire chunk_tokens = %d, want 200", got)
	}
}

// Client calls the embedding service of a plugin.
type Client struct {
	rpc numenv1connect.EmbeddingServiceClient
}

// NewClient creates a client of the embedding service served at baseURL.
func NewClient(httpClient connect.HTTPClient, baseURL string, opts ...connect.ClientOption) *Client {
	return &Client{rpc: numenv1connect.NewEmbeddingServiceClient(httpClient, baseURL, opts...)}
}

// GetModelInfo returns the description of the model the plugin serves.
func (c *Client) GetModelInfo(ctx context.Context) (ModelInfo, error) {
	res, err := c.rpc.GetEmbeddingModel(ctx, connect.NewRequest(&v1.GetEmbeddingModelRequest{}))
	if err != nil {
		return ModelInfo{}, err
	}
	return readModelInfo(res.Msg), nil
}

// EmbedTexts returns one vector per text, in order, made for the role.
func (c *Client) EmbedTexts(ctx context.Context, role Role, texts []string) ([][]float32, error) {
	res, err := c.rpc.EmbedBatch(ctx, connect.NewRequest(&v1.EmbedBatchRequest{Texts: texts, Role: writeRole(role)}))
	if err != nil {
		return nil, err
	}
	return readVectors(res.Msg), nil
}

func readModelInfo(res *v1.GetEmbeddingModelResponse) ModelInfo {
	return ModelInfo{
		Name:           res.GetName(),
		Dimensions:     res.GetDimensions(),
		MaxTokens:      res.GetMaxTokens(),
		Pooling:        res.GetPooling(),
		From:           res.GetFrom(),
		Concurrency:    res.GetConcurrency(),
		FloorHint:      res.GetFloorHint(),
		MinChunkLength: res.GetMinChunkLength(),
		ChunkTokens:    res.GetChunkTokens(),
		Handling:       res.GetHandling(),
	}
}

func writeRole(role Role) v1.EmbedRole {
	if role == RoleQuery {
		return v1.EmbedRole_EMBED_ROLE_QUERY
	}
	return v1.EmbedRole_EMBED_ROLE_DOCUMENT
}

func readVectors(res *v1.EmbedBatchResponse) [][]float32 {
	vectors := make([][]float32, len(res.GetVectors()))
	for i, one := range res.GetVectors() {
		vectors[i] = one.GetValues()
	}
	return vectors
}
