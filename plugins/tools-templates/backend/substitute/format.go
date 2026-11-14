package substitute

import (
	"strconv"
	"strings"
	"time"
)

type token struct {
	name   string
	format func(time.Time) string
}

func getTokens() []token {
	return []token{
		{"YYYY", func(t time.Time) string { return strconv.Itoa(t.Year()) }},
		{"YY", func(t time.Time) string { return pad(t.Year() % 100) }},
		{"MMMM", func(t time.Time) string { return t.Month().String() }},
		{"MMM", func(t time.Time) string { return t.Month().String()[:3] }},
		{"MM", func(t time.Time) string { return pad(int(t.Month())) }},
		{"M", func(t time.Time) string { return strconv.Itoa(int(t.Month())) }},
		{"DD", func(t time.Time) string { return pad(t.Day()) }},
		{"D", func(t time.Time) string { return strconv.Itoa(t.Day()) }},
		{"dddd", func(t time.Time) string { return t.Weekday().String() }},
		{"ddd", func(t time.Time) string { return t.Weekday().String()[:3] }},
		{"HH", func(t time.Time) string { return pad(t.Hour()) }},
		{"H", func(t time.Time) string { return strconv.Itoa(t.Hour()) }},
		{"hh", func(t time.Time) string { return pad(getTwelveHour(t)) }},
		{"h", func(t time.Time) string { return strconv.Itoa(getTwelveHour(t)) }},
		{"mm", func(t time.Time) string { return pad(t.Minute()) }},
		{"m", func(t time.Time) string { return strconv.Itoa(t.Minute()) }},
		{"ss", func(t time.Time) string { return pad(t.Second()) }},
		{"s", func(t time.Time) string { return strconv.Itoa(t.Second()) }},
		{"A", func(t time.Time) string { return strings.ToUpper(getMeridiem(t)) }},
		{"a", getMeridiem},
	}
}

func formatTime(t time.Time, layout, fallback string) string {
	if layout == "" {
		layout = fallback
	}
	tokens := getTokens()
	var out strings.Builder
	for rest := layout; rest != ""; {
		if strings.HasPrefix(rest, "[") {
			if text, after, isClosed := strings.Cut(rest[1:], "]"); isClosed {
				out.WriteString(text)
				rest = after
				continue
			}
		}
		if token, isToken := findToken(tokens, rest); isToken {
			out.WriteString(token.format(t))
			rest = rest[len(token.name):]
			continue
		}
		out.WriteByte(rest[0])
		rest = rest[1:]
	}
	return out.String()
}

func findToken(tokens []token, text string) (token, bool) {
	for _, token := range tokens {
		if strings.HasPrefix(text, token.name) {
			return token, true
		}
	}
	return token{name: "", format: nil}, false
}

func pad(number int) string {
	if number < 10 {
		return "0" + strconv.Itoa(number)
	}
	return strconv.Itoa(number)
}

func getTwelveHour(t time.Time) int {
	if hour := t.Hour() % 12; hour != 0 {
		return hour
	}
	return 12
}

func getMeridiem(t time.Time) string {
	if t.Hour() < 12 {
		return "am"
	}
	return "pm"
}
