package tokenizer_test

import (
	"encoding/base64"
	"encoding/binary"
	"slices"
	"strings"
	"testing"
)

// charsmapNode is a node of the byte trie buildCharsmapTrie lays out.
type charsmapNode struct {
	children map[byte]*charsmapNode
	value    int
}

func buildCharsmapTrie(t *testing.T, keys map[string]uint32) []uint32 {
	t.Helper()
	// The layout is a darts-clone double array, the trie format of the sentencepiece charsmap. A unit
	// holds the label in bits 0-7, has_leaf in bit 8 and the XOR offset to the base of its children in
	// bits 10-30; a leaf unit sets bit 31 and holds the value. The root is unit 0. Every node gets a
	// base of its own, so a byte never matches the child of another node.
	root := &charsmapNode{children: map[byte]*charsmapNode{}, value: -1}
	for key, value := range keys {
		node := root
		for i := range len(key) {
			next, ok := node.children[key[i]]
			if !ok {
				next = &charsmapNode{children: map[byte]*charsmapNode{}, value: -1}
				node.children[key[i]] = next
			}
			node = next
		}
		node.value = int(value)
	}
	units := []uint32{0}
	used := map[uint32]bool{0: true}
	bases := map[uint32]bool{}
	set := func(pos, unit uint32) {
		for int(pos) >= len(units) {
			units = append(units, 0)
		}
		units[pos] = unit
	}
	var place func(node *charsmapNode, pos uint32)
	place = func(node *charsmapNode, pos uint32) {
		labels := slices.Sorted(func(yield func(byte) bool) {
			for label := range node.children {
				if !yield(label) {
					return
				}
			}
		})
		slots := slices.Clone(labels)
		if node.value >= 0 {
			slots = append(slots, 0)
		}
		base := uint32(1)
		for ; ; base++ {
			if bases[base] {
				continue
			}
			free := true
			for _, label := range slots {
				if used[base^uint32(label)] {
					free = false
					break
				}
			}
			if free {
				break
			}
		}
		offset := pos ^ base
		if offset >= 1<<21 {
			t.Fatalf("the trie needs an offset of %d, past what the builder writes", offset)
		}
		bases[base] = true
		unit := units[pos] | offset<<10
		if node.value >= 0 {
			unit |= 1 << 8
		}
		set(pos, unit)
		for _, label := range slots {
			used[base^uint32(label)] = true
		}
		if node.value >= 0 {
			set(base, 1<<31|uint32(node.value))
		}
		for _, label := range labels {
			child := base ^ uint32(label)
			set(child, uint32(label))
			place(node.children[label], child)
		}
	}
	place(root, 0)
	return units
}

func encodeCharsmap(units []uint32, strs []byte) string {
	// precompiled_charsmap is the trie size in bytes as a little-endian uint32, the trie units, then
	// the NUL-terminated normalized strings.
	raw := binary.LittleEndian.AppendUint32(nil, uint32(4*len(units)))
	for _, unit := range units {
		raw = binary.LittleEndian.AppendUint32(raw, unit)
	}
	return base64.StdEncoding.EncodeToString(append(raw, strs...))
}

func buildCharsmap(t *testing.T, mapping map[string]string) string {
	t.Helper()
	var strs []byte
	keys := map[string]uint32{}
	for _, key := range slices.Sorted(func(yield func(string) bool) {
		for key := range mapping {
			if !yield(key) {
				return
			}
		}
	}) {
		keys[key] = uint32(len(strs))
		strs = append(append(strs, mapping[key]...), 0)
	}
	return encodeCharsmap(buildCharsmapTrie(t, keys), strs)
}

func writePrecompiledVariant(t *testing.T, charsmap string) string {
	t.Helper()
	return writeVariant(t, func(file map[string]any) {
		file["normalizer"] = map[string]any{"type": "Precompiled", "precompiled_charsmap": charsmap}
	})
}

