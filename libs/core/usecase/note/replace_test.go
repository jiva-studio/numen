package note_test

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/jiva-studio/numen/libs/core/internal/adapter/filesystem/vaultreader"
	"github.com/jiva-studio/numen/libs/core/internal/adapter/filesystem/vaultwriter"
	"github.com/jiva-studio/numen/libs/core/usecase/note"
	"github.com/jiva-studio/numen/libs/notes/domain"
)

func (c changing) replace() note.Replace {
	return note.NewReplace(vaultreader.Readers{}, vaultwriter.VaultWriters{}, c.index, time.Now)
}

// What is asked for is replaced, and what is not asked for is the bytes it was.
func TestReplace_ChangesOnlyRequestedSpan(t *testing.T) {
	t.Parallel()
	c := openChanging(t, map[string]string{
		"Aggressor.md": "# The aggressor\n\nA hedgehog is named.\n\nAnd nothing else.\n",
	})

	isDone, err := c.replace().Execute(t.Context(), c.vault, "Aggressor.md",
		"A hedgehog", "An axe", domain.FileStat{})
	if err != nil {
		t.Fatal(err)
	}

	body := c.read(t, "Aggressor.md")
	if !strings.Contains(body, "An axe is named.") {
		t.Errorf("the stretch was not replaced:\n%s", body)
	}
	if !strings.Contains(body, "# The aggressor\n") || !strings.Contains(body, "And nothing else.\n") {
		t.Errorf("what was not asked for changed:\n%s", body)
	}
	if isDone.Matched != "A hedgehog" {
		t.Errorf("what stood there is reported as %q", isDone.Matched)
	}
	if isDone.IsPlain {
		t.Error("a stretch that stood exactly is reported as read plainly")
	}
}

// The offsets answer where the new text now stands, so a window can draw it
// without being told anything else.
func TestAReplacementReturnsWhereItLanded(t *testing.T) {
	t.Parallel()
	c := openChanging(t, map[string]string{"Aggressor.md": "one two three\n"})

	isDone, err := c.replace().Execute(t.Context(), c.vault, "Aggressor.md", "two", "four", domain.FileStat{})
	if err != nil {
		t.Fatal(err)
	}
	if isDone.Range.Start != 4 || isDone.Range.End != 8 {
		t.Errorf("landed at %d..%d, wanted 4..8", isDone.Range.Start, isDone.Range.End)
	}
}

// The frontmatter is the person's, and a replacement in the prose is not a
// reason to touch it.
func TestTheFrontmatterSurvivesAReplacement(t *testing.T) {
	t.Parallel()
	front := "---\nkeep: 'this'   # and this\nid: 01J8XYZ\n---\n"
	c := openChanging(t, map[string]string{"Aggressor.md": front + "\nA hedgehog.\n"})

	if _, err := c.replace().Execute(t.Context(), c.vault, "Aggressor.md",
		"A hedgehog", "An axe", domain.FileStat{}); err != nil {
		t.Fatal(err)
	}
	if body := c.read(t, "Aggressor.md"); !strings.HasPrefix(body, front) {
		t.Errorf("the frontmatter changed:\n%s", body)
	}
}

// A span standing twice does not say which was meant, and guessing at one
// is how the wrong half of a note is rewritten.
func TestASpanActiveTwiceIsRefused(t *testing.T) {
	t.Parallel()
	was := "A foe advances.\n\nAnother foe advances.\n"
	c := openChanging(t, map[string]string{"Aggressor.md": was})

	_, err := c.replace().Execute(t.Context(), c.vault, "Aggressor.md",
		"foe advances", "foe retreats", domain.FileStat{})
	var ambiguous note.AmbiguousSpan
	if !errors.As(err, &ambiguous) {
		t.Fatalf("want AmbiguousSpan, got %v", err)
	}
	if ambiguous.Places != 2 {
		t.Errorf("counted %d places", ambiguous.Places)
	}
	if body := c.read(t, "Aggressor.md"); body != was {
		t.Errorf("the rejected replacement landed anyway:\n%s", body)
	}
}

