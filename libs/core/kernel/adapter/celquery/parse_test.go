package celquery_test

import (
	"errors"
	"slices"
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
	if !hasProblem(problems, "Unknown function has") {
		t.Errorf("problems %+v, want Unknown function has", problems)
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
	if !hasProblem(problems, "Unknown stage top") {
		t.Errorf("problems %+v, want Unknown stage top", problems)
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
		if !hasProblem(getProblems(t, text), "A query needs a condition that is not negated") {
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

func TestASyntaxProblemReadsInPlainWordsAtItsPlace(t *testing.T) {
	owner := "(meanling\n(\"Мадхавендра Пури) && meaning(\"Кришна\"\n)).rerank(\"как Мадхавендрапури и Кришна связаны?\")."
	tests := []struct {
		name string
		text string
		want []string
	}{
		{"the query of the owner", owner, []string{
			"1:2: Unknown function meanling: did you mean meaning?",
			"2:33: A quote is missing before Кришна",
			"3:4: Unknown function rerank",
			"3:51: The query ends with a dot: remove it or write a stage after it",
		}},
		{"an unclosed string", `meaning("river`, []string{"1:9: This quote is not closed"}},
		{"an unclosed string before another line", "words(\"a\" && meaning(\"b)\n|| words(\"c\")", []string{"1:22: This quote is not closed"}},
		{"a word with no opening quote", `meaning(Кришна")`, []string{"1:9: A quote is missing before Кришна"}},
		{"an unclosed parenthesis", `(meaning("a") && words("b")`, []string{"1:1: This parenthesis is not closed"}},
		{"a parenthesis with no opening one", `meaning("a"))`, []string{"1:13: This parenthesis has no opening one"}},
		{"an unknown function", `meanling("a")`, []string{"1:1: Unknown function meanling: did you mean meaning?"}},
		{"an unknown function next to a syntax error", "meanling(\"a\") &&", []string{
			"1:1: Unknown function meanling: did you mean meaning?", "1:15: A condition must follow &&",
		}},
		{"a trailing dot", `meaning("a").`, []string{"1:13: The query ends with a dot: remove it or write a stage after it"}},
		{"a trailing operator", `meaning("a") || !`, []string{"1:17: A condition must follow !"}},
		{"letters outside a string", `meaning("a") && Мадхавендра Пури`, []string{"1:17: Put this text in quotes: Мадхавендра Пури"}},
		{"a single ampersand", `meaning("a") & words("b")`, []string{`1:14: Write && for "and"`}},
		{"a character outside the language", `meaning("a") @ words("b")`, []string{`1:14: "@" is not allowed in a query`}},
		{"two conditions with no operator", `meaning("a") words("b")`, []string{"1:14: Join the conditions with && or || before words"}},
		{"a number for a text", `meaning(1)`, []string{"1:1: meaning takes one text in quotes"}},
		{"an operator outside the language", `meaning("a") + 1`, []string{"1:14: The operator + is not allowed in a query"}},
		{"a misspelled member function", `path.startsWth("books/")`, []string{"1:6: Unknown function startsWth: did you mean startsWith?"}},
		{"a function named with an underscore", `_words("river")`, []string{"1:1: Unknown function _words: did you mean words?"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			problems := getProblems(t, tt.text)
			got := make([]string, len(problems))
			for i, p := range problems {
				got[i] = searchquery.FormatProblem(p)
			}
			if !slices.Equal(got, tt.want) {
				t.Errorf("problems\n%s\nwant\n%s", strings.Join(got, "\n"), strings.Join(tt.want, "\n"))
			}
		})
	}
}

func TestConsecutiveLettersOutsideQuotesAreOneProblemOverTheirRange(t *testing.T) {
	problems := getProblems(t, `meaning("a") && Кришна`)
	want := searchquery.Problem{
		Position: searchquery.Position{Line: 1, Column: 17}, End: searchquery.Position{Line: 1, Column: 22},
		Message: "Put this text in quotes: Кришна",
	}
	if len(problems) != 1 || problems[0] != want {
		t.Errorf("problems %+v, want %+v", problems, want)
	}
}
