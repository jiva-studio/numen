package chunk

import (
	"slices"
	"testing"
)

func TestExpressions(t *testing.T) {
	for _, c := range []struct {
		name    string
		typed   string
		growing bool
		want    []string
	}{
		{"blank", "  ", true, nil},
		{"a finished query is exact", "knigi", false, []string{`"knigi"`}},
		{"a long word: exact, prefix, stem", "knigi", true, []string{`"knigi"`, `"knigi"*`, `"knig"*`}},
		{"two letters off from seven letters", "knigami", true, []string{`"knigami"`, `"knigami"*`, `"kniga"*`}},
		{"a three-letter word stays exact and prefix", "dom", true, []string{`"dom"`, `"dom"*`}},
		{"only the last word is a prefix, every long word a stem", "voda knigi", true,
			[]string{`"voda" "knigi"`, `"voda" "knigi"*`, `"vod"* "knig"*`}},
		{"a short word beside a long one is not cut", "on knigi", true,
			[]string{`"on" "knigi"`, `"on" "knigi"*`, `"on" "knig"*`}},
		{"Cyrillic", "Книга", true, []string{`"Книга"`, `"Книга"*`, `"Книг"*`}},
		{"a quote is doubled", `a"bcd`, true, []string{`"a""bcd"`, `"a""bcd"*`, `"a""bc"*`}},
	} {
		t.Run(c.name, func(t *testing.T) {
			if got := Expressions(c.typed, c.growing); !slices.Equal(got, c.want) {
				t.Errorf("Expressions(%q, %v) = %q, want %q", c.typed, c.growing, got, c.want)
			}
		})
	}
}
