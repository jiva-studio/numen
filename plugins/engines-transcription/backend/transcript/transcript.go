// Package transcript is what a model heard in a recording: cues, and the text
// segments a cue is written as in the common format of text versions.
//
// A cue is one segment. Its text ends in a newline, so the joined text of a
// transcript is one cue to a line, and its selector is the cue's span of the
// recording in seconds, `{"t":[start,end]}` (W3C Media Fragments).
//
// It is pure: no filesystem, no clock, no model.
package transcript

import (
	"strconv"
	"strings"

	"github.com/jiva-studio/numen/libs/plugin-sdk-go/host/domain"
)

// Cue is one stretch of speech: the words, and the milliseconds of the recording it spans.
type Cue struct {
	Text string
	From int
	To   int
}

// GetSegments is the cues as segments of a text version. A cue carrying no
// words is not written: a timing over nothing is a moment the recording never had.
func GetSegments(cues []Cue) ([]domain.TextSegment, error) {
	out := make([]domain.TextSegment, 0, len(cues))
	for _, cue := range cues {
		text := strings.TrimSpace(cue.Text)
		if text == "" {
			continue
		}
		selector, err := domain.EncodeSelector(domain.TimeSelector{T: []float64{toSeconds(cue.From), toSeconds(cue.To)}})
		if err != nil {
			return nil, err
		}
		out = append(out, domain.TextSegment{Text: text + "\n", Heading: "", Selector: selector})
	}
	return out, nil
}

// GetCues is the segments of a text version as cues. A segment whose selector
// names no span is a cue at zero.
func GetCues(segments []domain.TextSegment) []Cue {
	out := make([]Cue, 0, len(segments))
	for _, segment := range segments {
		cue := Cue{Text: strings.TrimRight(segment.Text, "\n"), From: 0, To: 0}
		if selector, err := domain.DecodeSelector[domain.TimeSelector](segment.Selector); err == nil && len(selector.T) == 2 {
			cue.From, cue.To = toMS(selector.T[0]), toMS(selector.T[1])
		}
		out = append(out, cue)
	}
	return out
}

func toSeconds(ms int) float64 { return float64(ms) / 1000 }

func toMS(seconds float64) int { return int(seconds*1000 + 0.5) }

// FormatTimestamp is a millisecond as hours, minutes, seconds and thousandths.
func FormatTimestamp(ms int) string {
	ms = max(ms, 0)
	h, m, s, f := ms/3600000, (ms/60000)%60, (ms/1000)%60, ms%1000
	var b strings.Builder
	if h < 10 {
		b.WriteByte('0')
	}
	b.WriteString(strconv.Itoa(h))
	for _, part := range []int{m, s} {
		b.WriteByte(':')
		b.WriteByte(byte('0' + part/10))
		b.WriteByte(byte('0' + part%10))
	}
	b.WriteByte('.')
	b.WriteString(strconv.Itoa(1000 + f)[1:])
	return b.String()
}
