package transcription

import (
	"os"
	"testing"
)

// The pieces are joined as they stand, and the mark a word opens with is the
// space before it.
func TestPiecesJoinIntoWords(t *testing.T) {
	p := pieces{"<unk>", "▁what", "▁the", "▁page", " say", "s", "<blk>"}
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
	if len(said) != 4 || said[1] != "▁a" || said.getIndex("<blk>") != 3 {
		t.Errorf("the file names %v", said)
	}
}
