package chaincourier

import (
	"testing"
	"time"
	"unicode/utf8"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/health"
)

func TestNotices_FitTheTextLimitOfAHealthItemWithTheLongestDate(t *testing.T) {
	t.Parallel()
	longest := time.Date(2026, time.September, 30, 0, 0, 0, 0, time.UTC)
	for name, text := range map[string]string{"revoked": getRevokedNotice(longest), "added": getAddedNotice(longest)} {
		if got := utf8.RuneCountInString(text); got > health.MaxTextLength {
			t.Errorf("the %s notice has %d characters, want at most %d: %q", name, got, health.MaxTextLength, text)
		}
	}
}
