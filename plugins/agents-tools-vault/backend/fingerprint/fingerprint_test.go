package fingerprint_test

import (
	"testing"
	"time"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/domain"
	"github.com/jiva-studio/numen/plugins/agents-tools-vault/backend/fingerprint"
)

func TestParseReadsWhatFormatWrote(t *testing.T) {
	stat := domain.FileStat{Path: "", Kind: "", Size: 482, ModTime: time.Unix(1700000000, 123)}
	got, err := fingerprint.Parse(fingerprint.Format(stat))
	if err != nil {
		t.Fatal(err)
	}
	if got.Size != 482 || !got.ModTime.Equal(stat.ModTime) {
		t.Errorf("got size %d and time %v, want 482 and %v", got.Size, got.ModTime, stat.ModTime)
	}
}

func TestParseRejectsATokenThatIsNotASizeAndATime(t *testing.T) {
	for _, token := range []string{"", "482", "big-1", "482-late"} {
		if _, err := fingerprint.Parse(token); err == nil {
			t.Errorf("Parse(%q) returned no error", token)
		}
	}
}
