package celquery

type textScan struct {
	unclosedQuotes []int
	openParens     []int
	closeParens    []int
}

func scanText(t queryText) textScan {
	var s textScan
	for i := 0; i < len(t.runes); i++ {
		switch t.runes[i] {
		case '"', '\'':
			i = s.skipString(t.runes, i)
		case '(':
			s.openParens = append(s.openParens, i)
		case ')':
			if n := len(s.openParens); n > 0 {
				s.openParens = s.openParens[:n-1]
			} else {
				s.closeParens = append(s.closeParens, i)
			}
		}
	}
	return s
}

func (s *textScan) skipString(runes []rune, start int) int {
	if isTripleQuote(runes, start) {
		return s.skipTripleString(runes, start)
	}
	quote := runes[start]
	for i := start + 1; i < len(runes); i++ {
		switch runes[i] {
		case '\\':
			i++
		case quote:
			return i
		case '\n':
			s.unclosedQuotes = append(s.unclosedQuotes, start)
			return i
		}
	}
	s.unclosedQuotes = append(s.unclosedQuotes, start)
	return len(runes)
}

func (s *textScan) skipTripleString(runes []rune, start int) int {
	for i := start + 3; i < len(runes); i++ {
		switch {
		case runes[i] == '\\':
			i++
		case isTripleQuote(runes, i) && runes[i] == runes[start]:
			return i + 2
		}
	}
	s.unclosedQuotes = append(s.unclosedQuotes, start)
	return len(runes)
}

func isTripleQuote(runes []rune, i int) bool {
	return i+2 < len(runes) && runes[i] == runes[i+1] && runes[i] == runes[i+2]
}
