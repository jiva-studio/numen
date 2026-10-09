// Package transcript is what a model heard in a recording, written down as
// WebVTT.
//
//	WEBVTT
//
//	00:00:01.500 --> 00:00:04.200
//	what was said
//
//	00:00:04.200 --> 00:00:09.100
//	what was said next
//
//	NOTE heard 9100
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
	Text   string
	From   int
	To     int
	Offset int
}

// Marshal is the artifact for a run of cues.
//
// A cue carrying no words is not written: silence is not something a person
// scrolls past, and a timing over nothing is a moment the recording never had.
func Marshal(cues []Cue) []byte {
	size := len(Head) + 1
	for _, cue := range cues {
		size += 33 + len(cue.Text)
	}
	out := make([]byte, 0, size)
	out = append(out, Head...)
	out = append(out, '\n')
	for _, cue := range cues {
		text := strings.TrimSpace(cue.Text)
		if text == "" {
			continue
		}
		out = append(out, '\n')
		out = appendStamp(out, cue.From)
		out = append(out, ' ')
		out = append(out, arrow...)
		out = append(out, ' ')
		out = appendStamp(out, cue.To)
		out = append(out, '\n')
		out = append(out, text...)
		out = append(out, '\n')
	}
	return out
}

// Parse is an artifact, as the words it holds and the cues they came from.
//
// The words of a cue stand one to a line, which is what a person reading the
// transcript sees and what a chunk is cut out of.
func Parse(raw []byte) (string, []Cue) {
	var outBuf []byte
	var cues []Cue
	var cueLens []int

	if len(raw) > 0 {
		estCues := len(raw) / 60
		cues = make([]Cue, 0, estCues)
		cueLens = make([]int, 0, estCues)
		outBuf = make([]byte, 0, len(raw)/2)
	}

	var lineBuf [16][]byte
	lines := lineBuf[:0]

	for pos := 0; pos < len(raw); {
		lineEnd := bytes.IndexByte(raw[pos:], '\n')
		var line []byte
		if lineEnd < 0 {
			line = raw[pos:]
			pos = len(raw)
		} else {
			line = raw[pos : pos+lineEnd]
			pos += lineEnd + 1
		}

		if len(line) > 0 && line[len(line)-1] == '\r' {
			line = line[:len(line)-1]
		}

		if len(line) == 0 {
			if len(lines) > 0 {
				outBuf, cues, cueLens = parseBlock(lines, outBuf, cues, cueLens)
				lines = lineBuf[:0]
			}
			continue
		}

		lines = append(lines, line)
	}

	if len(lines) > 0 {
		outBuf, cues, cueLens = parseBlock(lines, outBuf, cues, cueLens)
	}

	said := string(outBuf)
	for i := range cues {
		cues[i].Text = said[cues[i].Offset : cues[i].Offset+cueLens[i]]
	}
	return said, cues
}

// parseBlock is one block of the file as a cue, and whether it is one. The
// header, a note and anything a later version of the format adds are not.
func parseBlock(lines [][]byte, outBuf []byte, cues []Cue, cueLens []int) ([]byte, []Cue, []int) {
	at, from, to, ok := findTiming(lines)
	if !ok {
		return outBuf, cues, cueLens
	}
	var offset, length int
	outBuf, offset, length, ok = appendCueText(outBuf, lines[at+1:], len(cues) == 0)
	if !ok {
		return outBuf, cues, cueLens
	}
	cues = append(cues, Cue{From: from, To: to, Offset: offset})
	cueLens = append(cueLens, length)
	return outBuf, cues, cueLens
}

// findTiming is where the arrow stands in a block, and the timings it connects.
func findTiming(lines [][]byte) (at int, from, to int, ok bool) {
	for i, line := range lines {
		before, after, found := bytes.Cut(line, []byte(arrow))
		if !found {
			// A cue may be named on the line above its timing. Anything else
			// standing there is not a cue.
			if i > 0 || bytes.HasPrefix(line, []byte(Head)) || bytes.HasPrefix(line, []byte("NOTE")) {
				return 0, 0, 0, false
			}
			continue
		}
		from, ok = parseStamp(before)
		if !ok {
			return 0, 0, 0, false
		}
		to, ok = parseSecondStamp(after)
		if !ok {
			return 0, 0, 0, false
		}
		return i, from, to, true
	}
	return 0, 0, 0, false
}

