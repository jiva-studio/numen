// Package searchnote reads and writes saved search notes: notes of type search whose body holds the query
// in a fenced block numen-query.
package searchnote

import (
	"errors"
	"fmt"
	"path"
	"strings"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/domain"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/markdown"
)

// NoteType is the frontmatter type that makes a note a saved search.
const NoteType domain.NoteType = "search"

const (
	problemNoBlock     = "the note has no " + Language + " block"
	problemSecondBlock = "the note has more than one " + Language + " block"
	problemEmptyBlock  = "the " + Language + " block is empty"
	problemNotASearch  = "the note is not a saved search (type: %s)"
	problemFrontmatter = "the frontmatter is not valid YAML"
	firstColumn        = 1
	firstLine          = 1
)

// ErrNotASearch is what a note gets that is not of type search.
var ErrNotASearch = errors.New("not a saved search")

// Definition is what a saved search note asks for: the frontmatter id, empty when the note has none; the
// frontmatter title, else the file name without .md; the text inside the numen-query block without the
// fences, as written.
type Definition struct {
	ID    string
	Title string
	Query string
}

// Problem is a reason the note cannot run. Line and Column count from 1 over the whole file.
type Problem struct {
	Line, Column int
	Message      string
}

// Read reads a saved search from the bytes of its note.
func Read(stat domain.FileStat, raw []byte) (Definition, []Problem) {
	note := markdown.Parse(stat, raw)
	definition := Definition{ID: note.ID, Title: getTitle(note, stat.Path), Query: ""}
	if note.Type != NoteType {
		return definition, []Problem{{Line: firstLine, Column: firstColumn, Message: fmt.Sprintf(problemNotASearch, note.Type)}}
	}
	doc, err := markdown.Open(raw)
	if err != nil {
		return definition, []Problem{{Line: firstLine, Column: firstColumn, Message: problemFrontmatter}}
	}
	var problems []Problem
	bodyLine := getBodyLine(raw, doc.Body())
	blocks := findQueryBlocks(doc.Body())
	switch {
	case len(blocks) == 0:
		problems = append(problems, Problem{Line: bodyLine, Column: firstColumn, Message: problemNoBlock})
	case len(blocks) > 1:
		second := bodyLine + countLines(doc.Body()[:blocks[1].opening.start])
		problems = append(problems, Problem{Line: second, Column: firstColumn, Message: problemSecondBlock})
	default:
		definition.Query = blocks[0].getText()
		if strings.TrimSpace(definition.Query) == "" {
			opening := bodyLine + countLines(doc.Body()[:blocks[0].opening.start])
			problems = append(problems, Problem{Line: opening, Column: firstColumn, Message: problemEmptyBlock})
		}
	}
	return definition, problems
}

// IsSearch reports whether the note is of type search.
func IsSearch(stat domain.FileStat, raw []byte) bool {
	return markdown.Parse(stat, raw).Type == NoteType
}

// GetQueryLine returns the line of the file, counted from 1, of the first line of the query of a saved
// search note, and false when the note has no numen-query block.
func GetQueryLine(raw []byte) (int, bool) {
	doc, err := markdown.Open(raw)
	if err != nil {
		return 0, false
	}
	blocks := findQueryBlocks(doc.Body())
	if len(blocks) == 0 {
		return 0, false
	}
	return getBodyLine(raw, doc.Body()) + countLines(doc.Body()[:blocks[0].opening.start]) + 1, true
}

func getTitle(note domain.Note, notePath string) string {
	if title, isText := note.Frontmatter["title"].(string); isText && strings.TrimSpace(title) != "" {
		return title
	}
	base := path.Base(notePath)
	return strings.TrimSuffix(base, path.Ext(base))
}

func getBodyLine(raw []byte, body string) int {
	return firstLine + countLines(string(raw[:len(raw)-len(body)]))
}

func countLines(text string) int {
	return strings.Count(text, "\n")
}
