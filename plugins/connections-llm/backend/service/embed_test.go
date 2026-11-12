package service

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"slices"
	"testing"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/connection"
	"github.com/jiva-studio/numen/plugins/connections-llm/backend/fakeopenai"
)

func TestEmbedSendsTheTextsAndReturnsOneVectorForEach(t *testing.T) {
	fake, address := startFake(t, fakeopenai.Config{APIKey: testKey, Dimensions: 6})
	texts := []string{"first", "second", "third"}

	vectors, err := newTestService(t).Embed(newContext(t), newCall("openai", address, testKey, "text-embedding-3-small"), texts)
	if err != nil {
		t.Fatal(err)
	}
	if len(vectors) != 3 {
		t.Fatalf("got %d vectors, want 3", len(vectors))
	}
	for i, text := range texts {
		if !slices.Equal(vectors[i], fake.Vector(text)) {
			t.Errorf("vector %d is not the vector of %q", i, text)
		}
	}
	requests := fake.Requests()
	if len(requests) != 1 || requests[0].Path != "/embeddings" || requests[0].Header.Get("Authorization") != "Bearer "+testKey {
		t.Fatalf("requests = %+v", requests)
	}
	var body struct {
		Model string   `json:"model"`
		Input []string `json:"input"`
	}
	if err := json.Unmarshal(requests[0].Body, &body); err != nil {
		t.Fatal(err)
	}
	if body.Model != "text-embedding-3-small" || !slices.Equal(body.Input, texts) {
		t.Errorf("body = %s", requests[0].Body)
	}
}

func TestEmbedOrdersTheVectorsByTheIndexInTheAnswer(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		body := `{"data":[{"index":1,"embedding":[0,1]},{"index":0,"embedding":[1,0]}]}`
		if _, err := w.Write([]byte(body)); err != nil {
			t.Error(err)
		}
	}))
	t.Cleanup(ts.Close)

	vectors, err := newTestService(t).Embed(newContext(t), newCall("openai-compatible", ts.URL, "", "m"), []string{"a", "b"})
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(vectors[0], []float32{1, 0}) || !slices.Equal(vectors[1], []float32{0, 1}) {
		t.Errorf("vectors = %v", vectors)
	}
}

func TestEmbedWithNoTextsSendsNothing(t *testing.T) {
	fake, address := startFake(t, fakeopenai.Config{})
	vectors, err := newTestService(t).Embed(newContext(t), newCall("openai-compatible", address, "", "m"), nil)
	if err != nil || len(vectors) != 0 {
		t.Errorf("vectors = %v, err = %v", vectors, err)
	}
	if len(fake.Requests()) != 0 {
		t.Errorf("the service received %d requests", len(fake.Requests()))
	}
}

func TestEmbedStopsOnAnAnswerThatDoesNotMatchTheTexts(t *testing.T) {
	bodies := map[string]string{
		"fewer vectors than texts": `{"data":[{"index":0,"embedding":[1,0]}]}`,
		"an empty vector":          `{"data":[{"index":0,"embedding":[]},{"index":1,"embedding":[1]}]}`,
		"a repeated index":         `{"data":[{"index":0,"embedding":[1]},{"index":0,"embedding":[1]}]}`,
		"an index out of range":    `{"data":[{"index":0,"embedding":[1]},{"index":5,"embedding":[1]}]}`,
		"not JSON":                 `<html>`,
	}
	for name, body := range bodies {
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			if _, err := w.Write([]byte(body)); err != nil {
				t.Error(err)
			}
		}))
		t.Cleanup(ts.Close)
		_, err := newTestService(t).Embed(newContext(t), newCall("openai-compatible", ts.URL, "", "m"), []string{"a", "b"})
		if err == nil || !connection.IsStop(err) {
			t.Errorf("%s: err = %v, want a stop", name, err)
		}
	}
}

func TestProbeEmbeddingWidthEmbedsOneTextAndReturnsItsWidth(t *testing.T) {
	fake, address := startFake(t, fakeopenai.Config{Dimensions: 12})
	width, err := newTestService(t).ProbeEmbeddingWidth(newContext(t), newCall("openai-compatible", address, "", "unlisted-model"))
	if err != nil {
		t.Fatal(err)
	}
	if width != 12 {
		t.Errorf("width = %d, want 12", width)
	}
	var body struct {
		Input []string `json:"input"`
	}
	if err := json.Unmarshal(fake.Requests()[0].Body, &body); err != nil || len(body.Input) != 1 || body.Input[0] == "" {
		t.Errorf("body = %s, err = %v", fake.Requests()[0].Body, err)
	}
}

func TestProbeEmbeddingWidthReportsTheFailureOfTheService(t *testing.T) {
	fake, address := startFake(t, fakeopenai.Config{})
	fake.Fail("/embeddings", http.StatusPaymentRequired, "")
	_, err := newTestService(t).ProbeEmbeddingWidth(newContext(t), newCall("openrouter", address, testKey, "m"))
	checkFailure(t, err, failureWant{isTryNext: true, reason: connection.ReasonPaymentRequired})
}

func TestTheProfileOfAKnownModelIsReadWithNoRequest(t *testing.T) {
	fake, address := startFake(t, fakeopenai.Config{})
	cases := map[string]struct {
		typeID, model string
		dimensions    int
	}{
		"openai small":            {"openai", "text-embedding-3-small", 1536},
		"openai large":            {"openai", "text-embedding-3-large", 3072},
		"openai ada":              {"openai", "text-embedding-ada-002", 1536},
		"openrouter with vendor":  {"openrouter", "openai/text-embedding-3-small", 1536},
		"openrouter bge":          {"openrouter", "baai/bge-m3", 1024},
		"openrouter qwen":         {"openrouter", "qwen/qwen3-embedding-8b", 4096},
		"compatible case ignored": {"openai-compatible", "Text-Embedding-3-Large", 3072},
	}
	for name, one := range cases {
		profile, err := newTestService(t).GetModelProfile(newContext(t), newCall(one.typeID, address, "", one.model))
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		if !profile.IsKnown || profile.Dimensions != one.dimensions || profile.MaxTokens <= 0 || profile.Pooling == "" {
			t.Errorf("%s: profile = %+v", name, profile)
		}
	}
	if len(fake.Requests()) != 0 {
		t.Errorf("the service received %d requests", len(fake.Requests()))
	}
}

func TestTheProfileOfAnUnknownModelIsMarkedUnknown(t *testing.T) {
	_, address := startFake(t, fakeopenai.Config{})
	profile, err := newTestService(t).GetModelProfile(newContext(t), newCall("openai-compatible", address, "", "nomic-embed-text"))
	if err != nil {
		t.Fatal(err)
	}
	if profile.IsKnown || profile.Dimensions != 0 {
		t.Errorf("profile = %+v, want an unknown profile", profile)
	}
}

func TestTheSourceOfAProfileNamesTheTypeAndTheAddressAndNeverTheKey(t *testing.T) {
	_, address := startFake(t, fakeopenai.Config{})
	call := newCall("openai-compatible", address+"?token=abc#frag", testKey, "text-embedding-3-small")
	profile, err := newTestService(t).GetModelProfile(newContext(t), call)
	if err != nil {
		t.Fatal(err)
	}
	want := "md.numen.connections.llm/openai-compatible " + address
	if profile.Source != want {
		t.Errorf("source = %q, want %q", profile.Source, want)
	}
}
