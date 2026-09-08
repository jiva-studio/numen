package urlfile_test

import (
	"errors"
	"testing"

	"github.com/jiva-studio/numen/modules/libs/core/domain"
	"github.com/jiva-studio/numen/modules/libs/core/urlfile"
)

// A `.url` file is what every system writes, and what one wrote is read back.
func TestAnAddressIsReadBackFromWhatWasWritten(t *testing.T) {
	at, err := domain.ParseURL("https://www.youtube.com/watch?v=dQw4w9WgXcQ")
	if err != nil {
		t.Fatal(err)
	}
	read, err := urlfile.Read(urlfile.Write(at))
	if err != nil {
		t.Fatal(err)
	}
	if string(read) != string(at) {
		t.Errorf("read %q, wrote %q", string(read), string(at))
	}
}

// What another program wrote is read too: the section carries other keys, the
// key is spelled how that program spells it, and the lines end as Windows ends
// them.
func TestWhatAnotherProgramWroteIsRead(t *testing.T) {
	raw := "[InternetShortcut]\r\nIDList=\r\nurl=https://example.com/a\r\nIconIndex=0\r\n"
	at, err := urlfile.Read([]byte(raw))
	if err != nil {
		t.Fatal(err)
	}
	if string(at) != "https://example.com/a" {
		t.Errorf("read %q", string(at))
	}
}

// A file naming no address is one the vault says so about, and not one that
// silently points nowhere.
func TestAFileWithNoAddressSaysSo(t *testing.T) {
	if _, err := urlfile.Read([]byte("[InternetShortcut]\nIconIndex=0\n")); !errors.Is(err, urlfile.ErrNoAddress) {
		t.Errorf("a file with no URL gave %v", err)
	}
}
