package domain_test

import (
	"errors"
	"testing"

	"github.com/jiva-studio/numen/modules/libs/core/domain"
)

// A `.url` file is what every system writes, and what one wrote is read back.
func TestAnAddressIsReadBackFromWhatWasWritten(t *testing.T) {
	at, err := domain.ParseWebAddress("https://www.youtube.com/watch?v=dQw4w9WgXcQ")
	if err != nil {
		t.Fatal(err)
	}
	read, err := domain.ReadURL(domain.WriteURL(at))
	if err != nil {
		t.Fatal(err)
	}
	if read.URL != at.URL {
		t.Errorf("read %q, wrote %q", read.URL, at.URL)
	}
}

// What another program wrote is read too: the section carries other keys, the
// key is spelled how that program spells it, and the lines end as Windows ends
// them.
func TestWhatAnotherProgramWroteIsRead(t *testing.T) {
	raw := "[InternetShortcut]\r\nIDList=\r\nurl=https://example.com/a\r\nIconIndex=0\r\n"
	at, err := domain.ReadURL([]byte(raw))
	if err != nil {
		t.Fatal(err)
	}
	if at.URL != "https://example.com/a" {
		t.Errorf("read %q", at.URL)
	}
}

// A file naming no address is one the vault says so about, and not one that
// silently points nowhere.
func TestAFileWithNoAddressSaysSo(t *testing.T) {
	if _, err := domain.ReadURL([]byte("[InternetShortcut]\nIconIndex=0\n")); !errors.Is(err, domain.ErrNoAddress) {
		t.Errorf("a file with no URL gave %v", err)
	}
}
