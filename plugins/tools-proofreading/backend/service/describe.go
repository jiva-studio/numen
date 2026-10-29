package service

import (
	"strings"
	"unicode"
	"unicode/utf8"
)

// Digest limits: the number of opening words quoted, the number of names
// listed, and the minimum number of occurrences for a name to be listed.
const (
	quoted = 60
	listed = 24
	recurs = 2
)

// describeSpeech is what a recording holds, in the words of its own transcript: how
// the speech opens, and the words that recur through it. A transcript saying
// nothing is described as nothing.
func describeSpeech(lines []Line) string {
	var response []string
	for _, line := range lines {
		response = append(response, strings.Fields(line.Text)...)
	}
	if len(response) == 0 {
		return ""
	}

	var out strings.Builder
	out.WriteString("The speech opens: " + getOpeningWords(response, quoted) + "\n")
	if recurring := getRecurringNames(response); len(recurring) > 0 {
		out.WriteString("\nWords recurring through it, as the machine transcribed them: " +
			strings.Join(recurring, ", ") + "\n")
	}
	return out.String()
}

// getOpeningWords is the opening words of the speech, with a mark where it was cut short.
func getOpeningWords(response []string, words int) string {
	if len(response) <= words {
		return strings.Join(response, " ")
	}
	return strings.Join(response[:words], " ") + " …"
}

// getRecurringNames returns the words of the transcript that are names: words
// capitalised somewhere other than at the start of a sentence, and occurring
// there more than once. They are in order of first occurrence, spelled as in
// the first occurrence.
func getRecurringNames(response []string) []string {
	times := map[string]int{}
	content := map[string]string{}
	var order []string

	opens := true
	for _, word := range response {
		here := opens
		opens = isSentenceEnd(word)

		cleanWord := strings.TrimFunc(word, func(r rune) bool {
			return !unicode.IsLetter(r) && !unicode.IsMark(r)
		})
		if cleanWord == "" || here || !isName(cleanWord) {
			continue
		}
		key := strings.ToLower(cleanWord)
		if times[key] == 0 {
			order = append(order, key)
			content[key] = cleanWord
		}
		times[key]++
	}

	var out []string
	for _, key := range order {
		if times[key] < recurs {
			continue
		}
		out = append(out, content[key])
		if len(out) == listed {
			break
		}
	}
	return out
}

// isName reports whether a word is written as a name: it starts with a capital
// and has either more than one letter before an apostrophe or more than two
// letters after it, so O'Brien is a name and I'm is not.
func isName(word string) bool {
	if letter, _ := utf8.DecodeRuneInString(word); !unicode.IsUpper(letter) {
		return false
	}
	head, tail := word, ""
	if pos := strings.IndexAny(word, "'’"); pos >= 0 {
		_, runeLen := utf8.DecodeRuneInString(word[pos:])
		head, tail = word[:pos], word[pos+runeLen:]
	}
	return utf8.RuneCountInString(head) > 1 || utf8.RuneCountInString(tail) > 2
}

// or quotation mark after the stop belongs to the sentence it closes.
func isSentenceEnd(word string) bool {
	word = strings.TrimRightFunc(word, func(r rune) bool {
		return unicode.In(r, unicode.Pe, unicode.Pf) || r == '"' || r == '\''
	})
	last, _ := utf8.DecodeLastRuneInString(word)
	return strings.ContainsRune(".!?…", last) && !isShortening(word)
}

// isShortening reports whether a word ending in a period is an abbreviation:
// a capital with at most two letters before the period, or a period inside the
// word.
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
