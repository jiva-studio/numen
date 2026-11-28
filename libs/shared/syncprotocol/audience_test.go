package syncprotocol_test

import (
	"errors"
	"testing"
	"time"

	"github.com/jiva-studio/numen/libs/shared/syncprotocol"
	syncv1 "github.com/jiva-studio/numen/libs/shared/syncprotocol/gen/numen/sync/v1"
)

func TestIsSameAddress_ComparesSchemeHostAndPortAfterNormalizing(t *testing.T) {
	t.Parallel()
	for _, one := range []struct {
		a, b string
		same bool
	}{
		{"https://sync.numen.md", "https://sync.numen.md", true},
		{"https://Sync.Numen.MD", "https://sync.numen.md", true},
		{"https://sync.numen.md:443", "https://sync.numen.md", true},
		{"https://sync.numen.md/", "https://sync.numen.md", true},
		{"http://127.0.0.1:8080", "http://127.0.0.1:8080/", true},
		{"http://sync.numen.md", "https://sync.numen.md", false},
		{"https://sync.numen.md:8443", "https://sync.numen.md", false},
		{"https://sync.numen.md.evil.example", "https://sync.numen.md", false},
		{"https://evil.example", "https://sync.numen.md", false},
		{"http://127.0.0.1:8080", "http://localhost:8080", false},
		{"http://127.0.0.1:8080", "http://127.0.0.1:8081", false},
	} {
		if got := syncprotocol.IsSameAddress(one.a, one.b); got != one.same {
			t.Errorf("IsSameAddress(%q, %q) = %v, want %v", one.a, one.b, got, one.same)
		}
	}
}

func TestIsSameAddress_NeverMatchesAnAddressThatIsNotOne(t *testing.T) {
	t.Parallel()
	for _, bad := range []string{"", "sync.numen.md", "ftp://sync.numen.md", "https://", "https://user@sync.numen.md", "://x"} {
		if syncprotocol.IsSameAddress(bad, bad) {
			t.Errorf("IsSameAddress(%q, %q) = true, want false", bad, bad)
		}
		if _, err := syncprotocol.NormalizeAddress(bad); err == nil {
			t.Errorf("NormalizeAddress(%q) accepted it", bad)
		}
	}
}

func TestNormalizeAddress_ReturnsSchemeHostAndPortInLowerCaseWithoutTheDefaultPort(t *testing.T) {
	t.Parallel()
	for in, want := range map[string]string{
		"https://Sync.Numen.MD/":     "https://sync.numen.md",
		"https://sync.numen.md:443":  "https://sync.numen.md",
		"http://127.0.0.1:8080/":     "http://127.0.0.1:8080",
		"http://sync.numen.md:80":    "http://sync.numen.md",
		"https://sync.numen.md:8443": "https://sync.numen.md:8443",
	} {
		got, err := syncprotocol.NormalizeAddress(in)
		if err != nil || got != want {
			t.Errorf("NormalizeAddress(%q) = %q, %v, want %q", in, got, err, want)
		}
	}
}

func clockNow() time.Time {
	return time.Date(2026, time.November, 22, 12, 0, 0, 0, time.UTC)
}

func challengeExpiringIn(audience string, in time.Duration) *syncv1.GetChallengeResponse {
	challenge := newChallenge()
	challenge.Audience = audience
	challenge.ExpiresAt = clockNow().Add(in).Unix()
	return challenge
}

func TestCheckChallenge_AcceptsTheAddressItDialedAndAnExpiryNearItsClock(t *testing.T) {
	t.Parallel()
	for _, in := range []time.Duration{60 * time.Second, 0, -120 * time.Second, 120 * time.Second} {
		if err := syncprotocol.CheckChallenge(challengeExpiringIn("https://sync.numen.md", in), "https://sync.numen.md:443", clockNow()); err != nil {
			t.Errorf("expiry %v from now: %v", in, err)
		}
	}
}

func TestCheckChallenge_RefusesAnAudienceThatIsNotTheAddressItDialed(t *testing.T) {
	t.Parallel()
	for _, audience := range []string{"https://evil.example", "http://sync.numen.md", "https://sync.numen.md:8443", ""} {
		err := syncprotocol.CheckChallenge(challengeExpiringIn(audience, 60*time.Second), "https://sync.numen.md", clockNow())
		if !errors.Is(err, syncprotocol.ErrAudienceMismatch) {
			t.Errorf("audience %q: error = %v, want ErrAudienceMismatch", audience, err)
		}
	}
}

func TestCheckChallenge_RefusesAnExpiryMoreThan120SecondsFromItsClock(t *testing.T) {
	t.Parallel()
	for _, in := range []time.Duration{121 * time.Second, -121 * time.Second, 24 * time.Hour, -24 * time.Hour} {
		err := syncprotocol.CheckChallenge(challengeExpiringIn("https://sync.numen.md", in), "https://sync.numen.md", clockNow())
		if !errors.Is(err, syncprotocol.ErrChallengeOutOfRange) {
			t.Errorf("expiry %v from now: error = %v, want ErrChallengeOutOfRange", in, err)
		}
	}
}

func TestCheckChallenge_RefusesANonceOrIDOfTheWrongLength(t *testing.T) {
	t.Parallel()
	short := challengeExpiringIn("https://sync.numen.md", 60*time.Second)
	short.Nonce = short.GetNonce()[:31]
	noID := challengeExpiringIn("https://sync.numen.md", 60*time.Second)
	noID.ChallengeId = nil

	for name, challenge := range map[string]*syncv1.GetChallengeResponse{"short nonce": short, "no id": noID} {
		if err := syncprotocol.CheckChallenge(challenge, "https://sync.numen.md", clockNow()); !errors.Is(err, syncprotocol.ErrMalformedChallenge) {
			t.Errorf("%s: error = %v, want ErrMalformedChallenge", name, err)
		}
	}
}
