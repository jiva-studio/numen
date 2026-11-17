package searchnote_test

import (
	"strings"
	"testing"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/domain"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/markdown"
	"github.com/jiva-studio/numen/plugins/tools-saved-search/backend/searchnote"
)

const (
	notePath = "Harbor/Harbor master- winter storms.md"
	noteID   = "01JCX6Q4ZB3W8N2V7T5R9K1M0P"
)

// The block opens on line 8 of the file: the query starts on line 9.
const harborNote = "---\n" +
	"id: 01JCX6Q4ZB3W8N2V7T5R9K1M0P\n" +
	"type: search\n" +
	"title: \"Harbor master: winter storms\"\n" +
	"rerank: Stories of the harbor master keeping the boats safe in winter storms.\n" +
	"limit: 30\n" +
	"---\n" +
	"```numen-query\n" +
	"kind in [\"note\", \"pdf\"]\n" +
	"&& meaning(\"the harbor master in a winter storm\")\n" +
	"```\n" +
	"\n" +
	"Stories for the winter evening reading.\n"

const harborQuery = "kind in [\"note\", \"pdf\"]\n&& meaning(\"the harbor master in a winter storm\")"

func read(raw string) (searchnote.Definition, []searchnote.Problem) {
	return searchnote.Read(domain.FileStat{Path: notePath}, []byte(raw))
}

func hasProblem(problems []searchnote.Problem, message string) bool {
	for _, problem := range problems {
		if problem.Message == message {
			return true
		}
	}
	return false
}

func findProblem(t *testing.T, problems []searchnote.Problem, message string) searchnote.Problem {
	t.Helper()
	for _, problem := range problems {
		if problem.Message == message {
			return problem
		}
	}
	t.Fatalf("problems = %+v, want %q", problems, message)
	return searchnote.Problem{}
}

func TestReadReadsTheQueryAndPassesOverRerankAndLimit(t *testing.T) {
	definition, problems := read(harborNote)

	if len(problems) != 0 {
		t.Fatalf("problems = %+v, want none", problems)
	}
	want := searchnote.Definition{ID: noteID, Title: "Harbor master: winter storms", Query: harborQuery}
	if definition != want {
		t.Errorf("definition = %+v, want %+v", definition, want)
	}
}

func TestReadPassesOverRerankAndLimitOfAnyKind(t *testing.T) {
	raw := "---\ntype: search\nlimit: many\nrerank: [storms, gales]\n---\n```numen-query\nwords(\"gale\")\n```\n"

	definition, problems := read(raw)

	if len(problems) != 0 || definition.Query != `words("gale")` {
		t.Errorf("definition = %+v, problems = %+v, want the query and no problem", definition, problems)
	}
}

func TestReadUsesTheFileNameAsTheTitleWithoutATitle(t *testing.T) {
	raw := "---\ntype: search\n---\n```numen-query\nwords(\"gale\")\n```\n"

	definition, problems := read(raw)

	if len(problems) != 0 || definition.Title != "Harbor master- winter storms" || definition.ID != "" {
		t.Errorf("definition = %+v, problems = %+v, want the file name as the title and nothing else set", definition, problems)
	}
}

func TestReadReadsATildeFence(t *testing.T) {
	raw := "---\ntype: search\n---\n~~~~numen-query\n" + "words(\"gale\")\n" + "```\n" + "~~~~\n"

	definition, problems := read(raw)

	if len(problems) != 0 || definition.Query != "words(\"gale\")\n```" {
		t.Errorf("definition = %+v, problems = %+v, want the lines inside the tilde fence", definition, problems)
	}
}

func TestReadReadsABlockThatRunsToTheEndOfTheBody(t *testing.T) {
	raw := "---\ntype: search\n---\n```numen-query\nwords(\"gale\")\n"

	definition, problems := read(raw)

	if len(problems) != 0 || definition.Query != "words(\"gale\")" {
		t.Errorf("definition = %+v, problems = %+v, want the unclosed block to run to the end", definition, problems)
	}
}

func TestReadIgnoresABlockOfAnotherLanguage(t *testing.T) {
	raw := "---\ntype: search\n---\n```numen-query-old\nwords(\"tide\")\n```\n```numen-query\nwords(\"gale\")\n```\n"

	definition, problems := read(raw)

	if len(problems) != 0 || definition.Query != "words(\"gale\")" {
		t.Errorf("definition = %+v, problems = %+v, want only the numen-query block", definition, problems)
	}
}

func TestReadReportsANoteWithoutABlock(t *testing.T) {
	raw := "---\ntype: search\nlimit: 5\n---\nStories of the harbor.\n"

	_, problems := read(raw)

	problem := findProblem(t, problems, "the note has no numen-query block")
	if problem.Line != 1 || problem.Column != 1 {
		t.Errorf("problem = %+v, want line 1, column 1 of the query", problem)
	}
}

func TestReadReportsASecondBlock(t *testing.T) {
	raw := "---\ntype: search\n---\n```numen-query\nwords(\"gale\")\n```\n\n```numen-query\nwords(\"tide\")\n```\n"

	_, problems := read(raw)

	problem := findProblem(t, problems, "the note has more than one numen-query block")
	if problem.Line != 1 || problem.Column != 1 {
		t.Errorf("problem = %+v, want line 1, column 1 of the query", problem)
	}
}

