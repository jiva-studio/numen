package domain

import (
	"errors"
	"strings"
)

// A `.url` file is an INI with one section and one key that matters.
const (
	urlSection = "[InternetShortcut]"
	urlKey     = "URL"
)

// ErrNoAddress is a `.url` file with no address in it.
var ErrNoAddress = errors.New("this file names no address")

// ReadURL is the address a `.url` file points at.
//
// The section is not required and the key is matched without regard to case:
// what other programs write varies, and every one of them writes `URL=`.
func ReadURL(raw []byte) (WebAddress, error) {
	for line := range strings.Lines(string(raw)) {
		key, value, split := strings.Cut(strings.TrimSpace(line), "=")
		if !split || !strings.EqualFold(strings.TrimSpace(key), urlKey) {
			continue
		}
		return ParseWebAddress(strings.TrimSpace(value))
	}
	return WebAddress{}, ErrNoAddress
}

// WriteURL is a `.url` file holding one address, in the form every system reads.
func WriteURL(at WebAddress) []byte {
	return []byte(urlSection + "\n" + urlKey + "=" + at.URL + "\n")
}
