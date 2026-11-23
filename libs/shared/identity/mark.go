package identity

import (
	"encoding/hex"
	"fmt"
	"strconv"
	"strings"
)

const markHeadSize = 6

// Mark is where a chain stood once: how many entries it had and the start of the hash of its last one. A person can copy it from a
// printed sheet, and a chain that does not contain it is older than the sheet or a different history.
type Mark struct {
	Length int
	Head   [markHeadSize]byte
}

// GetMark returns the mark of the chain as it is now.
func (c Chain) GetMark() Mark {
	if len(c.entries) == 0 {
		return Mark{}
	}
	return c.getMarkAt(len(c.entries))
}

func (c Chain) getMarkAt(length int) Mark {
	head := hashEntry(c.entries[length-1])
	return Mark{Length: length, Head: [markHeadSize]byte(head[:markHeadSize])}
}

// Contains reports whether the chain has at least m.Length entries and its entry at that length ends with the head of the mark.
func (c Chain) Contains(m Mark) bool {
	return m.Length >= 1 && len(c.entries) >= m.Length && c.getMarkAt(m.Length) == m
}

// String returns the length and the head in hex, as "7-3FA912C077B1".
func (m Mark) String() string {
	return strconv.Itoa(m.Length) + "-" + strings.ToUpper(hex.EncodeToString(m.Head[:]))
}

// ParseMark reads what Mark.String wrote; spaces and the case of the letters do not matter.
func ParseMark(text string) (Mark, error) {
	text = strings.Join(strings.Fields(text), "")
	length, head, found := strings.Cut(text, "-")
	count, err := strconv.Atoi(length)
	raw, decodeErr := hex.DecodeString(head)
	if !found || err != nil || count < 1 || decodeErr != nil || len(raw) != markHeadSize {
		return Mark{}, fmt.Errorf("%w: a chain mark is a number, a dash and %d bytes in hex", ErrMalformed, markHeadSize)
	}
	return Mark{Length: count, Head: [markHeadSize]byte(raw)}, nil
}

// IsZero reports whether the mark is empty.
func (m Mark) IsZero() bool { return m == Mark{} }
