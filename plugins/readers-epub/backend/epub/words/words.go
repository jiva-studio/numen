// Package words normalises and matches the words of markup attributes and names.
package words

import "strings"

// NormaliseSpaces is one line of text: the words of a name, single-spaced.
func NormaliseSpaces(s string) string {
	return strings.Join(strings.Fields(s), " ")
}

// HasToken reports whether a space-separated attribute carries a value.
func HasToken(attribute, token string) bool {
	for field := range strings.FieldsSeq(attribute) {
		if strings.EqualFold(field, token) {
			return true
		}
	}
	return false
}

// HasTokenBytes reports whether a space-separated attribute carries a value.
func HasTokenBytes(attribute []byte, token string) bool {
	for len(attribute) > 0 {
		for len(attribute) > 0 && isSpaceByte(attribute[0]) {
			attribute = attribute[1:]
		}
		end := 0
		for end < len(attribute) && !isSpaceByte(attribute[end]) {
			end++
		}
		field := attribute[:end]
		attribute = attribute[end:]
		if strings.EqualFold(string(field), token) {
			return true
		}
	}
	return false
}

func isSpaceByte(b byte) bool {
	return b == ' ' || b == '\t' || b == '\n' || b == '\r'
}
