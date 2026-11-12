package serving

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/jiva-studio/numen/plugins/engines-embeddings/backend/domain/model"
	"github.com/jiva-studio/numen/plugins/engines-embeddings/backend/internal/adapter/profilefiles"
)

func getReportedChunkTokens(t *testing.T, srv *Server) int {
	t.Helper()
	info, err := srv.GetModelInfo(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	return info.ChunkTokens
}

func TestGetModelInfoReportsTheDefaultChunkTokensOfALocalModelOf512(t *testing.T) {
	srv := newEmbeddingServer()
	srv.updateConfig(map[string]string{"source": "local", "model": model.DefaultModel})
	info, err := srv.GetModelInfo(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	if info.MaxTokens != 512 || info.ChunkTokens != 256 {
		t.Errorf("max tokens %d, chunk tokens %d; want 512 and 256", info.MaxTokens, info.ChunkTokens)
	}
}

func TestGetModelInfoReportsTheDefaultChunkTokensBeforeTheHostConfiguresThePlugin(t *testing.T) {
	if got := getReportedChunkTokens(t, newEmbeddingServer()); got != 256 {
		t.Errorf("chunk tokens before configure = %d, want 256", got)
	}
}

func TestGetModelInfoNeverReportsMoreChunkTokensThanTheModelReads(t *testing.T) {
	srv := newEmbeddingServer()
	srv.updateConfig(map[string]string{"source": "local", "model": model.DefaultModel, "chunkTokens": "1000"})
	if got := getReportedChunkTokens(t, srv); got != 512 {
		t.Errorf("chunk tokens for 1000 on a model of 512 = %d, want 512", got)
	}
}

func TestGetModelInfoReportsTheRequestedChunkTokensWithinTheBounds(t *testing.T) {
	srv := newEmbeddingServer()
	for value, want := range map[string]int{"abc": 256, "0": 16, "15": 16, "300": 300, "256.9": 256} {
		srv.updateConfig(map[string]string{"source": "local", "model": model.DefaultModel, "chunkTokens": value})
		if got := getReportedChunkTokens(t, srv); got != want {
			t.Errorf("configured %q: reported %d, want %d", value, got, want)
		}
	}
}

func TestAModelOfALowerLimitLowersChunkTokensAndTheOldModelRestoresThem(t *testing.T) {
	srv := newEmbeddingServer()
	dir := t.TempDir()
	srv.profileSrc = profilefiles.NewDirectory(dir)
	values := map[string]string{"source": "local", "model": model.DefaultModel, "chunkTokens": "400"}

	srv.updateConfig(values)
	first := getReportedChunkTokens(t, srv)
	writeFile(t, filepath.Join(dir, "small.json"), `{"id":"Xenova/multilingual-e5-small","maxTokens":256}`)
	srv.updateConfig(map[string]string{"source": "local", "model": model.DefaultModel})
	lowered := getReportedChunkTokens(t, srv)
	if err := os.Remove(filepath.Join(dir, "small.json")); err != nil {
		t.Fatal(err)
	}
	srv.updateConfig(map[string]string{"source": "local", "model": model.DefaultModel})
	restored := getReportedChunkTokens(t, srv)

	if first != 400 || lowered != 256 || restored != 400 {
		t.Errorf("chunk tokens = %d, %d, %d; want 400, 256, 400", first, lowered, restored)
	}
}

func TestAChangeOfChunkTokensDoesNotChangeTheModelNorItsHandling(t *testing.T) {
	srv := newEmbeddingServer()
	srv.updateConfig(map[string]string{"source": "local", "model": model.DefaultModel, "chunkTokens": "256"})
	before, err := srv.GetModelInfo(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	srv.updateConfig(map[string]string{"source": "local", "model": model.DefaultModel, "chunkTokens": "128"})
	after, err := srv.GetModelInfo(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	if after.ChunkTokens != 128 || before.ChunkTokens != 256 {
		t.Fatalf("chunk tokens = %d then %d, want 256 then 128", before.ChunkTokens, after.ChunkTokens)
	}
	after.ChunkTokens = before.ChunkTokens
	if after != before {
		t.Errorf("model changed with the setting: %+v -> %+v", before, after)
	}
}

func TestTheMockModelReportsChunkTokensLimitedByItsMaxTokens(t *testing.T) {
	t.Setenv("NUMEN_MOCK_ML", "true")
	srv := newEmbeddingServer()
	srv.updateConfig(map[string]string{"chunkTokens": "300"})
	if got := getReportedChunkTokens(t, srv); got != 300 {
		t.Errorf("mock chunk tokens = %d, want 300", got)
	}
	srv.updateConfig(map[string]string{"chunkTokens": "2000"})
	if got := getReportedChunkTokens(t, srv); got != 512 {
		t.Errorf("mock chunk tokens for 2000 = %d, want its limit 512", got)
	}
}
