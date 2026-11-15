package tokenizer

import (
	"bytes"
	"encoding/base64"
	"encoding/binary"
	"errors"
	"fmt"
	"strings"

	"github.com/rivo/uniseg"
)

// charsmap is the precompiled_charsmap of sentencepiece: a darts-clone double-array trie whose
// keys are UTF-8 strings and whose values are offsets of NUL-terminated replacements in strs.
type charsmap struct {
	trie []uint32
	strs []byte
}

// The bits of a darts-clone unit.
const (
	unitLabelMask = 1<<31 | 0xFF
	unitHasLeaf   = 1 << 8
	unitValueMask = 1<<31 - 1
)

// maxWholeCluster is the length in bytes below which a grapheme cluster is looked up whole, as
// spm_precompiled does.
const maxWholeCluster = 6

func parseCharsmap(encoded string) (*charsmap, error) {
	raw, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return nil, fmt.Errorf("the charsmap is not base64: %w", err)
	}
	if len(raw) < 4 {
		return nil, errors.New("the charsmap holds no trie size")
	}
	size := uint64(binary.LittleEndian.Uint32(raw))
	if size%4 != 0 {
		return nil, fmt.Errorf("the trie size %d is not a multiple of 4", size)
	}
	if size > uint64(len(raw)-4) {
		return nil, fmt.Errorf("the trie size %d exceeds the %d bytes of the charsmap", size, len(raw)-4)
	}
	trie := make([]uint32, size/4)
	for i := range trie {
		trie[i] = binary.LittleEndian.Uint32(raw[4+4*i:])
	}
	return &charsmap{trie: trie, strs: raw[4+size:]}, nil
}

func getUnitOffset(unit uint32) uint32 {
	return (unit >> 10) << ((unit & (1 << 9)) >> 6)
}

func (c *charsmap) lookup(key string) (string, bool) {
	if len(c.trie) == 0 {
		return "", false
	}
	pos := getUnitOffset(c.trie[0])
	for i := range len(key) {
		label := uint32(key[i])
		pos ^= label
		if uint64(pos) >= uint64(len(c.trie)) {
			return "", false
		}
		unit := c.trie[pos]
		if unit&unitLabelMask != label {
			return "", false
		}
		pos ^= getUnitOffset(unit)
		if unit&unitHasLeaf == 0 {
			continue
		}
		if uint64(pos) >= uint64(len(c.trie)) {
			return "", false
		}
		start := int(c.trie[pos] & unitValueMask)
		if start >= len(c.strs) {
			return "", false
		}
		end := bytes.IndexByte(c.strs[start:], 0)
		if end < 0 {
			end = len(c.strs) - start
		}
		return string(c.strs[start : start+end]), true
	}
	return "", false
}

func (c *charsmap) normalize(text string) string {
	var out strings.Builder
	out.Grow(len(text))
	graphemes := uniseg.NewGraphemes(text)
	for graphemes.Next() {
		cluster := graphemes.Str()
		if len(cluster) < maxWholeCluster {
			if replacement, ok := c.lookup(cluster); ok {
				out.WriteString(replacement)
				continue
			}
		}
		for _, r := range cluster {
			one := string(r)
			if replacement, ok := c.lookup(one); ok {
				out.WriteString(replacement)
			} else {
				out.WriteString(one)
			}
		}
	}
	return out.String()
}
