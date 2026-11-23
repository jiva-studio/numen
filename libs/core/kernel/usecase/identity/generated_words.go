package identity

import (
	"crypto/subtle"
	"slices"
	"strings"
	"sync"
)

// generatedWords holds the words the core made for the identity that is being created, until they are used or replaced.
type generatedWords struct {
	mu    sync.Mutex
	words []string
}

func (g *generatedWords) put(words []string) {
	g.mu.Lock()
	defer g.mu.Unlock()
	clear(g.words)
	g.words = slices.Clone(words)
}

// take reports whether words are the held words, and forgets them when they are.
func (g *generatedWords) take(words []string) bool {
	g.mu.Lock()
	defer g.mu.Unlock()
	if len(g.words) == 0 || len(words) != len(g.words) {
		return false
	}
	isSame := subtle.ConstantTimeCompare([]byte(strings.Join(normalizeWords(words), " ")), []byte(strings.Join(g.words, " "))) == 1
	if isSame {
		clear(g.words)
		g.words = nil
	}
	return isSame
}

func normalizeWords(words []string) []string {
	out := make([]string, len(words))
	for i, word := range words {
		out[i] = strings.ToLower(strings.TrimSpace(word))
	}
	return out
}
