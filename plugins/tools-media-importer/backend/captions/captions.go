// Package captions reads and writes the captions an address publishes as a WebVTT file.
package captions

import (
	"strconv"
	"strings"

	"github.com/jiva-studio/numen/plugins/tools-media-importer/backend/download"
)

// head is the line a WebVTT file begins with.
const head = "WEBVTT"

// Marshal is the cues as a WebVTT file. A cue carrying no words is not written.
func Marshal(cues []download.Cue) []byte {
	var b strings.Builder
	b.WriteString(head + "\n")
	for _, cue := range cues {
		words := strings.TrimSpace(cue.Text)
		if words == "" {
			continue
		}
		b.WriteString("\n" + formatStamp(cue.From) + " --> " + formatStamp(cue.To) + "\n" + words + "\n")
	}
	return []byte(b.String())
}

// Parse is the cues of a WebVTT file. The header, a note and a block with no timing are not cues.
func Parse(raw []byte) []download.Cue {
	var cues []download.Cue
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
		cues = append(cues, download.Cue{Text: body, From: from, To: to})
	}
	return cues
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
