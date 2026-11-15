package celquery_test

import (
	"strings"
	"testing"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/searchquery"
)

func TestThreeStagesParseInOrderWithTheirArguments(t *testing.T) {
	q, err := parse(`meaning("a").rerank("a").top(5).sortBy("modified")`)
	if err != nil {
		t.Fatal(err)
	}
	if len(q.Stages) != 3 {
		t.Fatalf("stages %+v, want three", q.Stages)
	}
	rerank, top, sort := q.Stages[0], q.Stages[1], q.Stages[2]
	if rerank.Kind != searchquery.StageRerank || rerank.Rerank.Query != "a" || rerank.Rerank.Window != 0 {
		t.Errorf("the first stage is %+v", rerank)
	}
	if top.Kind != searchquery.StageTop || top.Count != 5 {
		t.Errorf("the second stage is %+v", top)
	}
	if sort.Kind != searchquery.StageSortBy || sort.Sort.Key != searchquery.SortModified || !sort.Sort.IsDescending {
		t.Errorf("the third stage is %+v", sort)
	}
	if !q.IsSearchResults || len(q.Conditions) != 1 {
		t.Errorf("the query is %+v", q)
	}
}

func TestARerankWithoutAQueryTakesTheTextsOfTheConditions(t *testing.T) {
	q, err := parse(`meaning("a").rerank()`)
	if err != nil {
		t.Fatal(err)
	}
	if q.Stages[0].Rerank.Query != "a" {
		t.Errorf("the rerank query is %q, want a", q.Stages[0].Rerank.Query)
	}
	q, err = parse(`(words("tide") || meaning("sea") || !words("lake")).rerank()`)
	if err != nil {
		t.Fatal(err)
	}
	if q.Stages[0].Rerank.Query != "tide\nsea" {
		t.Errorf("the rerank query is %q, want the positive conditions", q.Stages[0].Rerank.Query)
	}
	if !hasProblem(getProblems(t, `(kind == "pdf").rerank()`), "rerank needs a query") {
		t.Error("a rerank of a document predicate without a query was accepted")
	}
}

func TestADocumentPredicateTakesStages(t *testing.T) {
	q, err := parse(`(kind == "pdf").sortBy("title", "desc").top(3)`)
	if err != nil {
		t.Fatal(err)
	}
	if q.IsSearchResults || len(q.Stages) != 2 || q.Stages[0].Sort.Key != searchquery.SortTitle || !q.Stages[0].Sort.IsDescending {
		t.Errorf("the query is %+v", q)
	}
}

func TestAWrongCountOfTopIsAProblemAtTheArgument(t *testing.T) {
	for _, text := range []string{`meaning("a").top("x")`, `meaning("a").top(0)`, `meaning("a").top(101)`} {
		problems := getProblems(t, text)
		if len(problems) != 1 || problems[0].Position != (searchquery.Position{Line: 1, Column: 18}) || !strings.Contains(problems[0].Message, "from 1 to 100") {
			t.Errorf("%s: problems %+v, want one at 1:18", text, problems)
		}
	}
	problems := getProblems(t, "meaning(\"a\")\n  .top(0)")
	if len(problems) != 1 || problems[0].Position != (searchquery.Position{Line: 2, Column: 8}) {
		t.Errorf("problems %+v, want one at 2:8", problems)
	}
}

func TestAMisspelledSortKeyIsAProblemWithASuggestion(t *testing.T) {
	problems := getProblems(t, `meaning("a").sortBy("modifed")`)
	if len(problems) != 1 || problems[0].Suggestion != "modified" || problems[0].Position.Column != 21 {
		t.Errorf("problems %+v, want one at 1:21 suggesting modified", problems)
	}
	problems = getProblems(t, `meaning("a").sortBy("path", "down")`)
	if len(problems) != 1 || problems[0].Position.Column != 29 || !strings.Contains(problems[0].Message, "direction") {
		t.Errorf("problems %+v, want one at the direction", problems)
	}
}

func TestAMisspelledStageIsUnknownWithASuggestion(t *testing.T) {
	problems := getProblems(t, `meaning("a").rerenk("a")`)
	if len(problems) != 1 || problems[0].Message != "unknown stage rerenk" || problems[0].Suggestion != searchquery.StageNameRerank {
		t.Errorf("problems %+v, want unknown stage rerenk suggesting rerank", problems)
	}
}

func TestTheWindowAndTheInstructionOfARerankAreRead(t *testing.T) {
	q, err := parse(`meaning("a").rerank("q", 70)`)
	if err != nil {
		t.Fatal(err)
	}
	if got := q.Stages[0].Rerank; got.Query != "q" || got.Window != 70 || got.Instruction != "" {
		t.Errorf("rerank(q, 70) is %+v", got)
	}
	q, err = parse(`meaning("a").rerank("q", "instr", 70)`)
	if err != nil {
		t.Fatal(err)
	}
	if got := q.Stages[0].Rerank; got.Query != "q" || got.Window != 70 || got.Instruction != "instr" {
		t.Errorf("rerank(q, instr, 70) is %+v", got)
	}
	q, err = parse(`meaning("a").rerank("q", "instr")`)
	if err != nil {
		t.Fatal(err)
	}
	if got := q.Stages[0].Rerank; got.Instruction != "instr" || got.Window != 0 {
		t.Errorf("rerank(q, instr) is %+v", got)
	}
	problems := getProblems(t, `meaning("a").rerank("q", 501)`)
	if len(problems) != 1 || problems[0].Position.Column != 26 {
		t.Errorf("rerank(q, 501): problems %+v, want one at the window", problems)
	}
}

func TestTheNinthStageIsAProblem(t *testing.T) {
	text := `meaning("a")` + strings.Repeat(".top(5)", 9)
	problems := getProblems(t, text)
	if len(problems) != 1 || problems[0].Position.Column != len(`meaning("a")`)+8*len(".top(5)")+2 || !strings.Contains(problems[0].Message, "at most 8 stages") {
		t.Errorf("problems %+v, want one at the ninth stage", problems)
	}
	if _, err := parse(`meaning("a")` + strings.Repeat(".top(5)", 8)); err != nil {
		t.Errorf("eight stages: %v", err)
	}
}

func TestAStageInsideAConditionIsAProblem(t *testing.T) {
	if !hasProblem(getProblems(t, `meaning("a").top(3) && kind == "pdf"`), "a stage comes after the whole query") {
		t.Error("a stage inside a condition was accepted")
	}
	if !hasProblem(getProblems(t, `top(3)`), "unknown function top") {
		t.Error("a stage called as a function was accepted")
	}
}
