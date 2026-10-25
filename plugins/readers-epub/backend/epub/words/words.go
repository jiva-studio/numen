package words

import "strings"

// NormaliseSpaces is one line of text: the words of a name, single-spaced.
func NormaliseSpaces(s string) string {
	return strings.Join(strings.Fields(s), " ")
}

// HasToken reports whether a space-separated attribute carries a value.
func HasToken(attribute, token string) bool {
	for _, field := range strings.Fields(attribute) {
		if strings.EqualFold(field, token) {
			return true
		}
	}
	return false
}

// HasTokenBytes reports whether a space-separated attribute carries a value.
func HasTokenBytes(attribute []byte, token string) bool {
	for len(attribute) > 0 {
		for len(attribute) > 0 && (attribute[0] == ' ' || attribute[0] == '\t' || attribute[0] == '\n' || attribute[0] == '\r') {
			attribute = attribute[1:]
		}
		end := 0
		for end < len(attribute) && attribute[end] != ' ' && attribute[end] != '\t' && attribute[end] != '\n' && attribute[end] != '\r' {
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
