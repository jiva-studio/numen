package note_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/jiva-studio/numen/modules/libs/core/adapter/filesystem"
	"github.com/jiva-studio/numen/modules/libs/core/domain"
	"github.com/jiva-studio/numen/modules/libs/core/usecase/note"
)

func (c changing) replace() note.Replace {
	return note.Replace{
		Readers: filesystem.Readers{}, Writers: filesystem.Writers{}, Index: c.index,
	}
}

// What is asked for is replaced, and what is not asked for is the bytes it was.
func TestOnlyTheStretchAskedForIsReplaced(t *testing.T) {
	t.Parallel()
	c := changeable(t, map[string]string{
		"Aggressor.md": "# The aggressor\n\nA hedgehog is named.\n\nAnd nothing else.\n",
	})

	done, err := c.replace().Execute(t.Context(), c.vault, "Aggressor.md",
		"A hedgehog", "An axe", domain.FileRef{})
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
	if done.Stood != "A hedgehog" {
		t.Errorf("what stood there is reported as %q", done.Stood)
	}
	if done.Plainly {
		t.Error("a stretch that stood exactly is reported as read plainly")
	}
}

// The offsets answer where the new text now stands, so a window can draw it
// without being told anything else.
func TestAReplacementSaysWhereItLanded(t *testing.T) {
	t.Parallel()
	c := changeable(t, map[string]string{"Aggressor.md": "one two three\n"})

	done, err := c.replace().Execute(t.Context(), c.vault, "Aggressor.md", "two", "four", domain.FileRef{})
	if err != nil {
		t.Fatal(err)
	}
	if done.Stretch.From != 4 || done.Stretch.To != 8 {
		t.Errorf("landed at %d..%d, wanted 4..8", done.Stretch.From, done.Stretch.To)
	}
}

// The frontmatter is the person's, and a replacement in the prose is not a
// reason to touch it.
func TestTheFrontmatterSurvivesAReplacement(t *testing.T) {
	t.Parallel()
	front := "---\nkeep: 'this'   # and this\nid: 01J8XYZ\n---\n"
	c := changeable(t, map[string]string{"Aggressor.md": front + "\nA hedgehog.\n"})

	if _, err := c.replace().Execute(t.Context(), c.vault, "Aggressor.md",
		"A hedgehog", "An axe", domain.FileRef{}); err != nil {
		t.Fatal(err)
	}
	if body := c.read(t, "Aggressor.md"); !strings.HasPrefix(body, front) {
		t.Errorf("the frontmatter changed:\n%s", body)
	}
}

// A stretch standing twice does not say which was meant, and guessing at one
// is how the wrong half of a note is rewritten.
func TestAStretchStandingTwiceIsRefused(t *testing.T) {
	t.Parallel()
	was := "A foe advances.\n\nAnother foe advances.\n"
	c := changeable(t, map[string]string{"Aggressor.md": was})

	_, err := c.replace().Execute(t.Context(), c.vault, "Aggressor.md",
		"foe advances", "foe retreats", domain.FileRef{})
	var twice note.Twice
	if !errors.As(err, &twice) {
		t.Fatalf("want Twice, got %v", err)
	}
	if twice.Places != 2 {
		t.Errorf("counted %d places", twice.Places)
	}
	if body := c.read(t, "Aggressor.md"); body != was {
		t.Errorf("the refused replacement landed anyway:\n%s", body)
	}
}

// A stretch that is not there is answered with where a copy of it stopped
// agreeing, which is what tells a caller its copy is one character out.
func TestAStretchThatIsNotThereSaysWhereItDiverged(t *testing.T) {
	t.Parallel()
	c := changeable(t, map[string]string{"Aggressor.md": "the wrath of the advancing foe\n"})

	_, err := c.replace().Execute(t.Context(), c.vault, "Aggressor.md",
		"the wrath of the retreating foe", "nothing", domain.FileRef{})
	var nowhere note.Nowhere
	if !errors.As(err, &nowhere) {
		t.Fatalf("want Nowhere, got %v", err)
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
func TestAReplacementAlreadyInTheNoteIsSaidSo(t *testing.T) {
	t.Parallel()
	c := changeable(t, map[string]string{"Aggressor.md": "An axe is named.\n"})

	_, err := c.replace().Execute(t.Context(), c.vault, "Aggressor.md",
		"A hedgehog", "An axe", domain.FileRef{})
	if !errors.Is(err, note.ErrAlreadyWritten) {
		t.Fatalf("want ErrAlreadyWritten, got %v", err)
	}
}

// A person's editor writes the quotes and dashes; a program writing about that
// prose rarely reproduces them. The stretch is found, and the reading is said.
func TestPunctuationThatDiffersIsFoundAndReported(t *testing.T) {
	t.Parallel()
	c := changeable(t, map[string]string{
		"Aggressor.md": "Он сказал «да» — и ушёл.\n",
	})

	done, err := c.replace().Execute(t.Context(), c.vault, "Aggressor.md",
		"сказал \"да\" - и ушёл", "промолчал", domain.FileRef{})
	if err != nil {
		t.Fatal(err)
	}
	if !done.Plainly {
		t.Error("the reading was not reported")
	}
	if done.Stood != "сказал «да» — и ушёл" {
		t.Errorf("what stood there is reported as %q", done.Stood)
	}
	if body := c.read(t, "Aggressor.md"); !strings.HasSuffix(body, "Он промолчал.\n") {
		t.Errorf("the note reads:\n%q", body)
	}
}

// A file written with CRLF is a file the application is a guest in.
func TestACRLFNoteKeepsItsBreaks(t *testing.T) {
	t.Parallel()
	c := changeable(t, map[string]string{
		"Aggressor.md": "# The aggressor\r\n\r\nA hedgehog.\r\n",
	})

	if _, err := c.replace().Execute(t.Context(), c.vault, "Aggressor.md",
		"A hedgehog", "An axe", domain.FileRef{}); err != nil {
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
func TestAReplacedNoteTakesAnIdentifier(t *testing.T) {
	t.Parallel()
	c := changeable(t, map[string]string{"Aggressor.md": "A hedgehog.\n"})

	if _, err := c.replace().Execute(t.Context(), c.vault, "Aggressor.md",
		"A hedgehog", "An axe", domain.FileRef{}); err != nil {
		t.Fatal(err)
	}
	if body := c.read(t, "Aggressor.md"); !strings.HasPrefix(body, "---\nid: ") {
		t.Errorf("no identifier was written:\n%s", body)
	}
}

// The fingerprint a replacement answers with is what the next one presents, so
// a caller changing a note twice does not read it back in between.
func TestAReplacementFollowsAReplacementWithNoReadBetween(t *testing.T) {
	t.Parallel()
	c := changeable(t, map[string]string{"Aggressor.md": "one two three\n"})
	replacing := c.replace()

	first, err := replacing.Execute(t.Context(), c.vault, "Aggressor.md", "one", "ONE", domain.FileRef{})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := replacing.Execute(
		t.Context(), c.vault, "Aggressor.md", "three", "THREE", first.At); err != nil {
		t.Fatalf("the second replacement was refused: %v", err)
	}
	if body := c.read(t, "Aggressor.md"); !strings.Contains(body, "ONE two THREE") {
		t.Errorf("the note reads:\n%s", body)
	}
}
