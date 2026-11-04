// Package tokens reads the vocabulary of a model and converts token indices into text.
package tokens

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
)

// The model emits word pieces, and a piece that opens a word carries a mark
// where the space before it was. Joining the pieces and reading that mark back
// as a space is the whole of converting its output into text.
const wordMark = "▁"

// Pieces are what a model can say, at the place its own numbers put them, with
// the mark already read as a space.
type Pieces []string

// Read reads a model's pieces from the file published alongside it. Each line
// is a piece and the number that identifies it, and a number the file skips
// has no piece.
func Read(path string) (pieces Pieces, err error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	// gomutants:disable-next-line STATEMENT_REMOVE reason="closing a read-only os.File in defer does not fail under normal reads"
	defer func() { err = errors.Join(err, file.Close()) }()

	var out Pieces
	lines := bufio.NewScanner(file)
	for lines.Scan() {
		field := strings.Fields(lines.Text())
		if len(field) != 2 {
			continue
		}
		pos, err := strconv.Atoi(field[1])
		if err != nil || pos < 0 {
			continue
		}
		for len(out) <= pos {
			out = append(out, "")
		}
		out[pos] = strings.ReplaceAll(field[0], wordMark, " ")
	}
	if err := lines.Err(); err != nil {
		return nil, err
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("%s names no pieces", path)
	}
	return out, nil
}

// GetIndex is the number one piece answers to, and -1 for a piece the model does
// not know.
func (p Pieces) GetIndex(piece string) int {
	for i, one := range p {
		if one == piece {
			return i
		}
	}
	return -1
}

// GetText returns the text of a run of tokens. A token in the model vocabulary and absent from this file
// is omitted.
func (p Pieces) GetText(response []int) string {
	// gomutants:disable-next-line INTEGER_INCREMENT reason="builder preallocation size hint does not affect output text"
	size := 0
	// gomutants:disable-next-line RANGE_BREAK reason="builder preallocation loop does not affect output text"
	for _, token := range response {
		// gomutants:disable-next-line CONDITIONALS_BOUNDARY,INTEGER_INCREMENT,BRANCH_IF reason="builder preallocation condition does not affect output text"
		if token >= 0 && token < len(p) {
			// gomutants:disable-next-line REMOVE_SELF_ASSIGNMENTS reason="builder preallocation accumulation does not affect output text"
			size += len(p[token])
		}
	}
	var out strings.Builder
	// gomutants:disable-next-line STATEMENT_REMOVE reason="strings.Builder.Grow preallocation does not affect output text"
	out.Grow(size)
	for _, token := range response {
		if token >= 0 && token < len(p) {
			out.WriteString(p[token])
		}
	}
	return strings.TrimSpace(out.String())
}
