// Package urlfile reads and writes the file a web address is kept in.
//
// It is an INI with one section and one key that matters, which is what every
// system writes an internet shortcut as. Nothing here knows what is at an
// address; it is the format and nothing else.
package urlfile

import (
	"errors"
	"strings"

	"github.com/jiva-studio/numen/modules/libs/core/domain"
)

const (
	section = "[InternetShortcut]"
	key     = "URL"
)

// ErrNoAddress is a file with no address in it.
var ErrNoAddress = errors.New("this file names no address")

// Read is the address a file points at.
//
// The first `URL=` line the file holds is the address. The section header is
// not required and the key is matched without regard to case.
func Read(raw []byte) (domain.URL, error) {
	for line := range strings.Lines(string(raw)) {
		name, value, split := strings.Cut(strings.TrimSpace(line), "=")
		if !split || !strings.EqualFold(strings.TrimSpace(name), key) {
			continue
		}
		return domain.ParseURL(strings.TrimSpace(value))
	}
	return "", ErrNoAddress
}

// Write is a file holding one address, in the form every system reads.
func Write(at domain.URL) []byte {
	return []byte(section + "\n" + key + "=" + string(at) + "\n")
}
