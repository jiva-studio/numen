package slip39

import (
	_ "embed"
	"strings"
)

//go:embed wordlist.txt
var wordListText string

func wordList() []string {
	return strings.Split(strings.TrimSuffix(wordListText, "\n"), "\n")
}

func wordIndex() map[string]int {
	list := wordList()
	index := make(map[string]int, len(list))
	for i, word := range list {
		index[word] = i
	}
	return index
}
