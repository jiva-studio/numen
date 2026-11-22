// Package testsupport holds the helpers the tests of the todos plugin share.
package testsupport

import (
	"github.com/jiva-studio/numen/plugins/todos/backend/ports"
	"github.com/jiva-studio/numen/plugins/todos/backend/todo"
	"github.com/jiva-studio/numen/plugins/todos/backend/usecase/index"
)

// NewIndex returns an empty index that reads its notes with todo.ParseNote.
func NewIndex(files ports.FileReader, notes ports.NoteLister, front ports.Frontmatter) *index.Index {
	return index.New(files, notes, front, todo.ParseNote)
}
