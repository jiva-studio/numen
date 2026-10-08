package transcription

import (
	"fmt"
	"math/rand"
	"os"
	"strconv"
	"strings"
	"testing"
)

// The pieces are joined as they stand, and the mark a word opens with is the
// space before it.
func TestPiecesJoinIntoWords(t *testing.T) {
	p := pieces{"<unk>", " what", " the", " page", " say", "s", "<blk>"}
	if got := p.text([]int{1, 2, 3, 4, 5}); got != "what the page says" {
		t.Errorf("the tokens say %q", got)
	}
	if got := p.text(nil); got != "" {
		t.Errorf("no tokens say %q", got)
	}
	// A number outside the file is a token this transcriber cannot write.
	if got := p.text([]int{1, 900}); got != "what" {
		t.Errorf("an unknown token said %q", got)
	}
	if p.getIndex("<blk>") != 6 || p.getIndex("nothing") != -1 {
		t.Errorf("the blank stands at %d", p.getIndex("<blk>"))
	}
}

func TestTokensReadsTheFilePublishedBesideAModel(t *testing.T) {
	at := t.TempDir() + "/tokens.txt"
	if err := os.WriteFile(at, []byte("<unk> 0\n▁a 1\n▁b 2\n<blk> 3\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	said, err := tokens(at)
	if err != nil {
		t.Fatal(err)
	}
	if len(said) != 4 || said[1] != " a" || said.getIndex("<blk>") != 3 {
		t.Errorf("the file names %v", said)
	}
}

func decodeReference(p pieces, said []int) string {
	var out strings.Builder
	for _, token := range said {
		if token < 0 || token >= len(p) {
			continue
		}
		out.WriteString(p[token])
	}
	return strings.TrimSpace(strings.ReplaceAll(out.String(), wordMark, " "))
}

func createVocabularyFile(t testing.TB, words []string) string {
	t.Helper()
	var file strings.Builder
	for i, word := range words {
		fmt.Fprintf(&file, "%s %d\n", word, i)
	}
	at := t.TempDir() + "/tokens.txt"
	if err := os.WriteFile(at, []byte(file.String()), 0o644); err != nil {
		t.Fatal(err)
	}
	return at
}

var vocabularyWords = []string{
	"<unk>", "▁what", "▁the", "▁page", "▁say", "s", "▁", "▁▁", "a▁b",
	"▁é", "ing", "▁world", "<blk>",
}

func TestTokensTextMatchesReference(t *testing.T) {
	loaded, err := tokens(createVocabularyFile(t, vocabularyWords))
	if err != nil {
		t.Fatal(err)
	}
	rng := rand.New(rand.NewSource(362))
	for range 5000 {
		said := make([]int, rng.Intn(24))
		for i := range said {
			said[i] = rng.Intn(len(vocabularyWords)+4) - 2
		}
		want := decodeReference(pieces(vocabularyWords), said)
		if got := loaded.text(said); got != want {
			t.Fatalf("tokens %v say %q, want %q", said, got, want)
		}
	}
}

func BenchmarkTokensText(b *testing.B) {
	words := make([]string, 1024)
	for i := range words {
		switch i % 4 {
		case 0:
			words[i] = "▁word" + strconv.Itoa(i)
		case 1:
			words[i] = "ing"
		case 2:
			words[i] = "▁the"
		default:
			words[i] = "s"
		}
	}
	loaded, err := tokens(createVocabularyFile(b, words))
	if err != nil {
		b.Fatal(err)
	}
	said := make([]int, 200)
	for i := range said {
		said[i] = (i * 7) % len(words)
	}
	b.ReportAllocs()
	for b.Loop() {
		_ = loaded.text(said)
	}
}
