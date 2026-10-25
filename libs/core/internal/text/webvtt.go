package text

import (
	"strconv"
	"strings"

	"github.com/jiva-studio/numen/libs/notes/domain"
)

// webVTTHead is the line a WebVTT file begins with. The captions a link's address publishes are
// kept as WebVTT.
const webVTTHead = "WEBVTT"

// MarshalWebVTT is the captions as a WebVTT file. A cue carrying no words is not written.
func MarshalWebVTT(cues []domain.TranscriptCue) []byte {
	var b strings.Builder
	b.WriteString(webVTTHead + "\n")
	for _, cue := range cues {
		words := strings.TrimSpace(cue.Text)
		if words == "" {
			continue
		}
		b.WriteString("\n" + formatStamp(cue.From) + " --> " + formatStamp(cue.To) + "\n" + words + "\n")
	}
	return []byte(b.String())
}

// ParseWebVTT is a WebVTT file as its words, one cue to a line, and the cues they came from, each at
// the offset its words begin at. The header, a note and a block with no timing are not cues.
func ParseWebVTT(raw []byte) (string, []domain.TranscriptCue) {
	var words strings.Builder
	var cues []domain.TranscriptCue
	for _, block := range strings.Split(strings.ReplaceAll(string(raw), "\r\n", "\n"), "\n\n") {
		lines := strings.Split(strings.Trim(block, "\n"), "\n")
		timing := -1
		for i, line := range lines {
			if strings.Contains(line, "-->") {
				timing = i
				break
			}
		}
		if timing < 0 || timing > 1 {
			continue
		}
		from, to, ok := parseTiming(lines[timing])
		body := strings.TrimSpace(strings.Join(lines[timing+1:], "\n"))
		if !ok || body == "" {
			continue
		}
		if len(cues) > 0 {
			words.WriteByte('\n')
		}
		cues = append(cues, domain.TranscriptCue{Text: body, From: from, To: to, Offset: words.Len()})
		words.WriteString(body)
	}
	return words.String(), cues
}

func parseTiming(line string) (from, to int, ok bool) {
	before, after, _ := strings.Cut(line, "-->")
	fields := strings.Fields(after)
	if len(fields) == 0 {
		return 0, 0, false
	}
	from, isFrom := parseStamp(strings.TrimSpace(before))
	to, isTo := parseStamp(fields[0])
	return from, to, isFrom && isTo
}

// parseStamp is a timing as the format writes it, the hours optional, in milliseconds.
func parseStamp(stamp string) (int, bool) {
	clock, thousandths, found := strings.Cut(stamp, ".")
	if !found || len(thousandths) != 3 {
		return 0, false
	}
	ms, err := strconv.Atoi(thousandths)
	if err != nil {
		return 0, false
	}
	parts := strings.Split(clock, ":")
	if len(parts) < 2 || len(parts) > 3 {
		return 0, false
	}
	seconds := 0
	for _, part := range parts {
		n, err := strconv.Atoi(part)
		if err != nil || n < 0 {
			return 0, false
		}
		seconds = seconds*60 + n
	}
	return seconds*1000 + ms, true
}

// formatStamp is a millisecond as the format writes it: hours, minutes, seconds and thousandths.
func formatStamp(ms int) string {
	ms = max(ms, 0)
	h, m, s, f := ms/3600000, ms/60000%60, ms/1000%60, ms%1000
	return pad(h, 2) + ":" + pad(m, 2) + ":" + pad(s, 2) + "." + pad(f, 3)
}

func pad(n, width int) string {
	digits := strconv.Itoa(n)
	return strings.Repeat("0", max(width-len(digits), 0)) + digits
}
