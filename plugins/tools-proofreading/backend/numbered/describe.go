package numbered

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

func DescribeSpeech(lines []Line) string {
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

func getOpeningWords(response []string, words int) string {
	if len(response) <= words {
		return strings.Join(response, " ")
	}
	return strings.Join(response[:words], " ") + " …"
}

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

func isSentenceEnd(word string) bool {
	word = strings.TrimRightFunc(word, func(r rune) bool {
		return unicode.In(r, unicode.Pe, unicode.Pf) || r == '"' || r == '\''
	})
	last, _ := utf8.DecodeLastRuneInString(word)
	return strings.ContainsRune(".!?…", last) && !isShortening(word)
}

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
