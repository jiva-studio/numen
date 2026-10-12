package text

import "unicode/utf8"

// WholeCharacters is the range with the character it opened inside and the one it
// closed inside left off.
//
// A range reaching the end of the file closes inside nothing: an incomplete
// character there is what the file holds.
func WholeCharacters(start int, raw []byte, whole int) (int, []byte) {
	for start > 0 && len(raw) > 0 && !utf8.RuneStart(raw[0]) {
		raw = raw[1:]
		start++
	}
	if start+len(raw) >= whole {
		return start, raw
	}
	for at := len(raw) - 1; at >= 0 && at > len(raw)-utf8.UTFMax; at-- {
		if !utf8.RuneStart(raw[at]) {
			continue
		}
		if !utf8.FullRune(raw[at:]) {
			raw = raw[:at]
		}
		break
	}
	return start, raw
}