// A span that is not there is answered with where a copy of it stopped
// agreeing, which is what tells a caller its copy is one character out.
func TestASpanThatIsNotThereReturnsWhereItDiverged(t *testing.T) {
	t.Parallel()
	c := openChanging(t, map[string]string{"Aggressor.md": "the wrath of the advancing foe\n"})

	_, err := c.replace().Execute(t.Context(), c.vault, "Aggressor.md",
		"the wrath of the retreating foe", "nothing", domain.FileStat{})
	var nowhere note.MissingSpan
	if !errors.As(err, &nowhere) {
		t.Fatalf("want MissingSpan, got %v", err)
	}
	if !strings.HasPrefix(nowhere.Matched, "the wrath of the ") {
		t.Errorf("what matched is reported as %q", nowhere.Matched)
	}
	if !strings.Contains(nowhere.Instead, "advancing") {
		t.Errorf("what stands there is reported as %q", nowhere.Instead)
	}
}

// A replacement already in the note and an original that is gone is a write
// that landed. Saying so is what stops it landing twice.
func TestReplace_ReportsReplacementAlreadyPresent(t *testing.T) {
	t.Parallel()
	c := openChanging(t, map[string]string{"Aggressor.md": "An axe is named.\n"})

	_, err := c.replace().Execute(t.Context(), c.vault, "Aggressor.md",
		"A hedgehog", "An axe", domain.FileStat{})
	if !errors.Is(err, note.ErrAlreadyWritten) {
		t.Fatalf("want ErrAlreadyWritten, got %v", err)
	}
}

// A person's editor writes the quotes and dashes; a program writing about that
// prose rarely reproduces them. The stretch is found, and the reading is said.
func TestPunctuationThatDiffersIsFoundAndReported(t *testing.T) {
	t.Parallel()
	c := openChanging(t, map[string]string{
		"Aggressor.md": "Он сказал «да» — и ушёл.\n",
	})

	isDone, err := c.replace().Execute(t.Context(), c.vault, "Aggressor.md",
		"сказал \"да\" - и ушёл", "промолчал", domain.FileStat{})
	if err != nil {
		t.Fatal(err)
	}
	if !isDone.IsPlain {
		t.Error("the reading was not reported")
	}
	if isDone.Matched != "сказал «да» — и ушёл" {
		t.Errorf("what stood there is reported as %q", isDone.Matched)
	}
	if body := c.read(t, "Aggressor.md"); !strings.HasSuffix(body, "Он промолчал.\n") {
		t.Errorf("the note reads:\n%q", body)
	}
}

// A file written with CRLF is a file the application is a guest in.
func TestACRLFNoteKeepsItsBreaks(t *testing.T) {
	t.Parallel()
	c := openChanging(t, map[string]string{
		"Aggressor.md": "# The aggressor\r\n\r\nA hedgehog.\r\n",
	})

	if _, err := c.replace().Execute(t.Context(), c.vault, "Aggressor.md",
		"A hedgehog", "An axe", domain.FileStat{}); err != nil {
		t.Fatal(err)
	}
	body := c.read(t, "Aggressor.md")
	if strings.Contains(strings.ReplaceAll(body, "\r\n", ""), "\n") {
		t.Errorf("a bare newline was written into a CRLF note:\n%q", body)
	}
	if !strings.Contains(body, "An axe.\r\n") {
		t.Errorf("the note reads:\n%q", body)
	}
}

// A replacement is the application changing what is in a note, so the note
// takes an identifier if it has none.
func TestAReplacedNoteConsumesAnIdentifier(t *testing.T) {
	t.Parallel()
	c := openChanging(t, map[string]string{"Aggressor.md": "A hedgehog.\n"})

	if _, err := c.replace().Execute(t.Context(), c.vault, "Aggressor.md",
		"A hedgehog", "An axe", domain.FileStat{}); err != nil {
		t.Fatal(err)
	}
	if body := c.read(t, "Aggressor.md"); !strings.HasPrefix(body, "---\nid: ") {
		t.Errorf("no identifier was written:\n%s", body)
	}
}

// The file stat a replacement answers with is what the next one presents, so
// a caller changing a note twice does not read it back in between.
func TestAReplacementFollowsAReplacementWithNoReadBetween(t *testing.T) {
	t.Parallel()
	c := openChanging(t, map[string]string{"Aggressor.md": "one two three\n"})
	replacing := c.replace()

	first, err := replacing.Execute(t.Context(), c.vault, "Aggressor.md", "one", "ONE", domain.FileStat{})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := replacing.Execute(
		t.Context(), c.vault, "Aggressor.md", "three", "THREE", first.FileStat); err != nil {
		t.Fatalf("the second replacement failed: %v", err)
	}
	if body := c.read(t, "Aggressor.md"); !strings.Contains(body, "ONE two THREE") {
		t.Errorf("the note reads:\n%s", body)
	}
}
