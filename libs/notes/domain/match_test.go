package domain_test

import (
	"testing"

	"github.com/jiva-studio/numen/libs/notes/domain"
)

func TestANameIsMatchedByTheOrderOfPriority(t *testing.T) {
	cases := []struct {
		name       string
		from       string
		input      string
		candidates []string
		want       string
		ambiguous  bool
	}{
		{"nothing answers", "a/Note.md", "Entropy", nil, "", false},
		{"an exact path from the root", "a/Note.md", "b/Entropy", []string{"a/b/Entropy.md", "b/Entropy.md"}, "b/Entropy.md", false},
		{"a path relative to the note", "a/Note.md", "b/Entropy", []string{"a/b/Entropy.md", "c/b/Entropy.md"}, "a/b/Entropy.md", false},
		{"a single match by name", "a/Note.md", "Entropy", []string{"z/Entropy.md"}, "z/Entropy.md", false},
		{"the nearest of several", "a/b/Note.md", "Entropy", []string{"c/Entropy.md", "a/Entropy.md"}, "a/Entropy.md", true},
		{"a name in another case", "Note.md", "entropy", []string{"Entropy.md"}, "Entropy.md", false},
	}
	for _, one := range cases {
		t.Run(one.name, func(t *testing.T) {
			got, ambiguous := domain.ChooseMatch(one.from, one.input, one.candidates)

			if got != one.want || ambiguous != one.ambiguous {
				t.Errorf("got %q, %v; want %q, %v", got, ambiguous, one.want, one.ambiguous)
			}
		})
	}
}
