package embedding

import (
	"context"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
)

type fakeHandler struct {
	info ModelInfo
}

func (f fakeHandler) GetModelInfo(context.Context) (ModelInfo, error) { return f.info, nil }

func (f fakeHandler) EmbedTexts(_ context.Context, texts []string) ([][]float32, error) {
	vectors := make([][]float32, len(texts))
	for i := range texts {
		vectors[i] = []float32{float32(i), float32(len(texts[i]))}
	}
	return vectors, nil
}

func TestClient_RoundTripsModelInfoAndVectorsThroughTheHandler(t *testing.T) {
	want := ModelInfo{Name: "small", Dimensions: 2, MaxTokens: 512, Pooling: "mean", From: "hub", Concurrency: 3, FloorHint: 0.25, MinChunkLength: 40}
	mux := http.NewServeMux()
	mux.Handle(NewHandler(fakeHandler{info: want}))
	web := httptest.NewServer(mux)
	defer web.Close()
	client := NewClient(web.Client(), web.URL)

	info, err := client.GetModelInfo(t.Context())
	if err != nil || info != want {
		t.Fatalf("GetModelInfo = %+v, %v; want %+v", info, err, want)
	}
	vectors, err := client.EmbedTexts(t.Context(), []string{"a", "bcd"})
	if err != nil || !reflect.DeepEqual(vectors, [][]float32{{0, 1}, {1, 3}}) {
		t.Fatalf("EmbedTexts = %v, %v", vectors, err)
	}
}
