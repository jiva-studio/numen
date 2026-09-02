package transcription

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
)

// The model writes word pieces, and a piece that opens a word carries a mark
// where the space before it was. Joining the pieces and reading that mark back
// as a space is the whole of turning what it said into text.
const wordMark = "▁"

// pieces are what a model can say, at the place its own numbers put them.
type pieces []string

// tokens reads a model's pieces out of the file published beside it. Each line
// is a piece and the number that names it, and a number the file skips says
// nothing.
func tokens(path string) (pieces, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var out pieces
	lines := bufio.NewScanner(file)
	for lines.Scan() {
		field := strings.Fields(lines.Text())
		if len(field) != 2 {
			continue
		}
		at, err := strconv.Atoi(field[1])
		if err != nil || at < 0 {
			continue
		}
		for len(out) <= at {
			out = append(out, "")
		}
		out[at] = field[0]
	}
	if err := lines.Err(); err != nil {
		return nil, err
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("%s names no pieces", path)
	}
	return out, nil
}

// at is the number one piece answers to, and -1 for a piece the model does not
// know.
func (p pieces) at(piece string) int {
	for i, one := range p {
		if one == piece {
			return i
		}
	}
	return -1
}

// text is what a run of tokens says. A token the model knows and this file does
// not is left out.
func (p pieces) text(said []int) string {
	var out strings.Builder
	for _, token := range said {
		if token < 0 || token >= len(p) {
			continue
		}
		out.WriteString(p[token])
	}
	return strings.TrimSpace(strings.ReplaceAll(out.String(), wordMark, " "))
}
