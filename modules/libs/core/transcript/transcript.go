// Package transcript is what a model heard in a recording, written down as
// WebVTT.
//
//	WEBVTT
//
//	NOTE heard 9100
//
//	00:00:01.500 --> 00:00:04.200
//	what was said
//
//	00:00:04.200 --> 00:00:09.100
//	what was said next
//
// The format is the W3C one, so the file opens in a player, shows the words
// against the recording in a browser, and is read by anything a person already
// has. A run stopped part way says how far it got in a NOTE, which every reader
// of the format passes over.
//
// The timings are not the words. Reading takes them out, so an offset in what
// comes back is an offset in the speech, and a chunk cut from it holds what was
// said and not the bookkeeping around it.
//
// It is pure: no filesystem, no clock, no model.
package transcript

import (
	"bytes"
	"fmt"
	"sort"
	"strconv"
	"strings"
)

// Head is the line every WebVTT file begins with.
const Head = "WEBVTT"

// arrow separates the two timings of a cue.
const arrow = "-->"

// A Cue is one stretch of speech: what was said, when it was said, and where it
// stands in the words a transcript reads as. At is filled by reading, because
// only then is there a text for it to be an offset into.
type Cue struct {
	Text string
	From int
	To   int
	At   int
}

// Marshal is the artifact for a run of cues.
//
// A cue carrying no words is not written: silence is not something a person
// scrolls past, and a timing over nothing is a moment the recording never had.
func Marshal(cues []Cue) []byte {
	var out strings.Builder
	out.WriteString(Head)
	out.WriteString("\n")
	for _, cue := range cues {
		text := strings.TrimSpace(cue.Text)
		if text == "" {
			continue
		}
		fmt.Fprintf(&out, "\n%s %s %s\n%s\n", Stamp(cue.From), arrow, Stamp(cue.To), text)
	}
	return []byte(out.String())
}

// Parse is an artifact, as the words it holds and the cues they came from.
//
// The words of a cue stand one to a line, which is what a person reading the
// transcript sees and what a chunk is cut out of.
func Parse(raw []byte) (string, []Cue) {
	var out strings.Builder
	var cues []Cue
	for _, block := range strings.Split(strings.ReplaceAll(string(raw), "\r\n", "\n"), "\n\n") {
		cue, ok := parse(block)
		if !ok {
			continue
		}
		if out.Len() > 0 {
			out.WriteString("\n")
		}
		cue.At = out.Len()
		out.WriteString(cue.Text)
		cues = append(cues, cue)
	}
	return out.String(), cues
}

// parse is one block of the file as a cue, and whether it is one. The header,
// a note and anything a later version of the format adds are not.
func parse(block string) (Cue, bool) {
	lines := strings.Split(strings.Trim(block, "\n"), "\n")
	for at, line := range lines {
		before, after, found := strings.Cut(line, arrow)
		if !found {
			// A cue may be named on the line above its timing. Anything else
			// standing there is not a cue.
			if at > 0 || strings.HasPrefix(line, Head) || strings.HasPrefix(line, "NOTE") {
				return Cue{}, false
			}
			continue
		}
		from, ok := parseStamp(before)
		if !ok {
			return Cue{}, false
		}
		// Cue settings may follow the second timing, separated by a space. A
		// timing with nothing after the arrow is a line somebody was still
		// writing.
		second := strings.Fields(after)
		if len(second) == 0 {
			return Cue{}, false
		}
		to, ok := parseStamp(second[0])
		if !ok {
			return Cue{}, false
		}
		text := strings.TrimSpace(strings.Join(lines[at+1:], "\n"))
		if text == "" {
			return Cue{}, false
		}
		return Cue{Text: text, From: from, To: to}, true
	}
	return Cue{}, false
}

// Stamp is a millisecond as the format writes it: hours, minutes, seconds and
// thousandths.
func Stamp(ms int) string {
	if ms < 0 {
		ms = 0
	}
	return fmt.Sprintf("%02d:%02d:%02d.%03d", ms/3600000, ms/60000%60, ms/1000%60, ms%1000)
}

