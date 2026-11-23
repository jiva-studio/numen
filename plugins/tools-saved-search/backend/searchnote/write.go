package searchnote

import (
	"errors"
	"strings"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/markdown"
)

const typeKey = "type"

var errSecondBlock = errors.New(problemSecondBlock)

// Create returns the bytes of a new saved search note: the id and the type in the frontmatter, the title
// when the file name cannot carry it (exact false), and the query in a numen-query block.
func Create(id string, title string, exact bool, definition Definition) ([]byte, error) {
	doc, err := markdown.Open(markdown.Create(id, "\n"+formatBlock(definition.Query)))
	if err != nil {
		return nil, err
	}
	if err := doc.SetScalar(typeKey, string(NoteType)); err != nil {
		return nil, err
	}
	if !exact {
		if err := doc.SetTitle(title); err != nil {
			return nil, err
		}
	}
	return doc.Bytes(), nil
}

// Apply returns the bytes of the note with the query written. The query replaces the lines inside the
// numen-query block, or is added at the end of a body that has none; the prose outside the block and the
// frontmatter are kept.
func Apply(raw []byte, query string) ([]byte, error) {
	doc, err := markdown.Open(raw)
	if err != nil {
		return nil, err
	}
	if err := setQuery(doc, query); err != nil {
		return nil, err
	}
	return doc.Bytes(), nil
}

func setQuery(doc *markdown.Document, query string) error {
	body := doc.Body()
	blocks := findQueryBlocks(body)
	switch {
	case len(blocks) > 1:
		return errSecondBlock
	case len(blocks) == 0:
		separator := "\n"
		if body != "" && !strings.HasSuffix(body, "\n") {
			separator = "\n\n"
		}
		return doc.ReplaceBody(len(body), len(body), separator+formatBlock(query))
	}
	found := blocks[0]
	if found.isClosed && getFenceLength(query, found.mark) <= found.length {
		start, end := found.opening.end, found.closing.start
		return doc.ReplaceBody(start, end, query+"\n")
	}
	end := len(body)
	if found.isClosed {
		end = found.closing.end
	}
	return doc.ReplaceBody(found.opening.start, end, formatBlock(query))
}
