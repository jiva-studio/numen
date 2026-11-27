package engine

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/hosttest"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/jiva-studio/numen/libs/shared/onnx"
	"github.com/jiva-studio/numen/libs/shared/tokenizer"
	"github.com/jiva-studio/numen/plugins/engines-embeddings/backend/domain/model"
)

type tokensFixture struct {
	Tokenizers string `json:"tokenizers"`
	Repo       string `json:"repo"`
	SHA256     string `json:"sha256"`
	Cases      []struct {
		Role string `json:"role"`
		Text string `json:"text"`
		IDs  []int  `json:"ids"`
	} `json:"cases"`
}

func readTokensFixture(t *testing.T) tokensFixture {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join("testdata", "e5-tokens.json"))
	if err != nil {
		t.Fatal(err)
	}
	var fixture tokensFixture
	if err := json.Unmarshal(raw, &fixture); err != nil {
		t.Fatal(err)
	}
	if len(fixture.Cases) == 0 {
		t.Fatal("e5-tokens.json holds no case: a comparison against nothing")
	}
	return fixture
}

func loadTestTokenizer(t *testing.T, path string) *tokenizer.Tokenizer {
	t.Helper()
	encoder, err := tokenizer.Load(path)
	if err != nil {
		t.Fatalf("Load(%s): %v", path, err)
	}
	return encoder
}

func getE5TokenizerPath(t *testing.T) string {
	t.Helper()
	root := os.Getenv(envTestModels)
	if root == "" {
		t.Skipf("set %s to a folder of model files to run the token test", envTestModels)
	}
	path := filepath.Join(root, "Xenova--multilingual-e5-small", model.TokenizerFile)
	if _, err := os.Stat(path); err != nil {
		t.Skipf("%s holds no tokenizer.json of Xenova/multilingual-e5-small", root)
	}
	return path
}

// TestE5TokensMatchHuggingFace reads each text as the plugin does, behind the prefix of the E5 profile
// and cut to the window of 512, and compares the ids with those Python `tokenizers` gave for the same
// prefixed text truncated to 512 (testdata/record_e5_tokens.py).
func TestE5TokensMatchHuggingFace(t *testing.T) {
	path := getE5TokenizerPath(t)
	fixture := readTokensFixture(t)
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if sum := sha256.Sum256(raw); hex.EncodeToString(sum[:]) != fixture.SHA256 {
		t.Fatalf("%s is not the file the reference was recorded from (sha256 %s)", path, fixture.SHA256)
	}
	profiles, err := model.GetShippedProfiles()
	if err != nil {
		t.Fatal(err)
	}
	profile, err := profiles.GetLocal(fixture.Repo)
	if err != nil {
		t.Fatal(err)
	}
	handling := profile.GetHandling()
	encoder := loadTestTokenizer(t, path)
	e := &Embedder{encoder: encoder, window: profile.GetTokens()}

	roles := map[string]model.Role{"query": model.RoleQuery, "document": model.RoleDocument}
	isCut := false
	for _, c := range fixture.Cases {
		role, ok := roles[c.Role]
		if !ok {
			t.Fatalf("unknown role %q", c.Role)
		}
		texts := handling.Apply(role, []string{c.Text})
		got := e.tokenize(texts)[0]
		if !slices.Equal(got, c.IDs) {
			t.Errorf("%s %q: ids %v, want %v", c.Role, shortenText(c.Text), got, c.IDs)
		}
		isCut = isCut || len(c.IDs) == e.window
	}
	if !isCut {
		t.Error("no case fills the window of 512: the cut is not compared")
	}
	if pad, ok := encoder.GetPadID(); !ok || pad != 1 {
		t.Errorf("GetPadID() = %d, %v, want 1, true", pad, ok)
	}
}

func TestEmbedderReportsTokenizerVersionForUnigramTokenizer(t *testing.T) {
	e := &Embedder{encoder: loadTestTokenizer(t, getE5TokenizerPath(t)), window: model.MaxChunkTokens}
	if e.GetTokenizerVersion() == "" {
		t.Error("an embedder that reads E5 with the Unigram tokenizer reports no tokenizer version")
	}
}

func TestEmbedderReportsNoTokenizerVersionForDelegatedBPETokenizer(t *testing.T) {
	e := &Embedder{encoder: loadTestTokenizer(t, filepath.Join("testdata", "bpe.json")), window: model.MaxChunkTokens}
	if got := e.GetTokenizerVersion(); got != "" {
		t.Errorf("an embedder that reads a BPE model reports tokenizer version %q, want none", got)
	}
}

func shortenText(text string) string {
	runes := []rune(text)
	if len(runes) <= 60 {
		return text
	}
	return string(runes[:60]) + "…"
}

// TestAnOpenWhoseTokenizerIsNotTheKindOfItsProfileFails gives a profile that names the Unigram
// tokenizer a folder whose tokenizer.json is a BPE model. The open fails before the runtime loads the
// model file, which here is no model at all.
func TestAnOpenWhoseTokenizerIsNotTheKindOfItsProfileFails(t *testing.T) {
	dir := t.TempDir()
	raw, err := os.ReadFile(filepath.Join("testdata", "bpe.json"))
	if err != nil {
		t.Fatal(err)
	}
	tokenizerPath := filepath.Join(dir, model.TokenizerFile)
	if err := os.WriteFile(tokenizerPath, raw, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "model.onnx"), []byte("not a model"), 0o600); err != nil {
		t.Fatal(err)
	}
	profiles, err := model.ReadProfiles([]byte(
		`{"id":"owner/mismatch","dims":8,"maxTokens":8,"pooling":"mean","onnxFile":"onnx/model.onnx","tokenizer":"unigram"}`,
	))
	if err != nil {
		t.Fatal(err)
	}
	info, local := model.NewConfig(profiles[0])
	local.Dir = dir

	emb, err := Open(t.Context(), onnx.NewLibrary(), info, local, Fetch{}, hosttest.NewQueue().EnqueueWork)
	if err == nil {
		if closeErr := emb.Close(); closeErr != nil {
			t.Errorf("Close() = %v", closeErr)
		}
		t.Fatal("a BPE tokenizer opened for a profile that names the Unigram tokenizer")
	}
	if !strings.Contains(err.Error(), "owner/mismatch") || !strings.Contains(err.Error(), tokenizerPath) {
		t.Errorf("error = %v, want one that names owner/mismatch and %s", err, tokenizerPath)
	}
}
