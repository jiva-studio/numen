package share_test

import (
	"crypto/rand"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/jiva-studio/numen/libs/shared/identity"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/domain/share"
)

func newInvitation(t *testing.T, server string) share.LinkInvitation {
	t.Helper()
	_, payload, err := identity.NewLinkDisplayer(rand.Reader, [16]byte{1, 2, 3}, server)
	if err != nil {
		t.Fatal(err)
	}
	return share.LinkInvitation{Payload: payload, Folder: "Food", Fingerprint: "0F3A 9C21", ExpiresAt: time.Unix(1_800_000_000, 0).UTC()}
}

func TestParseLink_ReadsWhatNewLinkWrote(t *testing.T) {
	want := newInvitation(t, "https://sync.example.org")

	text, err := share.NewLink(want)
	if err != nil {
		t.Fatal(err)
	}
	got, err := share.ParseLink(text)

	if err != nil || got != want {
		t.Errorf("link = %+v, %v, want %+v", got, err, want)
	}
}

func TestParseLink_RefusesWhatIsNotAnIntactLinkWithAServer(t *testing.T) {
	text, err := share.NewLink(newInvitation(t, "https://sync.example.org"))
	if err != nil {
		t.Fatal(err)
	}
	noServer, err := share.NewLink(newInvitation(t, ""))
	if err != nil {
		t.Fatal(err)
	}
	flip := "A"
	if text[20] == 'A' {
		flip = "B"
	}
	for name, bad := range map[string]string{
		"empty": "", "another scheme": "numen-link:" + strings.TrimPrefix(text, "numen-share:"), "cut": text[:len(text)-3],
		"changed": text[:20] + flip + text[21:], "no server": noServer, "not base64": "numen-share:!!!",
	} {
		if _, err := share.ParseLink(bad); !errors.Is(err, share.ErrBadLink) {
			t.Errorf("%s: error = %v, want ErrBadLink", name, err)
		}
	}
}

func TestCleanDeviceName_ShortensAndNeverEmpty(t *testing.T) {
	if got := share.CleanDeviceName("  \n"); got != "Unknown device" {
		t.Errorf("name = %q, want Unknown device", got)
	}
	if got := share.CleanDeviceName(strings.Repeat("a", 500)); len(got) > 120 {
		t.Errorf("name has %d bytes, want at most 120", len(got))
	}
}