func TestReadReportsAnEmptyBlock(t *testing.T) {
	for name, raw := range map[string]string{
		"empty": "---\ntype: search\n---\n```numen-query\n```\n",
		"blank": "---\ntype: search\n---\n```numen-query\n   \n\n```\n",
	} {
		t.Run(name, func(t *testing.T) {
			if _, problems := read(raw); !hasProblem(problems, "the numen-query block is empty") {
				t.Errorf("problems = %+v, want the empty block reported", problems)
			}
		})
	}
}
func TestReadRejectsANoteOfAnotherType(t *testing.T) {
	raw := "---\ntype: deck\n---\n```numen-query\nwords(\"gale\")\n```\n"

	if _, problems := read(raw); !hasProblem(problems, "the note is not a saved search (type: deck)") {
		t.Errorf("problems = %+v, want the type reported", problems)
	}
}

func TestReadIgnoresProseOutsideTheBlock(t *testing.T) {
	withProse := strings.Replace(harborNote, "---\n```numen-query", "---\nThe gales of the winter of 1911.\n\n```numen-query", 1) +
		"\n```go\nfmt.Println(\"storm\")\n```\n"

	definition, problems := read(withProse)

	if len(problems) != 0 || definition.Query != harborQuery {
		t.Errorf("definition = %+v, problems = %+v, want the query alone", definition, problems)
	}
}

func TestCreateWritesANoteThatReadReadsBack(t *testing.T) {
	definition := searchnote.Definition{ID: noteID, Title: "Harbor master: winter storms", Query: harborQuery}

	raw, err := searchnote.Create(noteID, "Harbor master: winter storms", false, definition)
	if err != nil {
		t.Fatal(err)
	}

	got, problems := read(string(raw))
	if len(problems) != 0 || got != definition {
		t.Errorf("Read(Create) = %+v, problems = %+v, want %+v\n%s", got, problems, definition, raw)
	}
	note := markdown.Parse(domain.FileStat{Path: notePath}, raw)
	if note.Type != "search" || note.ID != noteID || note.Frontmatter["title"] != "Harbor master: winter storms" {
		t.Errorf("note = %+v, want type search, the id and the title in the frontmatter", note)
	}
	for _, key := range []string{"rerank", "limit"} {
		if _, isThere := note.Frontmatter[key]; isThere {
			t.Errorf("frontmatter = %v, want no %s", note.Frontmatter, key)
		}
	}
}

func TestCreateWritesNoTitleForAnExactFileName(t *testing.T) {
	definition := searchnote.Definition{ID: noteID, Title: "Winter storms", Query: "words(\"gale\")"}

	raw, err := searchnote.Create(noteID, "Winter storms", true, definition)
	if err != nil {
		t.Fatal(err)
	}

	note := markdown.Parse(domain.FileStat{Path: "Winter storms.md"}, raw)
	if _, isThere := note.Frontmatter["title"]; isThere {
		t.Errorf("frontmatter = %v, want no title", note.Frontmatter)
	}
	if !strings.Contains(string(raw), "\n```numen-query\nwords(\"gale\")\n```\n") {
		t.Errorf("note = %q, want the query in a numen-query block", raw)
	}
}

func TestCreateLengthensTheFenceForAQueryWithBackticks(t *testing.T) {
	query := "words(\"gale\")\n```\n&& words(\"quay\")"
	definition := searchnote.Definition{ID: noteID, Title: "Winter storms", Query: query}

	raw, err := searchnote.Create(noteID, "Winter storms", true, definition)
	if err != nil {
		t.Fatal(err)
	}

	if !strings.Contains(string(raw), "````numen-query\n"+query+"\n````\n") {
		t.Errorf("note = %q, want a fence of four backticks", raw)
	}
	if got, problems := searchnote.Read(domain.FileStat{Path: "Winter storms.md"}, raw); len(problems) != 0 || got.Query != query {
		t.Errorf("Read = %+v, problems = %+v, want the query back whole", got, problems)
	}
}

func TestApplyChangesTheQueryAndKeepsTheRestOfTheNote(t *testing.T) {
	query := "words(\"gale\") && words(\"quay\")"

	raw, err := searchnote.Apply([]byte(harborNote), query)
	if err != nil {
		t.Fatal(err)
	}

	definition, problems := read(string(raw))
	if len(problems) != 0 || definition.Query != query {
		t.Errorf("definition = %+v, problems = %+v, want the new query", definition, problems)
	}
	for _, kept := range []string{"\nStories for the winter evening reading.\n", "id: " + noteID, "limit: 30\n"} {
		if !strings.Contains(string(raw), kept) {
			t.Errorf("note = %q, want %q kept", raw, kept)
		}
	}
}

func TestApplyLengthensTheFenceForAQueryWithBackticks(t *testing.T) {
	query := "words(\"gale\")\n```"

	raw, err := searchnote.Apply([]byte(harborNote), query)
	if err != nil {
		t.Fatal(err)
	}

	if definition, problems := read(string(raw)); len(problems) != 0 || definition.Query != query {
		t.Errorf("definition = %+v, problems = %+v, want the query back whole", definition, problems)
	}
}
