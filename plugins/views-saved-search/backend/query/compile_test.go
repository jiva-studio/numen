package query_test

import (
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/jiva-studio/numen/plugins/views-saved-search/backend/query"
)

func compile(t *testing.T, text string) query.Query {
	t.Helper()
	compiled, err := query.Compile(text)
	if err != nil {
		t.Fatalf("Compile(%q) = %v, want a query", text, err)
	}
	return compiled
}

func getProblems(t *testing.T, text string) []query.Problem {
	t.Helper()
	_, err := query.Compile(text)
	var compileError *query.CompileError
	if !errors.As(err, &compileError) {
		t.Fatalf("Compile(%q) = %v, want a *query.CompileError", text, err)
	}
	if len(compileError.Problems) == 0 {
		t.Fatalf("Compile(%q) rejected the query with no problem", text)
	}
	return compileError.Problems
}

func hasProblem(problems []query.Problem, line, column int, message string) bool {
	for _, problem := range problems {
		if problem.Line == line && problem.Column == column && strings.Contains(problem.Message, message) {
			return true
		}
	}
	return false
}

func hasMessage(problems []query.Problem, message string) bool {
	for _, problem := range problems {
		if strings.Contains(problem.Message, message) {
			return true
		}
	}
	return false
}

func TestCompileReturnsThePositionOfASyntaxError(t *testing.T) {
	problems := getProblems(t, `meaning("tides") &&`)

	if problems[0].Line != 1 || problems[0].Column != 20 || problems[0].Message == "" {
		t.Errorf("problems = %+v, want a message at 1:20, the end of the query", problems)
	}
}

func TestCompileErrorNamesEveryProblemWithItsPlace(t *testing.T) {
	_, err := query.Compile(`meaning("tides") &&`)

	if err == nil || !strings.HasPrefix(err.Error(), "the query is not valid: 1:20: ") {
		t.Errorf("error = %v, want it to begin with %q", err, "the query is not valid: 1:20: ")
	}
}

func TestCompileReturnsLineAndColumnOnTheThirdLine(t *testing.T) {
	problems := getProblems(t, "meaning(\"tides\")\n&& words(\"harbor\")\n&& similar(\"storms\")")

	// CEL places a call at its opening parenthesis.
	if !hasProblem(problems, 3, 11, "undeclared reference to 'similar'") {
		t.Errorf("problems = %+v, want undeclared reference to 'similar' at 3:11", problems)
	}
}

func TestCompileCountsColumnsInCharacters(t *testing.T) {
	problems := getProblems(t, `meaning("приливы") && similar("x")`)

	if !hasProblem(problems, 1, 30, "undeclared reference to 'similar'") {
		t.Errorf("problems = %+v, want 1:30, counted in characters, not bytes", problems)
	}
}

func TestCompileRejectsAnUnknownFunction(t *testing.T) {
	problems := getProblems(t, `similar("tides")`)

	if !hasProblem(problems, 1, 8, "undeclared reference to 'similar'") {
		t.Errorf("problems = %+v, want undeclared reference to 'similar' at 1:8", problems)
	}
}

func TestCompileRejectsAnUnknownField(t *testing.T) {
	for _, field := range []string{"tags", "modified", "title"} {
		problems := getProblems(t, `meaning("tides") && `+field+` == "x"`)

		if !hasProblem(problems, 1, 21, "undeclared reference to '"+field+"'") {
			t.Errorf("problems of %s = %+v, want undeclared reference at 1:21", field, problems)
		}
	}
}

func TestCompileRejectsAMacro(t *testing.T) {
	for _, text := range []string{`has(meaning.x) && meaning("tides")`, `size("tides") == 5 && meaning("tides")`} {
		problems := getProblems(t, text)

		if !hasMessage(problems, "undeclared reference to") {
			t.Errorf("problems of %q = %+v, want an undeclared reference", text, problems)
		}
	}
}

func TestCompileRejectsArithmeticAndOtherStringFunctions(t *testing.T) {
	for _, text := range []string{
		`words("tide" + "s")`,
		`meaning("tides") && path.endsWith(".md")`,
		`meaning("tides") && path.contains("Logs")`,
		`meaning("tides") && [1, 2].exists(x, x > 1)`,
	} {
		if _, err := query.Compile(text); err == nil {
			t.Errorf("Compile(%q) accepted the query", text)
		}
	}
}

func TestCompileRejectsANonLiteralText(t *testing.T) {
	problems := getProblems(t, `meaning(path)`)

	if !hasMessage(problems, "meaning takes a string literal") || problems[0].Line != 1 {
		t.Errorf("problems = %+v, want %q on line 1", problems, "meaning takes a string literal")
	}
	if problems := getProblems(t, `words(kind)`); !hasMessage(problems, "words takes a string literal") {
		t.Errorf("problems = %+v, want %q", problems, "words takes a string literal")
	}
}

