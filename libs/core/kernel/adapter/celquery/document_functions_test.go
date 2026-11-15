package celquery_test

import (
	"testing"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/searchquery"
)

func TestANoteTypeAndAModifiedRangeParseToTwoComparisons(t *testing.T) {
	q, err := parse(`noteType == "book" && modified >= date("2026-01-01")`)
	if err != nil {
		t.Fatal(err)
	}
	if q.IsSearchResults || countLeaves(q.Root, searchquery.LeafCompare) != 2 {
		t.Fatalf("the query is %+v, want two comparisons of type bool", q)
	}
	modified := q.Root.Children[1].Leaf
	if modified.Attribute != searchquery.AttributeModified || modified.Comparison != searchquery.ComparisonGreaterOrEqual || modified.Operands[0].Kind != searchquery.ValueTime {
		t.Errorf("the second leaf is %+v", modified)
	}
}

func TestALiteralOnTheLeftIsTurnedAround(t *testing.T) {
	q, err := parse(`4 <= property("rating")`)
	if err != nil {
		t.Fatal(err)
	}
	if len(q.Sets) != 1 {
		t.Fatalf("sets %+v, want one", q.Sets)
	}
	set := q.Sets[0]
	if set.Function != searchquery.FunctionProperty || set.Key != "rating" || set.Comparison != searchquery.ComparisonGreaterOrEqual || set.Operands[0].Int != 4 {
		t.Errorf("the set is %+v, want property(rating) >= 4", set)
	}
}

func TestEveryShapeOfTheDocumentFunctionsParses(t *testing.T) {
	for _, text := range []string{
		`title == "Lecture 1"`, `meaning("a") && title != "x"`, `title in ["a", "b"]`, `title.startsWith("Lec")`, `title.endsWith("1")`,
		`title.contains("ctu")`, `noteType in ["book", "article"]`, `modified < date("2026-06-01T12:00")`, `size > 1000`,
		`property("status") == "done"`, `property("draft") == true`, `property("source") == null`, `property("r") > 1.5`,
		`property("due") < date("2026-12-01")`, `property("status") in ["done", 1]`, `"physics" in property("topics")`,
		`hasProperty("source")`, `hasHeading("Summary")`, `meaning("a") && !hasHeading("Summary")`,
	} {
		if _, err := parse(text); err != nil {
			t.Errorf("%s: %v", text, err)
		}
	}
}

func TestABadArgumentIsAProblemAtTheArgument(t *testing.T) {
	cases := map[string]int{
		`modified < date("2026-13-01")`: 17,
		`modified < date(path)`:         17,
		`property(path) == 1`:           10,
		`property("") == 1`:             10,
		`hasHeading("  ")`:              12,
	}
	for text, column := range cases {
		problems := getProblems(t, text)
		if len(problems) == 0 || problems[0].Position != (searchquery.Position{Line: 1, Column: column}) {
			t.Errorf("%s: problems %+v, want one at 1:%d", text, problems, column)
		}
	}
}

func TestShapesOutsideTheDocumentFunctionsAreRejected(t *testing.T) {
	cases := map[string]string{
		`title < "b"`:                              "Compare title",
		`noteType.startsWith("b")`:                 "Compare noteType",
		`property("a") == property("b")`:           "property takes",
		`property("k").startsWith("x")`:            "property takes",
		`title in property("k")`:                   "property takes",
		`property("k") in [["a"]]`:                 "property takes",
		`size == 1.5`:                              "",
		`modified > "2024-01-01"`:                  "",
		`date("2026-01-01") == date("2026-01-02")`: "",
	}
	for text, message := range cases {
		problems := getProblems(t, text)
		if len(problems) == 0 || !hasProblem(problems, message) {
			t.Errorf("%s: problems %+v, want one naming %q", text, problems, message)
		}
	}
}

func TestAMisspelledHasPropertyIsSuggested(t *testing.T) {
	problems := getProblems(t, `hasPropety("x")`)
	if len(problems) != 1 || problems[0].Suggestion != searchquery.FunctionHasProperty {
		t.Errorf("problems %+v, want a suggestion of hasProperty", problems)
	}
}

func TestEveryExampleOfTheCatalogParses(t *testing.T) {
	for _, f := range searchquery.GetBuiltinCatalog().Functions {
		if _, err := parse(f.Example); err != nil {
			t.Errorf("the example of %s, %s: %v", f.Name, f.Example, err)
		}
	}
}
