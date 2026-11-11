package serving

import "testing"

// The model is read while Configure changes it.
func TestTheModelIsReadWhileConfigureChangesIt(t *testing.T) {
	srv := newEmbeddingServer()
	isDone := make(chan struct{})
	go func() {
		defer close(isDone)
		for range 200 {
			srv.updateConfig(map[string]string{"model": "bge-m3"}, nil)
		}
	}()

	for range 200 {
		if _, err := srv.GetModelInfo(); err != nil {
			t.Fatal(err)
		}
	}
	<-isDone
}

// A dimension or a token limit the model answer cannot hold is not taken from the environment.
func TestAnOverrideAboveWhatTheAnswerContainsIsIgnored(t *testing.T) {
	want := newEmbeddingServer().modelInfo
	t.Setenv("NUMEN_EMBEDDING_DIMENSIONS", "4294967296")
	t.Setenv("NUMEN_EMBEDDING_MAX_TOKENS", "4294967296")

	got := newEmbeddingServer().modelInfo

	if got.Dimensions != want.Dimensions || got.MaxTokens != want.MaxTokens {
		t.Fatalf("dimensions %d and tokens %d, want %d and %d", got.Dimensions, got.MaxTokens, want.Dimensions, want.MaxTokens)
	}
}