func getTestCharsmap() map[string]string {
	return map[string]string{
		"Ｈ":  "H",
		"ｅ":  "e",
		"ｌ":  "l",
		"ｏ":  "o",
		"ﬁ":  "fi",
		"é": "é",
	}
}

func TestEncodeNormalizesGraphemesThroughThePrecompiledCharsmap(t *testing.T) {
	tok := loadTokenizer(t, writePrecompiledVariant(t, buildCharsmap(t, getTestCharsmap())))
	checkEncode(t, tok, "Ｈｅｌｌｏ", getSingleIDs(t, "▁Hello"))
	checkEncode(t, tok, "ﬁ", getSingleIDs(t, "▁fi"))
	checkEncode(t, tok, "é", getSingleIDs(t, "▁é"))
	checkEncode(t, tok, "ж", getSingleIDs(t, "▁ж"))
}

// A cluster under 6 bytes is looked up whole and replaced whole by its first prefix match, as
// spm_precompiled does: ﬁ and a combining acute (5 bytes) become fi.
func TestEncodeReplacesAShortClusterByItsFirstPrefixMatch(t *testing.T) {
	tok := loadTokenizer(t, writePrecompiledVariant(t, buildCharsmap(t, getTestCharsmap())))
	checkEncode(t, tok, "ﬁ́", getSingleIDs(t, "▁fi"))
}

// A cluster of 6 bytes or more is looked up rune by rune: ﬁ becomes fi, and the two combining
// acutes, which the charsmap does not hold, stay and become one unknown.
func TestEncodeLooksUpALongClusterRuneByRune(t *testing.T) {
	tok := loadTokenizer(t, writePrecompiledVariant(t, buildCharsmap(t, getTestCharsmap())))
	checkEncode(t, tok, "ﬁ́́", getSingleIDs(t, "▁fi", "<unk>"))
}

func TestLoadRefusesABrokenCharsmap(t *testing.T) {
	header := func(size uint32, body int) string {
		raw := binary.LittleEndian.AppendUint32(nil, size)
		return base64.StdEncoding.EncodeToString(append(raw, make([]byte, body)...))
	}
	for name, charsmap := range map[string]string{
		"not base64":                    "!!not base64!!",
		"under 4 bytes":                 base64.StdEncoding.EncodeToString([]byte{1, 2}),
		"a size that is not a multiple": header(6, 8),
		"a size past the data":          header(64, 8),
	} {
		_, err := loadWithError(t, writePrecompiledVariant(t, charsmap))
		if err == nil || !strings.Contains(err.Error(), "Precompiled") {
			t.Errorf("%s: Load error %v, want one that names Precompiled", name, err)
		}
	}
}

// A trie that points outside itself or outside the strings matches nothing: the rune is kept.
func TestEncodeReturnsTheRuneUnchangedForALookupOutsideTheCharsmap(t *testing.T) {
	outsideRoot := encodeCharsmap([]uint32{1000 << 10}, []byte("x\x00"))

	pastStrings := encodeCharsmap(buildCharsmapTrie(t, map[string]uint32{"ж": 500}), []byte("x\x00"))

	units := buildCharsmapTrie(t, map[string]uint32{"ж": 0})
	first := (units[0] >> 10) ^ uint32("ж"[0])
	units[first] = uint32("ж"[0]) | (1<<21-1)<<10
	outsideChild := encodeCharsmap(units, []byte("x\x00"))

	for name, charsmap := range map[string]string{
		"a root offset past the trie":  outsideRoot,
		"a value past the strings":     pastStrings,
		"a child offset past the trie": outsideChild,
	} {
		tok := loadTokenizer(t, writePrecompiledVariant(t, charsmap))
		if got, want := tok.Encode("ж"), getSingleIDs(t, "▁ж"); !slices.Equal(got, want) {
			t.Errorf("%s: Encode(ж) = %v, want %v", name, got, want)
		}
	}
}