// Clock is a moment of a recording as a person reads one: the way a player
// writes where it stands. An hour that is not there is not written.
func Clock(ms int) string {
	whole := max(ms, 0) / 1000
	if hours := whole / 3600; hours > 0 {
		return fmt.Sprintf("%d:%02d:%02d", hours, whole/60%60, whole%60)
	}
	return fmt.Sprintf("%d:%02d", whole/60, whole%60)
}

// parseStamp is a timing the format writes. The hours are optional, which is
// what the format says and what other tools write.
func parseStamp(raw string) (int, bool) {
	parts := strings.Split(strings.TrimSpace(raw), ":")
	if len(parts) < 2 || len(parts) > 3 {
		return 0, false
	}
	ms := 0
	for _, part := range parts[:len(parts)-1] {
		n, err := strconv.Atoi(part)
		if err != nil {
			return 0, false
		}
		ms = ms*60 + n
	}
	ms *= 60000

	seconds, thousandths, found := strings.Cut(parts[len(parts)-1], ".")
	if !found {
		return 0, false
	}
	s, err := strconv.Atoi(seconds)
	if err != nil {
		return 0, false
	}
	t, err := strconv.Atoi(thousandths)
	if err != nil || len(thousandths) != 3 {
		return 0, false
	}
	return ms + s*1000 + t, true
}

// At is where a run of the words sits: the cues it falls in, in the order they
// were spoken. A run crossing a silence is in both of them.
func At(cues []Cue, start, length int) []Cue {
	if length <= 0 || len(cues) == 0 {
		return nil
	}
	end := start + length

	// The first cue that reaches into the run. A cue before it ends before the
	// run begins.
	at := sort.Search(len(cues), func(i int) bool {
		return cues[i].At+len(cues[i].Text) > start
	})

	var out []Cue
	for ; at < len(cues) && cues[at].At < end; at++ {
		out = append(out, cues[at])
	}
	return out
}

// Plays is the millisecond a run of the words is played from, and whether any
// cue holds it. A run no cue holds is nowhere to play.
func Plays(cues []Cue, start, length int) (int, bool) {
	found := At(cues, start, length)
	if len(found) == 0 {
		return 0, false
	}
	return found[0].From, true
}

// Heard is the note a run stopped part way leaves: how many milliseconds of the
// recording have been written down. It stands after the cues it claims, so a
// batch that did not land whole is one no note claims.
func Heard(ms int) []byte {
	return []byte(fmt.Sprintf("\nNOTE heard %d\n", ms))
}

// ByHand is the note a transcript a person wrote carries.
const ByHand = "NOTE by hand"

// Hand marks a transcript as the words a person put there. A transcript
// carrying it is left as they left it.
func Hand() []byte {
	return []byte("\n" + ByHand + "\n")
}

// Written says whether a person wrote these words. The mark is a note of its
// own, and the same words spoken in a cue are speech.
func Written(raw []byte) bool {
	for at := 0; at <= len(raw)-len(ByHand); {
		found := bytes.Index(raw[at:], []byte(ByHand))
		if found < 0 {
			return false
		}
		if found += at; begins(raw, found) {
			return true
		}
		at = found + 1
	}
	return false
}

// Reached is how far a run before this one got, and where the last note about
// it ends. A file carrying none is a recording nothing has listened to.
func Reached(raw []byte) (ms, end int) {
	note := []byte("NOTE heard ")
	// Each note is looked at once, and only what stands between it and the one
	// after it is read.
	for end := len(raw); ; {
		at := bytes.LastIndex(raw[:end], note)
		if at < 0 {
			return 0, 0
		}
		line := raw[at+len(note) : end]
		end = at
		stop := bytes.IndexByte(line, '\n')
		if stop < 0 || !begins(raw, at) {
			continue
		}
		ms, err := strconv.Atoi(strings.TrimSpace(string(line[:stop])))
		if err != nil {
			continue
		}
		return ms, at + len(note) + stop + 1
	}
}

// begins says whether a byte is where a block of the file starts: the top of
// it, or the line after a blank one. A note stands at the top of its own block.
func begins(raw []byte, at int) bool {
	if at == 0 {
		return true
	}
	if raw[at-1] != '\n' {
		return false
	}
	blank := at - 1
	if blank > 0 && raw[blank-1] == '\r' {
		blank--
	}
	return blank > 0 && raw[blank-1] == '\n'
}
