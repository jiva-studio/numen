package searchquery

import (
	"errors"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/searchquery"
	"github.com/jiva-studio/numen/libs/core/kernel/domain/source"
)

type warnings struct {
	list []string
}

func (w *warnings) add(text string) {
	w.list = append(w.list, text)
}

func getWordsError(req *searchquery.Evaluation, index int, err error) error {
	if !isInvalidWords(err) {
		return err
	}
	position, _ := findConditionPosition(req.Query.Root, index)
	return &searchquery.ParseError{Problems: []searchquery.Problem{{Position: position, Message: messageInvalidWords}}}
}

func isInvalidWords(err error) bool {
	return errors.Is(err, source.ErrInvalidWords)
}

func findConditionPosition(node searchquery.Node, index int) (searchquery.Position, bool) {
	if node.Operator == searchquery.OperatorLeaf {
		return node.Leaf.Position, node.Leaf.IsChunkLevel() && node.Leaf.ConditionIndex == index
	}
	for _, child := range node.Children {
		if position, found := findConditionPosition(child, index); found {
			return position, true
		}
	}
	return searchquery.StartPosition(), false
}
