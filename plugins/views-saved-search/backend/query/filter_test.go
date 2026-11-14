package query_test

import (
	"slices"
	"testing"
)

func TestFilterTakesKindsFromTopLevelConjuncts(t *testing.T) {
	for text, want := range map[string][]string{
		`kind == "note" && meaning("tides")`:                             {"note"},
		`meaning("tides") && kind in ["pdf", "audio"]`:                   {"file"},
		`meaning("tides") && kind != "note"`:                             {"file"},
		`meaning("tides") && kind in ["note", "epub"] && kind != "epub"`: {"note"},
		`meaning("tides") && kind != "pdf"`:                              {"note", "file"},
	} {
		got := compile(t, text).Filter.Kinds
		slices.Sort(got)
		slices.Sort(want)
		if !slices.Equal(got, want) && (len(want) != 2 || len(got) != 0) {
			t.Errorf("kinds of %q = %v, want %v", text, got, want)
		}
	}
}

func TestFilterIgnoresKindTermsInsideAnOr(t *testing.T) {
	filter := compile(t, `meaning("tides") && (kind == "note" || words("harbor"))`).Filter

	if len(filter.Kinds) != 0 || len(filter.Prefixes) != 0 {
		t.Errorf("filter = %+v, want every kind and no prefix", filter)
	}
}

func TestFilterIgnoresATermThatIsTheWholeOr(t *testing.T) {
	filter := compile(t, `kind == "note" || meaning("tides")`).Filter

	if len(filter.Kinds) != 0 {
		t.Errorf("filter = %+v, want every kind", filter)
	}
}

func TestFilterTakesPrefixesFromATopLevelOrOfPaths(t *testing.T) {
	filter := compile(t, `(path.startsWith("Atlases/") || path.startsWith("Logs/")) && meaning("tides")`).Filter

	if !slices.Equal(filter.Prefixes, []string{"Atlases/", "Logs/"}) {
		t.Errorf("prefixes = %v, want Atlases/ and Logs/", filter.Prefixes)
	}
}

func TestFilterTakesThePrefixOfATopLevelPathTerm(t *testing.T) {
	filter := compile(t, `meaning("tides") && path.startsWith("Logs/")`).Filter

	if !slices.Equal(filter.Prefixes, []string{"Logs/"}) {
		t.Errorf("prefixes = %v, want Logs/", filter.Prefixes)
	}
}

func TestFilterTakesNoPrefixFromANegatedPathOrAMixedOr(t *testing.T) {
	for _, text := range []string{
		`meaning("tides") && !path.startsWith("Archive/")`,
		`meaning("tides") && (path.startsWith("Logs/") || words("harbor"))`,
		`meaning("tides") || path.startsWith("Logs/")`,
	} {
		if filter := compile(t, text).Filter; len(filter.Prefixes) != 0 {
			t.Errorf("prefixes of %q = %v, want none", text, filter.Prefixes)
		}
	}
}