// parseSecondStamp is the second timing of a cue, which cue settings may
// follow.
func parseSecondStamp(after []byte) (int, bool) {
	after = bytes.TrimLeft(after, " \t\r\n")
	if len(after) == 0 {
		return 0, false
	}
	second := after
	if end := indexSpace(after); end >= 0 {
		second = after[:end]
	}
	return parseStamp(second)
}

// appendCueText writes the words of a cue into outBuf.
func appendCueText(outBuf []byte, lines [][]byte, isFirstCue bool) (buf []byte, offset, length int, ok bool) {
	if len(lines) == 0 {
		return outBuf, 0, 0, false
	}
	cueStart := len(outBuf)
	if !isFirstCue {
		outBuf = append(outBuf, '\n')
	}
	textOffset := len(outBuf)

	if len(lines) == 1 {
		trimmed := bytes.TrimSpace(lines[0])
		if len(trimmed) == 0 {
			return outBuf[:cueStart], 0, 0, false
		}
		outBuf = append(outBuf, trimmed...)
		return outBuf, textOffset, len(trimmed), true
	}

	for i, line := range lines {
		if i > 0 {
			outBuf = append(outBuf, '\n')
		}
		outBuf = append(outBuf, line...)
	}
	textBytes := outBuf[textOffset:]
	trimmed := bytes.TrimSpace(textBytes)
	if len(trimmed) == 0 {
		return outBuf[:cueStart], 0, 0, false
	}
	if &trimmed[0] != &textBytes[0] {
		copy(outBuf[textOffset:], trimmed)
	}
	outBuf = outBuf[:textOffset+len(trimmed)]
	return outBuf, textOffset, len(trimmed), true
}

// indexSpace is the index of the first ASCII whitespace byte in b.
func indexSpace(b []byte) int {
	for i, c := range b {
		if c == ' ' || c == '\t' || c == '\r' || c == '\n' {
			return i
		}
	}
	return -1
}

// Stamp is a millisecond as the format writes it: hours, minutes, seconds and
// thousandths.
func Stamp(ms int) string {
	var buf [16]byte
	b := appendStamp(buf[:0], ms)
	return string(b)
}

// appendStamp formats a millisecond as hours, minutes, seconds and
// thousandths.
func appendStamp(dst []byte, ms int) []byte {
	if ms < 0 {
		ms = 0
	}
	h := ms / 3600000
	m := (ms / 60000) % 60
	s := (ms / 1000) % 60
	f := ms % 1000

	if h < 100 {
		dst = append(dst, byte('0'+h/10), byte('0'+h%10))
	} else {
		dst = strconv.AppendInt(dst, int64(h), 10)
	}
	dst = append(dst, ':', byte('0'+m/10), byte('0'+m%10), ':', byte('0'+s/10), byte('0'+s%10), '.')
	dst = append(dst, byte('0'+f/100), byte('0'+(f/10)%10), byte('0'+f%10))
	return dst
}

// Clock is a moment of a recording as a person reads one: the way a player
// writes where it stands. An hour that is not there is not written.
func Clock(ms int) string {
	whole := max(ms, 0) / 1000
	m := whole / 60 % 60
	s := whole % 60
	var buf [16]byte
	b := buf[:0]
	if hours := whole / 3600; hours > 0 {
		b = strconv.AppendInt(b, int64(hours), 10)
		b = append(b, ':', byte('0'+m/10), byte('0'+m%10), ':', byte('0'+s/10), byte('0'+s%10))
		return string(b)
	}
	b = strconv.AppendInt(b, int64(whole/60), 10)
	b = append(b, ':', byte('0'+s/10), byte('0'+s%10))
	return string(b)
}

