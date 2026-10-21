package note_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/jiva-studio/numen/modules/libs/core/port"
	"github.com/jiva-studio/numen/modules/libs/core/usecase/note"
	"github.com/jiva-studio/numen/modules/libs/notes/domain"
	"github.com/jiva-studio/numen/modules/libs/notes/markdown"
)

func TestAMissingSpanSaysWhatItIs(t *testing.T) {
	t.Parallel()

	if got, want := (note.MissingSpan{}).Error(), "no part of this span is in the note"; got != want {
		t.Errorf("got %q, want %q", got, want)
	}
	got := note.MissingSpan{Matched: "the wrath", Instead: "the calm"}.Error()
	want := `this span is not in the note; it holds "the calm" where the span has "the wrath…"`
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestAnAmbiguousSpanSaysHowManyPlaces(t *testing.T) {
	t.Parallel()

	got := note.AmbiguousSpan{Places: 3}.Error()
	want := "this span stands in 3 places; take in enough of what is around one of them to tell it from the others"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestAReplacementMustNameTheTextItReplaces(t *testing.T) {
	t.Parallel()
	was := "one two three\n"
	c := openChanging(t, map[string]string{"Aggressor.md": was})

	if _, err := c.replace().Execute(t.Context(), c.vault, "Aggressor.md", "", "four", domain.Fingerprint{}); err == nil {
		t.Fatal("a replacement naming nothing was accepted")
	}
	if body := c.read(t, "Aggressor.md"); body != was {
		t.Errorf("the note changed:\n%s", body)
	}
}

// Emptying a span is a replacement; a span that is not there is not made
// present by it.
func TestAMissingSpanReplacedByNothingIsMissing(t *testing.T) {
	t.Parallel()
	c := openChanging(t, map[string]string{"Aggressor.md": "one two three\n"})

	_, err := c.replace().Execute(t.Context(), c.vault, "Aggressor.md", "four", "", domain.Fingerprint{})
	var nowhere note.MissingSpan
	if !errors.As(err, &nowhere) {
		t.Fatalf("want MissingSpan, got %v", err)
	}
}

func TestAReplacementPresentingAStaleFingerprintIsRefused(t *testing.T) {
	t.Parallel()
	c := openChanging(t, map[string]string{"Aggressor.md": "one two three\n"})
	replacing := c.replace()

	first, err := replacing.Execute(t.Context(), c.vault, "Aggressor.md", "one", "ONES", domain.Fingerprint{})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := replacing.Execute(
		t.Context(), c.vault, "Aggressor.md", "two", "TWOS", first.Fingerprint); err != nil {
		t.Fatal(err)
	}
	before := c.read(t, "Aggressor.md")

	_, err = replacing.Execute(t.Context(), c.vault, "Aggressor.md", "three", "THREES", first.Fingerprint)
	if !errors.Is(err, port.ErrStale) {
		t.Fatalf("want ErrStale, got %v", err)
	}
	if body := c.read(t, "Aggressor.md"); body != before {
		t.Errorf("the rejected replacement landed anyway:\n%s", body)
	}
}

func TestAReplacementAnswersWithTheFileItWrote(t *testing.T) {
	t.Parallel()
	c := openChanging(t, map[string]string{"Aggressor.md": "one two three\n"})

	done, err := c.replace().Execute(t.Context(), c.vault, "Aggressor.md", "one", "ONE", domain.Fingerprint{})
	if err != nil {
		t.Fatal(err)
	}
	written := c.read(t, "Aggressor.md")
	if done.Fingerprint.Path != "Aggressor.md" || done.Fingerprint.Size != int64(len(written)) {
		t.Errorf("the answer describes %+v, and the file is %d bytes", done.Fingerprint, len(written))
	}
}

func TestAReplacementThatMakesTheNoteTooLargeIsRefused(t *testing.T) {
	t.Parallel()
	was := "# Big\n\n" + strings.Repeat("a", note.MaxBytes-100) + "\nneedle\n"
	c := openChanging(t, map[string]string{"Big.md": was})

	_, err := c.replace().Execute(t.Context(), c.vault, "Big.md", "needle", strings.Repeat("n", 200), domain.Fingerprint{})
	if !errors.Is(err, note.ErrTooLarge) {
		t.Fatalf("want ErrTooLarge, got %v", err)
	}
	if body := c.read(t, "Big.md"); body != was {
		t.Error("the rejected replacement landed anyway")
	}
}

// Frontmatter that is never closed leaves no prose to replace.
func TestAReplacementInANoteWhoseFrontmatterIsNeverClosedIsRefused(t *testing.T) {
	t.Parallel()
	was := "---\ntitle: Aggressor\nA hedgehog.\n"
	c := openChanging(t, map[string]string{"Aggressor.md": was})

	_, err := c.replace().Execute(t.Context(), c.vault, "Aggressor.md", "A hedgehog", "An axe", domain.Fingerprint{})
	if !errors.Is(err, markdown.ErrUnterminated) {
		t.Fatalf("want ErrUnterminated, got %v", err)
	}
	if body := c.read(t, "Aggressor.md"); body != was {
		t.Errorf("the rejected replacement landed anyway:\n%s", body)
	}
}

// Whoever is drawing the note is told the stretch in the units a client counts,
// and told when the change is over.
func TestWhoeverIsDrawingTheNoteIsToldTheStretchInUTF16(t *testing.T) {
	t.Parallel()
	c := openChanging(t, map[string]string{"Aggressor.md": "ééé 😀 target tail\n"})
	var told []domain.Edit
	replacing := c.replace()
	replacing.Drawing = func(_ context.Context, said domain.Edit) { told = append(told, said) }

	if _, err := replacing.Execute(t.Context(), c.vault, "Aggressor.md", "target", "goal", domain.Fingerprint{}); err != nil {
		t.Fatal(err)
	}
	if len(told) != 2 {
		t.Fatalf("told %d times: %+v", len(told), told)
	}
	begun, ended := told[0], told[1]
	if begun.Path != "Aggressor.md" || begun.From != 7 || begun.To != 13 || begun.Text != "goal" || begun.IsDone {
		t.Errorf("the change began as %+v", begun)
	}
	if begun.Change == "" {
		t.Error("the change has no name")
	}
	if ended.Change != begun.Change || ended.Path != "Aggressor.md" || !ended.IsDone {
		t.Errorf("the change ended as %+v", ended)
	}
}

// What a missing span says about itself: how much of its opening stands in the
// note, and what stands there from that point.
func TestAMissingSpanNamesWhereACopyStoppedAgreeing(t *testing.T) {
	t.Parallel()
	for name, tc := range map[string]struct {
		note, stood      string
		matched, instead string
	}{
		"a single byte that is nowhere": {
			note: "abc\n", stood: "z",
		},
		"an opening of one byte": {
			note: "ab\n", stood: "ax",
			matched: "a", instead: "ab",
		},
		"an opening that is most of the span": {
			note: "abcdZ\n", stood: "abcdxy",
			matched: "abcd", instead: "abcdZ\n",
		},
		"a letter that differs inside a rune": {
			note: "xéz\n", stood: "xèy",
			matched: "x", instead: "xéz",
		},
		"a rune the note cuts at the end of what stands there": {
			note: "abcéfgh\n", stood: "abXY",
			matched: "ab", instead: "abcé",
		},
		"a first rune that is nowhere": {
			note: "é\n", stood: "èy",
		},
		"bytes that are not text": {
			note: "abc\n", stood: "\x80ab",
		},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			c := openChanging(t, map[string]string{"Aggressor.md": tc.note})

			_, err := c.replace().Execute(t.Context(), c.vault, "Aggressor.md", tc.stood, "q", domain.Fingerprint{})
			var nowhere note.MissingSpan
			if !errors.As(err, &nowhere) {
				t.Fatalf("want MissingSpan, got %v", err)
			}
			if nowhere.Matched != tc.matched || nowhere.Instead != tc.instead {
				t.Errorf("matched %q and instead %q, want %q and %q",
					nowhere.Matched, nowhere.Instead, tc.matched, tc.instead)
			}
		})
	}
}

