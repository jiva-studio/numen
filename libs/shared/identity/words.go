package identity

import (
	"crypto/sha256"
	_ "embed"
	"fmt"
	"io"
	"strings"
)

const (
	WordCount   = 12
	entropySize = 16
	wordBits    = 11
)

//go:embed english.txt
var englishText string

func listEnglishWords() []string {
	return strings.Split(strings.TrimSuffix(englishText, "\n"), "\n")
}

func indexEnglishWords() map[string]int {
	index := make(map[string]int, 2048)
	for i, word := range listEnglishWords() {
		index[word] = i
	}
	return index
}

// RecoveryWords holds the entropy of the 12 words. It prints no word.
type RecoveryWords struct{ entropy [entropySize]byte }

func NewRecoveryWords(random io.Reader) (RecoveryWords, error) {
	var words RecoveryWords
	if _, err := io.ReadFull(random, words.entropy[:]); err != nil {
		return RecoveryWords{}, fmt.Errorf("identity: read entropy: %w", err)
	}
	return words, nil
}

// ParseRecoveryWords takes the 12 whole words in order, trims and lower-cases each.
func ParseRecoveryWords(words []string) (RecoveryWords, error) {
	if len(words) != WordCount {
		return RecoveryWords{}, fmt.Errorf("%w: got %d", ErrWordCount, len(words))
	}
	var packed [entropySize + 1]byte
	index := indexEnglishWords()
	for i, word := range words {
		at, ok := index[strings.ToLower(strings.TrimSpace(word))]
		if !ok {
			return RecoveryWords{}, fmt.Errorf("word %d: %w", i+1, ErrUnknownWord)
		}
		writeBits(packed[:], i*wordBits, wordBits, at)
	}
	var parsed RecoveryWords
	copy(parsed.entropy[:], packed[:entropySize])
	if parsed.pack()[entropySize] != packed[entropySize] {
		return RecoveryWords{}, ErrChecksum
	}
	return parsed, nil
}

func (w RecoveryWords) pack() [entropySize + 1]byte {
	var packed [entropySize + 1]byte
	copy(packed[:], w.entropy[:])
	sum := sha256.Sum256(w.entropy[:])
	packed[entropySize] = sum[0] & 0xF0
	return packed
}

func (w RecoveryWords) List() []string {
	packed := w.pack()
	list := listEnglishWords()
	out := make([]string, WordCount)
	for i := range out {
		out[i] = list[readBits(packed[:], i*wordBits, wordBits)]
	}
	return out
}

func (w RecoveryWords) String() string { return "[recovery words]" }

func (w RecoveryWords) GoString() string { return w.String() }

func (w RecoveryWords) DeriveRecoveryKey() (RecoveryKey, error) {
	secret := deriveKey(w.entropy[:], labelRecovery)
	return newRecoveryKey(deriveKey(secret[:], labelRecoverySign), deriveKey(secret[:], labelRecoveryBox))
}

func readBits(b []byte, offset, count int) int {
	value := 0
	for i := range count {
		position := offset + i
		value = value<<1 | int(b[position/8]>>(7-position%8)&1)
	}
	return value
}

func writeBits(b []byte, offset, count, value int) {
	for i := range count {
		if value>>(count-1-i)&1 == 1 {
			position := offset + i
			b[position/8] |= 1 << (7 - position%8)
		}
	}
}