func TestCompileRejectsAnEmptyText(t *testing.T) {
	for _, text := range []string{`meaning("")`, `meaning("  \t")`, `words("") || meaning("tides")`} {
		problems := getProblems(t, text)

		if !hasMessage(problems, "needs a text of 1 to 8192 bytes") {
			t.Errorf("problems of %q = %+v, want a text of 1 to 8192 bytes", text, problems)
		}
	}
}

func TestCompileRejectsATextOverEightThousandOneHundredNinetyTwoBytes(t *testing.T) {
	problems := getProblems(t, `meaning("`+strings.Repeat("a", 8193)+`")`)

	if !hasMessage(problems, "meaning needs a text of 1 to 8192 bytes") {
		t.Errorf("problems = %+v, want a text of 1 to 8192 bytes", problems)
	}
	compile(t, `meaning("`+strings.Repeat("a", 8192)+`")`)
}

func TestCompileSendsATextVerbatim(t *testing.T) {
	compiled := compile(t, `words(' "harbor master" NEAR tides ')`)

	if len(compiled.Leaves) != 1 || compiled.Leaves[0].Text != ` "harbor master" NEAR tides ` {
		t.Errorf("leaves = %+v, want the text as written, with its spaces", compiled.Leaves)
	}
}

func TestCompileRejectsAKindOtherThanTheKindsOfFiles(t *testing.T) {
	for _, text := range []string{
		`meaning("tides") && kind == "book"`,
		`meaning("tides") && kind in ["note", "recording"]`,
		`meaning("tides") && kind == path`,
		`meaning("tides") && kind.startsWith("n")`,
	} {
		if problems := getProblems(t, text); !hasMessage(problems, "kind is compared with") {
			t.Errorf("problems of %q = %+v, want %q", text, problems, "kind is compared with")
		}
	}
}

func TestCompileAcceptsEveryKindOfFile(t *testing.T) {
	for _, kind := range []string{"note", "pdf", "epub", "audio", "video", "image"} {
		compile(t, `meaning("tides") && kind == "`+kind+`"`)
	}
	compile(t, `meaning("tides") && kind in ["pdf", "epub"] && kind != "image"`)
}

func TestCompileRejectsPathOutsideStartsWith(t *testing.T) {
	for _, text := range []string{
		`meaning("tides") && path == "Logs/Storm.md"`,
		`meaning("tides") && path.startsWith(kind)`,
	} {
		if problems := getProblems(t, text); !hasMessage(problems, `path is used as path.startsWith("folder/")`) {
			t.Errorf("problems of %q = %+v, want the form of path", text, problems)
		}
	}
}

func TestCompileRejectsPathEndsWith(t *testing.T) {
	problems := getProblems(t, `meaning("tides") && path.endsWith(".md")`)

	if problems[0].Line != 1 {
		t.Errorf("problems = %+v, want a problem on line 1", problems)
	}
}

func TestCompileRejectsALoneLiteral(t *testing.T) {
	for _, text := range []string{`meaning("tides") || true`, `"tides"`} {
		problems := getProblems(t, text)

		if !hasMessage(problems, "the query is a condition over meaning, words, kind and path") {
			t.Errorf("problems of %q = %+v, want the condition message", text, problems)
		}
	}
}

func TestCompileRejectsSeventeenLeafNodes(t *testing.T) {
	leaves := make([]string, 17)
	for i := range leaves {
		leaves[i] = fmt.Sprintf(`meaning("tide %d")`, i)
	}

	problems := getProblems(t, strings.Join(leaves, " || "))

	if !hasMessage(problems, "a query has at most 16 meaning and words conditions") {
		t.Errorf("problems = %+v, want at most 16 conditions", problems)
	}
	compile(t, strings.Join(leaves[:16], " || "))
}

func TestCompileCountsARepeatedLeafOnce(t *testing.T) {
	leaves := make([]string, 16)
	for i := range leaves {
		leaves[i] = fmt.Sprintf(`words("tide%d")`, i)
	}
	compiled := compile(t, strings.Join(leaves, " || ")+` || words("tide0") || !words("tide1")`)

	if len(compiled.Leaves) != 16 {
		t.Errorf("leaves = %d, want 16 distinct texts", len(compiled.Leaves))
	}
}

func TestCompileRejectsAQueryWithoutAPositiveLeaf(t *testing.T) {
	for _, text := range []string{`!words("lighthouse")`, `kind == "note"`, `path.startsWith("Logs/")`, `!(meaning("tides") || words("harbor"))`} {
		problems := getProblems(t, text)

		if !hasMessage(problems, "a query needs a meaning or words condition that is not negated") {
			t.Errorf("problems of %q = %+v, want a positive condition", text, problems)
		}
	}
}

func TestCompileCollectsDistinctLeafNodesInOrderOfFirstAppearance(t *testing.T) {
	compiled := compile(t, `meaning("tides") || (words("harbor") && meaning("tides")) || !words("storm")`)

	want := []query.Leaf{
		{IsMeaning: true, Text: "tides", IsPositive: true},
		{IsMeaning: false, Text: "harbor", IsPositive: true},
		{IsMeaning: false, Text: "storm", IsPositive: false},
	}
	if fmt.Sprint(compiled.Leaves) != fmt.Sprint(want) {
		t.Errorf("leaves = %+v, want %+v", compiled.Leaves, want)
	}
}

