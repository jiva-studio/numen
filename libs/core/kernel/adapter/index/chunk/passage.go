package chunk

import (
	"fmt"
	"strconv"

	"github.com/jiva-studio/numen/libs/note/domain"
)

// Passage is one chunk, and where its text is read from.
//
// chunk.
type Passage struct {
	Chunk       int64
	Path        string
	Start       int
	Length      int
	Location    string
	Parent      int64
	Producer    string
	ContentHash string
	ChunkHash   string
	MTime       int64
	Source      int64
}

// ID is how the core addresses the chunk on a row. This is the one place a
// chunk's identity is spelled, and the core carries it back untouched.
func ID(row int64) domain.ChunkID {
	return domain.ChunkID(strconv.FormatInt(row, 10))
}

// Row is the chunk an identifier names, and an error where it names none this
// index handed out.
func Row(id domain.ChunkID) (int64, error) {
	row, err := strconv.ParseInt(string(id), 10, 64)
	if err != nil {
		return 0, fmt.Errorf("%q is no chunk of this index", id)
	}
	return row, nil
}
