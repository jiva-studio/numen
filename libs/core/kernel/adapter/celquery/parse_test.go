package celquery_test

import (
	"errors"
	"strconv"
	"strings"
	"testing"

	"github.com/jiva-studio/numen/libs/core/kernel/adapter/celquery"
	"github.com/jiva-studio/numen/libs/core/kernel/domain/searchquery"
)

func parse(text string) (searchquery.Query, error) {
	limits := searchquery.DefaultLimits()
	if err := searchquery.CheckQueryText(text, limits); err != nil {
		return searchquery.Query{}, err
	}
	raw, err := celquery.NewParser(limits).ParseQuery(text, searchquery.GetBuiltinCatalog())
	if err != nil {
		return searchquery.Query{}, err
	}
	return searchquery.Normalize(raw, limits)
}

func getProblems(t *testing.T, text string) []searchquery.Problem {
	t.Helper()
	_, err := parse(text)
	parseError, ok := errors.AsType[*searchquery.ParseError](err)
	if !ok {
		t.Fatalf("%s returned %v, want a ParseError", text, err)
	}
	return parseError.Problems
}

func hasProblem(problems []searchquery.Problem, message string) bool {
	for _, p := range problems {
		if strings.Contains(p.Message, message) {
			return true
		}
	}
	return false
}

func countLeaves(node searchquery.Node, kind searchquery.LeafKind) int {
	if node.Operator == searchquery.OperatorLeaf {
		if node.Leaf.Kind == kind {
			return 1
		}
		return 0
	}
	count := 0
	for _, child := range node.Children {
		count += countLeaves(child, kind)
	}
	return count
}

func TestAMeaningWithDocumentConditionsParsesToSearchResults(t *testing.T) {
	q, err := parse(`meaning("a") && kind == "pdf" && path.startsWith("x/")`)
	if err != nil {
		t.Fatal(err)
	}
	if !q.IsSearchResults || len(q.Conditions) != 1 {
		t.Errorf("the query is %+v, want SearchResults with one condition", q)
	}
	if countLeaves(q.Root, searchquery.LeafMeaning) != 1 || countLeaves(q.Root, searchquery.LeafKindIn) != 1 || countLeaves(q.Root, searchquery.LeafPathPrefix) != 1 {
		t.Errorf("the tree is %+v", q.Root)
	}
}

func TestAKindListAloneParsesToADocumentPredicate(t *testing.T) {
	q, err := parse(`kind in ["note", "pdf"]`)
	if err != nil {
		t.Fatal(err)
	}
	if q.IsSearchResults || q.Root.Leaf.Kind != searchquery.LeafKindIn || len(q.Root.Leaf.Kinds) != 2 {
		t.Errorf("the query is %+v, want a kind leaf of type bool", q)
	}
}

func TestAMisspelledFunctionIsReportedWithASuggestion(t *testing.T) {
	problems := getProblems(t, `meanin("a")`)
	if len(problems) != 1 {
		t.Fatalf("problems %+v", problems)
	}
	p := problems[0]
	if p.Position != (searchquery.Position{Line: 1, Column: 1}) || !strings.Contains(p.Message, "meanin") || p.Suggestion != searchquery.FunctionMeaning {
		t.Errorf("the problem is %+v", p)
	}
}

func TestAProblemOnTheSecondLineIsReportedInCodePoints(t *testing.T) {
	problems := getProblems(t, "meaning(\"река\") &&\n  (meanin(\"b\"))")
	if len(problems) != 1 || problems[0].Position != (searchquery.Position{Line: 2, Column: 4}) {
		t.Errorf("problems %+v, want one at 2:4", problems)
	}
	problems = getProblems(t, "words(\"a\") ||\n\"ё\" == meanin(\"b\")")
	if len(problems) != 1 || problems[0].Position != (searchquery.Position{Line: 2, Column: 8}) {
		t.Errorf("problems %+v, want one at 2:8", problems)
	}
}

func TestAMacroIsAnUnknownFunction(t *testing.T) {
	problems := getProblems(t, `has(x.y) && meaning("a")`)
	if !hasProblem(problems, "unknown function has") {
		t.Errorf("problems %+v, want unknown function has", problems)
	}
}

func TestShapesOutsideTheLanguageAreRejectedAtTheirPosition(t *testing.T) {
	for _, text := range []string{`meaning(a)`, `path == "x"`, `kind.startsWith("p")`, `meaning("a") + 1`} {
		problems := getProblems(t, text)
		if len(problems) == 0 || problems[0].Message == "" || problems[0].Position.Line != 1 {
			t.Errorf("%s: problems %+v", text, problems)
		}
	}
	problems := getProblems(t, `words("b") || path == "x"`)
	if len(problems) != 1 || problems[0].Position.Column < 15 {
		t.Errorf("path == x: problems %+v, want one at its place", problems)
	}
}

func TestAStageIsUnknown(t *testing.T) {
	problems := getProblems(t, `meaning("a").top(5)`)
	if !hasProblem(problems, "unknown stage top") {
		t.Errorf("problems %+v, want unknown stage top", problems)
	}
}

