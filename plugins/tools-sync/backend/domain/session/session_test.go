package session_test

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/jiva-studio/numen/plugins/tools-sync/backend/domain/session"
)

func TestASessionNeverPrintsItsToken(t *testing.T) {
	s := session.Session{Token: "secret-bearer-token", ExpiresAt: time.Unix(1_795_000_000, 0), AccountFingerprint: "3F2A 91C0"}

	for name, text := range map[string]string{
		"%v": fmt.Sprintf("%v", s), "%+v": fmt.Sprintf("%+v", s), "%s": fmt.Sprint(s), "pointer": fmt.Sprintf("%v", &s),
	} {
		if strings.Contains(text, "secret-bearer-token") {
			t.Errorf("%s prints the token: %q", name, text)
		}
		if text != "[session]" {
			t.Errorf("%s = %q, want [session]", name, text)
		}
	}
}
