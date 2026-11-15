package searchnote

import (
	"errors"
	"strings"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/markdown"
)

const typeKey = "type"

var errSecondBlock = errors.New(problemSecondBlock)

// Change is what saved_search_update changes; a nil field stays as written.
type Change struct {
	Query, Rerank *string
	Limit         *int
}

// Create returns the bytes of a new saved search note: the id and the type in the frontmatter, the title
// when the file name cannot carry it (exact false), the rerank text and the limit when set, and the query
// in a numen-query block.
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
	if err := setRerank(doc, definition.Rerank); err != nil {
		return nil, err
	}
	if err := setLimit(doc, definition.Limit); err != nil {
		return nil, err
	}
	return doc.Bytes(), nil
}

// Apply returns the bytes of the note with the change written. A new query replaces the lines inside the
// numen-query block, or is added at the end of a body that has none; the prose outside the block and the
// other frontmatter keys are kept. A rerank text of "" and a limit of 0 remove their key.
func Apply(raw []byte, change Change) ([]byte, error) {
	doc, err := markdown.Open(raw)
	if err != nil {
		return nil, err
	}
	if change.Query != nil {
		if err := setQuery(doc, *change.Query); err != nil {
			return nil, err
		}
	}
	if change.Rerank != nil {
		if err := setRerank(doc, *change.Rerank); err != nil {
			return nil, err
		}
	}
	if change.Limit != nil {
		if err := setLimit(doc, *change.Limit); err != nil {
			return nil, err
		}
	}
	return doc.Bytes(), nil
}

func setRerank(doc *markdown.Document, rerank string) error {
	if strings.TrimSpace(rerank) == "" {
		return doc.SetScalar(rerankKey, "")
	}
	return doc.SetScalar(rerankKey, rerank)
}

func setLimit(doc *markdown.Document, limit int) error {
	if limit <= 0 {
		return doc.SetScalar(limitKey, "")
	}
	return doc.SetValue(limitKey, limit)
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
		return doc.SpliceBody(len(body), len(body), separator+formatBlock(query))
	}
	found := blocks[0]
	if found.isClosed && getFenceLength(query, found.mark) <= found.length {
		start, end := found.opening.end, found.closing.start
		return doc.SpliceBody(start, end, query+"\n")
	}
	end := len(body)
	if found.isClosed {
		end = found.closing.end
	}
	return doc.SpliceBody(found.opening.start, end, formatBlock(query))
}
