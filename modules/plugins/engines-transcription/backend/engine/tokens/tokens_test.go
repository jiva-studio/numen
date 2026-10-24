package tokens

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
	p := Pieces{"<unk>", " what", " the", " page", " say", "s", "<blk>"}
	if got := p.GetText([]int{1, 2, 3, 4, 5}); got != "what the page says" {
		t.Errorf("the tokens say %q", got)
	}
	if got := p.GetText(nil); got != "" {
		t.Errorf("no tokens say %q", got)
	}
	// A number outside the file is a token this transcriber cannot write.
	if got := p.GetText([]int{1, 900}); got != "what" {
		t.Errorf("an unknown token said %q", got)
	}
	if p.GetIndex("<blk>") != 6 || p.GetIndex("nothing") != -1 {
		t.Errorf("blank token is at %d", p.GetIndex("<blk>"))
	}
}

func TestTokensReadsTheFilePublishedAdjacentAModel(t *testing.T) {
	pos := t.TempDir() + "/tokens.txt"
	if err := os.WriteFile(pos, []byte("<unk> 0\n▁a 1\n▁b 2\n<blk> 3\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	response, err := Read(pos)
	if err != nil {
		t.Fatal(err)
	}
	if len(response) != 4 || response[1] != " a" || response.GetIndex("<blk>") != 3 {
		t.Errorf("the file names %v", response)
	}
}

func decodeReference(p Pieces, response []int) string {
	var out strings.Builder
	for _, token := range response {
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
	pos := t.TempDir() + "/tokens.txt"
	if err := os.WriteFile(pos, []byte(file.String()), 0o644); err != nil {
		t.Fatal(err)
	}
	return pos
}

var vocabularyWords = []string{
	"<unk>", "▁what", "▁the", "▁page", "▁say", "s", "▁", "▁▁", "a▁b",
	"▁é", "ing", "▁world", "<blk>",
}

func TestTokensTextMatchesReference(t *testing.T) {
	loaded, err := Read(createVocabularyFile(t, vocabularyWords))
	if err != nil {
		t.Fatal(err)
	}
	rng := rand.New(rand.NewSource(362))
	for range 5000 {
		response := make([]int, rng.Intn(24))
		for i := range response {
			response[i] = rng.Intn(len(vocabularyWords)+4) - 2
		}
		want := decodeReference(Pieces(vocabularyWords), response)
		if got := loaded.GetText(response); got != want {
			t.Fatalf("tokens %v say %q, want %q", response, got, want)
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
	loaded, err := Read(createVocabularyFile(b, words))
	if err != nil {
		b.Fatal(err)
	}
	response := make([]int, 200)
	for i := range response {
		response[i] = (i * 7) % len(words)
	}
	b.ReportAllocs()
	for b.Loop() {
		_ = loaded.GetText(response)
	}
}