func TestAReplacementNamingNothingIsRefusedAsSuch(t *testing.T) {
	t.Parallel()
	c := openChanging(t, map[string]string{"Aggressor.md": "one two three\n"})

	_, err := c.replace().Execute(t.Context(), c.vault, "Aggressor.md", "", "four", domain.Fingerprint{})
	var nowhere note.MissingSpan
	if err == nil || errors.As(err, &nowhere) || !strings.Contains(err.Error(), "name the text to replace") {
		t.Errorf("want the rejection to name the text, got %v", err)
	}
}

// A rejected replacement draws nothing and names the note it failed in.
func TestAReplacementRefusedForItsFrontmatterDrawsNothing(t *testing.T) {
	t.Parallel()
	c := openChanging(t, map[string]string{"Aggressor.md": "---\ntitle: Aggressor\nA hedgehog.\n"})
	var told []domain.Edit
	replacing := c.replace()
	replacing.Drawing = func(_ context.Context, said domain.Edit) { told = append(told, said) }

	_, err := replacing.Execute(t.Context(), c.vault, "Aggressor.md", "A hedgehog", "An axe", domain.Fingerprint{})
	if err == nil || !strings.HasPrefix(err.Error(), "Aggressor.md: ") {
		t.Errorf("want the error to name the note, got %v", err)
	}
	if len(told) != 0 {
		t.Errorf("a rejected change was drawn: %+v", told)
	}
}

// A rune of three bytes puts the middle of the search inside it.
func TestAMissingSpanAfterAThreeByteRuneNamesWhereItStopped(t *testing.T) {
	t.Parallel()
	c := openChanging(t, map[string]string{"Aggressor.md": "x\n"})

	_, err := c.replace().Execute(t.Context(), c.vault, "Aggressor.md", "x€y", "q", domain.Fingerprint{})
	var nowhere note.MissingSpan
	if !errors.As(err, &nowhere) {
		t.Fatalf("want MissingSpan, got %v", err)
	}
	if nowhere.Matched != "x" || nowhere.Instead != "x\n" {
		t.Errorf("matched %q and instead %q", nowhere.Matched, nowhere.Instead)
	}
}
