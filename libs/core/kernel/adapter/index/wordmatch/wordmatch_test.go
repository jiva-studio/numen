package wordmatch_test

import (
	"context"
	"errors"
	"slices"
	"strings"
	"testing"

	"github.com/jiva-studio/numen/libs/core/kernel/adapter/index/wordmatch"
	"github.com/jiva-studio/numen/libs/core/kernel/domain/source"
)

func matchWords(t *testing.T, expression string, texts ...string) []bool {
	t.Helper()
	got, err := wordmatch.NewMatcher().MatchWords(t.Context(), expression, texts)
	if err != nil {
		t.Fatalf("%q: %v", expression, err)
	}
	if len(got) != len(texts) {
		t.Fatalf("%q answered %d texts of %d", expression, len(got), len(texts))
	}
	return got
}

func TestMatchWordsMatchesAPhrase(t *testing.T) {
	got := matchWords(t, `"gray heron"`, "the gray heron waits", "a heron, gray and old")

	if !slices.Equal(got, []bool{true, false}) {
		t.Errorf("a phrase matched %v, want only the text with the words in order", got)
	}
}

func TestMatchWordsMatchesEveryWordAnywhereInTheText(t *testing.T) {
	long := "heron " + strings.Repeat("filler ", 2000) + "reed"

	got := matchWords(t, "heron reed", long, "heron alone")

	if !slices.Equal(got, []bool{true, false}) {
		t.Errorf("two words matched %v, want the long text with both and not the one with one", got)
	}
}

func TestMatchWordsMatchesNEARWithinTheDistanceAndNotBeyond(t *testing.T) {
	got := matchWords(t, "NEAR(heron reed, 3)",
		"the heron waits by the reed",
		"the heron walks along the bank and later finds a reed")

	if !slices.Equal(got, []bool{true, false}) {
		t.Errorf("NEAR matched %v, want the words three apart and not the ones further apart", got)
	}
}

func TestMatchWordsMatchesAPrefix(t *testing.T) {
	got := matchWords(t, "strip*", "zebra stripes", "a strip of land", "zebra")

	if !slices.Equal(got, []bool{true, true, false}) {
		t.Errorf("a prefix matched %v, want the two texts with a word that starts with it", got)
	}
}

func TestMatchWordsExcludesATextWithTheWordAfterNOT(t *testing.T) {
	got := matchWords(t, "heron NOT reed", "the heron waits by the reed", "the heron fishes")

	if !slices.Equal(got, []bool{false, true}) {
		t.Errorf("NOT matched %v, want only the text without the excluded word", got)
	}
}

func TestMatchWordsMatchesWithAndOrAndParentheses(t *testing.T) {
	got := matchWords(t, "(heron OR crane) AND reed", "a crane in the reed", "a heron at dawn", "reed alone")

	if !slices.Equal(got, []bool{true, false, false}) {
		t.Errorf("OR and AND matched %v, want only the text with a bird and the reed", got)
	}
}

func TestMatchWordsAnswersEveryTextOfAnEmptyList(t *testing.T) {
	got, err := wordmatch.NewMatcher().MatchWords(t.Context(), "heron", nil)
	if err != nil || len(got) != 0 {
		t.Errorf("no texts answered %v, %v, want nothing", got, err)
	}
}

func TestMatchWordsRefusesAQueryFTS5CannotParse(t *testing.T) {
	for _, expression := range []string{`"heron`, "heron AND", "NEAR(heron", "AND"} {
		_, err := wordmatch.NewMatcher().MatchWords(t.Context(), expression, []string{"the heron"})
		if !errors.Is(err, source.ErrInvalidWords) {
			t.Errorf("%q returned %v, want ErrInvalidWords", expression, err)
		}
	}
}

func TestMatchWordsTreatsTheExpressionAsAQueryAndNotAsSQL(t *testing.T) {
	got := matchWords(t, `"heron'); DROP TABLE t; --"`, "heron drop table t", "a crane")

	if !slices.Equal(got, []bool{true, false}) {
		t.Errorf("a phrase with SQL in it matched %v, want only the text with its words", got)
	}
}

func TestMatchWordsStopsOnACanceledContext(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	if _, err := wordmatch.NewMatcher().MatchWords(ctx, "heron", []string{"the heron"}); !errors.Is(err, context.Canceled) {
		t.Errorf("a canceled call returned %v, want context.Canceled", err)
	}
}