func TestCompileTellsAMeaningLeafFromAWordsLeafOfTheSameText(t *testing.T) {
	compiled := compile(t, `meaning("tides") && words("tides")`)

	if len(compiled.Leaves) != 2 || !compiled.Leaves[0].IsMeaning || compiled.Leaves[1].IsMeaning {
		t.Errorf("leaves = %+v, want a meaning leaf and a words leaf", compiled.Leaves)
	}
}

func TestCompileKeepsALeafNegatedAndNotNegatedAsOnePositiveLeaf(t *testing.T) {
	compiled := compile(t, `meaning("tides") || !meaning("tides")`)

	if len(compiled.Leaves) != 1 || !compiled.Leaves[0].IsPositive {
		t.Errorf("leaves = %+v, want one positive leaf", compiled.Leaves)
	}
}

func collectLeafNodes(node query.Node) []query.Node {
	if node.Operator == query.OperatorLeaf {
		return []query.Node{node}
	}
	var found []query.Node
	for _, child := range node.Children {
		found = append(found, collectLeafNodes(child)...)
	}
	return found
}

func hasOperator(node query.Node, operator query.Operator) bool {
	if node.Operator == operator {
		return true
	}
	for _, child := range node.Children {
		if hasOperator(child, operator) {
			return true
		}
	}
	return false
}

func TestCompilePushesNegationOverOrToTheLeafNodes(t *testing.T) {
	compiled := compile(t, `meaning("tides") && !(meaning("lighthouses") || words("storm"))`)

	if hasOperator(compiled.Root, query.OperatorOr) {
		t.Errorf("root = %+v, want the negated or turned into an and", compiled.Root)
	}
	leaves := collectLeafNodes(compiled.Root)
	if len(leaves) != 3 {
		t.Fatalf("leaf nodes = %+v, want three", leaves)
	}
	negated := map[string]bool{}
	for _, leaf := range leaves {
		negated[compiled.Leaves[leaf.Term.LeafIndex].Text] = leaf.IsNegated
	}
	if negated["tides"] || !negated["lighthouses"] || !negated["storm"] {
		t.Errorf("negated leaves = %v, want lighthouses and storm negated, tides not", negated)
	}
}

func TestCompilePushesNegationOverAndToTheLeafNodes(t *testing.T) {
	compiled := compile(t, `meaning("tides") && !(words("storm") && kind == "note")`)

	inner := compiled.Root.Children[len(compiled.Root.Children)-1]
	if compiled.Root.Operator != query.OperatorAnd || inner.Operator != query.OperatorOr {
		t.Fatalf("root = %+v, want an and whose last child is an or", compiled.Root)
	}
	for _, leaf := range inner.Children {
		if leaf.Operator != query.OperatorLeaf || !leaf.IsNegated {
			t.Errorf("child of the or = %+v, want a negated leaf", leaf)
		}
	}
}

func TestCompileRemovesDoubleNegation(t *testing.T) {
	compiled := compile(t, `!!meaning("tides")`)

	if compiled.Root.Operator != query.OperatorLeaf || compiled.Root.IsNegated || compiled.Root.Term.Kind != query.TermMeaning {
		t.Errorf("root = %+v, want the meaning leaf, not negated", compiled.Root)
	}
	if len(compiled.Leaves) != 1 || !compiled.Leaves[0].IsPositive {
		t.Errorf("leaves = %+v, want one positive leaf", compiled.Leaves)
	}
}

func TestCompileAssignsEachLeafItsPosition(t *testing.T) {
	compiled := compile(t, "meaning(\"tides\")\n  && words(\"harbor\")")

	leaves := collectLeafNodes(compiled.Root)
	if len(leaves) != 2 {
		t.Fatalf("leaf nodes = %+v, want two", leaves)
	}
	words := leaves[1].Term
	if words.Kind != query.TermWords || words.Line != 2 || words.Column != 11 {
		t.Errorf("words term = %+v, want a words term at 2:11, the parenthesis CEL places the call at", words)
	}
}

func TestCompileReadsKindAndPathTerms(t *testing.T) {
	compiled := compile(t, `meaning("tides") && kind in ["pdf", "epub"] && path.startsWith("Atlases/")`)

	var kinds, prefix bool
	for _, leaf := range collectLeafNodes(compiled.Root) {
		switch leaf.Term.Kind {
		case query.TermKind:
			kinds = fmt.Sprint(leaf.Term.Kinds) == "[pdf epub]"
		case query.TermPath:
			prefix = leaf.Term.Prefix == "Atlases/"
		}
	}
	if !kinds || !prefix {
		t.Errorf("root = %+v, want a kind term of pdf and epub and a path term of Atlases/", compiled.Root)
	}
}
