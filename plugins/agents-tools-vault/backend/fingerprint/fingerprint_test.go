package fingerprint_test

import (
	"strings"
	"testing"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/domain"
	"github.com/jiva-studio/numen/plugins/agents-tools-vault/backend/fingerprint"
)

func TestParseReadsWhatFormatWrote(t *testing.T) {
	stat := domain.FileStat{Size: 482, Hash: domain.HashContent([]byte("body"))}

	got, err := fingerprint.Parse(fingerprint.Format(stat))

	if err != nil || got.Hash != stat.Hash || got.IsZero() {
		t.Errorf("got %+v (%v), want the hash %s", got, err, stat.Hash)
	}
}

func TestParseRejectsATokenThatIsNotAHash(t *testing.T) {
	for token, wantMsg := range map[string]string{
		"":                      "fingerprint is required",
		"482-1700000000":        `invalid fingerprint format: "482-1700000000"`,
		strings.Repeat("A", 64): `invalid fingerprint format: "` + strings.Repeat("A", 64) + `"`,
		strings.Repeat("a", 63): `invalid fingerprint format: "` + strings.Repeat("a", 63) + `"`,
	} {
		_, err := fingerprint.Parse(token)
		if err == nil || err.Error() != wantMsg {
			t.Errorf("Parse(%q) error = %v, want %q", token, err, wantMsg)
		}
	}
}