func TestNormalizeOfANegatedGroupNegatesEachLeaf(t *testing.T) {
	q, err := parse(`meaning("c") && !(meaning("a") || words("b"))`)
	if err != nil {
		t.Fatal(err)
	}
	if q.Root.Operator != searchquery.OperatorAnd || len(q.Root.Children) != 3 {
		t.Fatalf("the root is %+v", q.Root)
	}
	for i, child := range q.Root.Children {
		if child.Operator != searchquery.OperatorLeaf || child.IsNegated != (i > 0) {
			t.Errorf("child %d is %+v", i, child)
		}
	}
}

func TestSeventeenDistinctConditionsAreRejectedAndSixteenPass(t *testing.T) {
	var parts []string
	for i := range 16 {
		parts = append(parts, `words("w`+strconv.Itoa(i)+`")`)
	}
	if _, err := parse(strings.Join(parts, " || ")); err != nil {
		t.Errorf("16 conditions: %v", err)
	}
	problems := getProblems(t, strings.Join(append(parts, `words("w16")`), " || "))
	if len(problems) != 1 || problems[0].Position.Column < 16*15 {
		t.Errorf("17 conditions: %+v, want one problem at the last", problems)
	}
}

func TestAQueryWithOnlyNegatedConditionsIsRejected(t *testing.T) {
	for _, text := range []string{`!words("x")`, `kind != "pdf"`} {
		if !hasProblem(getProblems(t, text), "a query needs a condition that is not negated") {
			t.Errorf("%s was accepted", text)
		}
	}
}

func TestAQueryOverTheByteLimitIsRejectedAtTheStart(t *testing.T) {
	text := `words("` + strings.Repeat("a", searchquery.DefaultLimits().MaxQueryBytes) + `")`
	problems := getProblems(t, text[:searchquery.DefaultLimits().MaxQueryBytes+1])
	if len(problems) != 1 || problems[0].Position != searchquery.StartPosition() {
		t.Errorf("problems %+v", problems)
	}
}

func TestAnUnknownKindListsTheKinds(t *testing.T) {
	problems := getProblems(t, `kind == "book"`)
	if len(problems) != 1 {
		t.Fatalf("problems %+v", problems)
	}
	for _, kind := range searchquery.ListKinds() {
		if !strings.Contains(problems[0].Message, `"`+kind+`"`) {
			t.Errorf("the message %q does not name %s", problems[0].Message, kind)
		}
	}
}

func TestDeepNestingIsRejected(t *testing.T) {
	text := strings.Repeat("(", 40) + `words("a")` + strings.Repeat(")", 40)
	if len(getProblems(t, text)) == 0 {
		t.Error("40 levels of parentheses were accepted")
	}
}

func TestLinkFunctionsParseToSetConditionsWithTargetAndType(t *testing.T) {
	q, err := parse(`hasLinkTo("Entropy", "requires") && !isChildOf("Physics")`)
	if err != nil {
		t.Fatal(err)
	}
	if countLeaves(q.Root, searchquery.LeafSet) != 2 || len(q.Sets) != 2 || q.IsSearchResults {
		t.Fatalf("the query is %+v, want two set leaves", q)
	}
	if q.Sets[0] != (searchquery.SetCondition{Function: "hasLinkTo", Target: "Entropy", LinkType: "requires", IsPositive: true, Position: searchquery.StartPosition()}) {
		t.Errorf("the first set is %+v", q.Sets[0])
	}
	if q.Sets[1].Function != "isChildOf" || q.Sets[1].IsPositive || !q.Root.Children[1].IsNegated {
		t.Errorf("the second set is %+v, want a negated isChildOf", q.Sets[1])
	}
}

func TestALinkFunctionReportsABadArgumentOrUse(t *testing.T) {
	for text, message := range map[string]string{
		`hasLinkTo(path)`:             "takes a string literal",
		`hasLinkTo("  ")`:             "the target of hasLinkTo is empty",
		`isChildOf("a", "b", "c")`:    "no matching overload",
		`hasLinkTo("a", "b") == true`: "condition over",
	} {
		if problems := getProblems(t, text); !hasProblem(problems, message) {
			t.Errorf("%s gave %+v, want %q", text, problems, message)
		}
	}
}

func TestAMisspelledLinkFunctionSuggestsTheName(t *testing.T) {
	problems := getProblems(t, `isDescendentOf("x")`)
	if len(problems) != 1 || problems[0].Suggestion != "isDescendantOf" {
		t.Errorf("got %+v, want a suggestion of isDescendantOf", problems)
	}
}

func TestEveryCatalogExampleParses(t *testing.T) {
	for _, f := range searchquery.GetBuiltinCatalog().Functions {
		if _, err := parse(f.Example); err != nil {
			t.Errorf("the example of %s, %s, does not parse: %v", f.Name, f.Example, err)
		}
	}
}
