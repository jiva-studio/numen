package tags_test

import (
	"reflect"
	"slices"
	"testing"

	"github.com/jiva-studio/numen/plugins/tools-taxonomy/backend/domain/tags"
)

func TestTheTagsOfAValueOfTheKeyTags(t *testing.T) {
	for name, c := range map[string]struct {
		value any
		want  []string
	}{
		"a string is one tag":                  {value: "a/b", want: []string{"a/b"}},
		"a list, cleaned, with a repeat once":  {value: []any{"#A/ b ", "a/b"}, want: []string{"A/b"}},
		"a number is its text":                 {value: float64(5), want: []string{"5"}},
		"a map is no tag":                      {value: map[string]any{"x": float64(1)}, want: nil},
		"an empty list is no tag":              {value: []any{}, want: nil},
		"a list keeps its strings and numbers": {value: []any{"river", nil, []any{"x"}, float64(2.5)}, want: []string{"river", "2.5"}},
		"empty segments are dropped":           {value: "//delta//mouth/", want: []string{"delta/mouth"}},
		"a hash and spaces alone are no tag":   {value: []any{"#", "  ", "/"}, want: nil},
		"only one leading hash is dropped":     {value: "##river", want: []string{"#river"}},
		"a doubled hash is another tag":        {value: []any{"##river", "river"}, want: []string{"#river", "river"}},
	} {
		if got := tags.Parse(c.value); !slices.Equal(got, c.want) {
			t.Errorf("%s: Parse(%#v) = %q, want %q", name, c.value, got, c.want)
		}
	}
}

func TestATagIsTheSameInAnyCaseAndInAnyComposition(t *testing.T) {
	for _, one := range []string{"враджа", "ВРАДЖА", "#Враджа"} {
		if tags.GetKey(one) != tags.GetKey("Враджа") {
			t.Errorf("%q and Враджа are two tags", one)
		}
	}
	composed, decomposed := "\u0401лка", "\u0415\u0308лка"
	if tags.GetKey(decomposed) != tags.GetKey(composed) {
		t.Error("a decomposed Ё makes another tag than the composed one")
	}
	if got := tags.Parse([]any{decomposed, "ёлка"}); !slices.Equal(got, []string{composed}) {
		t.Errorf("Parse = %q, want one tag in the composed spelling", got)
	}
}

func TestTheTreeCountsTheNotesOfATagAndOfItsNestedTags(t *testing.T) {
	notes := map[string][]string{
		"a.md": {"Projects/Web", "Projects"},
		"b.md": {"projects/web/api"},
		"c.md": {"Projects/Print"},
		"d.md": {"River"},
		"e.md": {"river", "Delta"},
	}

	got := tags.GetTree(notes, "")

	want := []tags.Node{
		{Name: "Projects", Tag: "Projects", Count: 3, Children: []tags.Node{
			{Name: "Web", Tag: "Projects/Web", Count: 2, Children: []tags.Node{
				{Name: "api", Tag: "Projects/Web/api", Count: 1, Children: []tags.Node{}},
			}},
			{Name: "Print", Tag: "Projects/Print", Count: 1, Children: []tags.Node{}},
		}},
		{Name: "River", Tag: "River", Count: 2, Children: []tags.Node{}},
		{Name: "Delta", Tag: "Delta", Count: 1, Children: []tags.Node{}},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("tree = %+v\nwant %+v", got, want)
	}
}

func TestTheTreeOfOneTagIsThatTagAndItsNestedTags(t *testing.T) {
	notes := map[string][]string{"a.md": {"Projects/Web"}, "b.md": {"River"}}

	got := tags.GetTree(notes, "#projects/ ")

	want := []tags.Node{{Name: "Projects", Tag: "Projects", Count: 1, Children: []tags.Node{
		{Name: "Web", Tag: "Projects/Web", Count: 1, Children: []tags.Node{}},
	}}}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("tree = %+v\nwant %+v", got, want)
	}
	if got := tags.GetTree(notes, "Lake"); len(got) != 0 {
		t.Errorf("a tag no note has gave %+v", got)
	}
	if got := tags.GetTree(notes, "Projects/Web/Api"); len(got) != 0 {
		t.Errorf("a tag below every tag gave %+v", got)
	}
}