// parseStamp is a timing the format writes. The hours are optional, which is
// what the format says and what other tools write.
func parseStamp(raw []byte) (int, bool) {
	raw = bytes.TrimSpace(raw)
	c1 := bytes.IndexByte(raw, ':')
	if c1 < 0 {
		return 0, false
	}
	var p0, p1, last []byte
	c2 := bytes.IndexByte(raw[c1+1:], ':')
	if c2 < 0 {
		p0 = raw[:c1]
		last = raw[c1+1:]
	} else {
		c2 = c1 + 1 + c2
		if bytes.IndexByte(raw[c2+1:], ':') >= 0 {
			return 0, false
		}
		p0 = raw[:c1]
		p1 = raw[c1+1 : c2]
		last = raw[c2+1:]
	}

	ms := 0
	n0, ok := parseUint(p0)
	if !ok {
		return 0, false
	}
	ms = n0
	if p1 != nil {
		n1, ok := parseUint(p1)
		if !ok {
			return 0, false
		}
		ms = ms*60 + n1
	}
	ms *= 60000

	dot := bytes.IndexByte(last, '.')
	if dot < 0 {
		return 0, false
	}
	seconds := last[:dot]
	thousandths := last[dot+1:]
	if len(thousandths) != 3 {
		return 0, false
	}
	s, ok := parseUint(seconds)
	if !ok {
		return 0, false
	}
	t, ok := parseUint(thousandths)
	if !ok {
		return 0, false
	}
	return ms + s*1000 + t, true
}

// parseUint parses a non-negative decimal integer from b.
func parseUint(b []byte) (int, bool) {
	if len(b) == 0 {
		return 0, false
	}
	n := 0
	for _, c := range b {
		if c < '0' || c > '9' {
			return 0, false
		}
		n = n*10 + int(c-'0')
	}
	return n, true
}

// GetCuesAt is where a run of the words sits: the cues it falls in, in the
// order they were spoken. A run crossing a silence is in both of them.
func GetCuesAt(cues []Cue, start, length int) []Cue {
	if length <= 0 || len(cues) == 0 {
		return nil
	}
	end := start + length

	// The first cue that reaches into the run. A cue before it ends before the
	// run begins.
	at := sort.Search(len(cues), func(i int) bool {
		return cues[i].Offset+len(cues[i].Text) > start
	})

	var out []Cue
	for ; at < len(cues) && cues[at].Offset < end; at++ {
		out = append(out, cues[at])
	}
	return out
}

// reaches is the note a run stopped part way leaves, as it stands in the file.
// The words after it are how many milliseconds of the recording have been
// written down.
const reaches = "NOTE heard "

// GetReachMarker is that note. It stands after the cues it claims, so a batch
// that did not land whole is one no note claims.
func GetReachMarker(ms int) []byte {
	var buf [32]byte
	b := append(buf[:0], '\n')
	b = append(b, reaches...)
	b = strconv.AppendInt(b, int64(ms), 10)
	b = append(b, '\n')
	res := make([]byte, len(b))
	copy(res, b)
	return res
}

// ByHand is the note a transcript a person wrote carries.
const ByHand = "NOTE by hand"

// Hand marks a transcript as the words a person put there. A transcript
// carrying it is left as they left it.
func Hand() []byte {
	return []byte("\n" + ByHand + "\n")
}

// IsWrittenByHand says whether a person wrote these words. The mark is a note
// of its own, and the same words spoken in a cue are speech.
func IsWrittenByHand(raw []byte) bool {
	note := []byte(ByHand)
	for at := 0; at <= len(raw)-len(note); {
		found := bytes.Index(raw[at:], note)
		if found < 0 {
			return false
		}
		if found += at; isBlockStart(raw, found) {
			return true
		}
		at = found + 1
	}
	return false
}

// ReadReached is how far a run before this one got, and where the last note
// about it ends. A file carrying none is a recording nothing has transcribed.
func ReadReached(raw []byte) (ms, end int) {
	note := []byte(reaches)
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
		if stop < 0 || !isBlockStart(raw, at) {
			continue
		}
		ms, ok := parseUint(bytes.TrimSpace(line[:stop]))
		if !ok {
			continue
		}
		return ms, at + len(note) + stop + 1
	}
}

// isBlockStart says whether a byte is where a block of the file starts: the top
// of it, or the line after a blank one. A note stands at the top of its own
// block.
func isBlockStart(raw []byte, at int) bool {
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
