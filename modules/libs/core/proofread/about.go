package proofread

import (
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/jiva-studio/numen/modules/libs/core/internal/transcript"
)

// How much of a transcript the digest carries: the words of its opening it
// quotes, the words it lists, and how often one is said to be listed.
const (
	quoted = 60
	listed = 24
	recurs = 2
)

// About is what a recording holds, in the words of its own transcript: how the
// speech opens, and the words that recur through it.
//
// A transcript saying nothing is described as nothing.
func About(cues []transcript.Cue) string {
	var said []string
	for _, cue := range cues {
		said = append(said, strings.Fields(cue.Text)...)
	}
	if len(said) == 0 {
		return ""
	}

	var out strings.Builder
	out.WriteString("The speech opens: " + first(said, quoted) + "\n")
	if recurring := names(said); len(recurring) > 0 {
		out.WriteString("\nWords recurring through it, as the machine transcribed them: " +
			strings.Join(recurring, ", ") + "\n")
	}
	return out.String()
}

// first is the opening words of the speech, with a mark where it was cut short.
func first(said []string, words int) string {
	if len(said) <= words {
		return strings.Join(said, " ")
	}
	return strings.Join(said[:words], " ") + " …"
}

// names are the words the speech carries as names: a word standing capitalised
// somewhere other than at the opening of a sentence, and standing there more
// than once. They come in the order they are first said, written as they were
// first said.
func names(said []string) []string {
	times := map[string]int{}
	written := map[string]string{}
	var order []string

	opens := true
	for _, word := range said {
		here := opens
		opens = closes(word)

		bare := strings.TrimFunc(word, func(r rune) bool {
			return !unicode.IsLetter(r) && !unicode.IsMark(r)
		})
		if bare == "" || here || !isName(bare) {
			continue
		}
		key := strings.ToLower(bare)
		if times[key] == 0 {
			order = append(order, key)
			written[key] = bare
		}
		times[key]++
	}

	var out []string
	for _, key := range order {
		if times[key] < recurs {
			continue
		}
		out = append(out, written[key])
		if len(out) == listed {
			break
		}
	}
	return out
}

// isName says whether a word is written the way a name is: a capital, and
// either more than one letter before whatever is stuck to it with an apostrophe
// or more than two after it, so that O'Brien is a name and I'm is not.
func isName(bare string) bool {
	if letter, _ := utf8.DecodeRuneInString(bare); !unicode.IsUpper(letter) {
		return false
	}
	head, tail := bare, ""
	if at := strings.IndexAny(bare, "'’"); at >= 0 {
		_, wide := utf8.DecodeRuneInString(bare[at:])
		head, tail = bare[:at], bare[at+wide:]
	}
	return utf8.RuneCountInString(head) > 1 || utf8.RuneCountInString(tail) > 2
}

// closes says whether a word ends the sentence it stands in. A closing bracket
// or quotation mark after the stop belongs to the sentence it closes.
func closes(word string) bool {
	word = strings.TrimRightFunc(word, func(r rune) bool {
		return unicode.In(r, unicode.Pe, unicode.Pf) || r == '"' || r == '\''
	})
	last, _ := utf8.DecodeLastRuneInString(word)
	return strings.ContainsRune(".!?…", last) && !isShortening(word)
}

// isShortening says whether a word ending in a stop is a shortening: a capital
// and no more than two letters before the stop, or a stop standing inside it.
func isShortening(word string) bool {
	body := strings.TrimRight(word, ".")
	if body == "" {
		return false
	}
	if strings.ContainsRune(body, '.') {
		return true
	}
	letter, _ := utf8.DecodeRuneInString(body)
	return unicode.IsUpper(letter) && utf8.RuneCountInString(body) <= 2
}
