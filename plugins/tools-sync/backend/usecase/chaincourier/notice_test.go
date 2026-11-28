package chaincourier

import (
	"crypto/rand"
	"strings"
	"testing"
	"time"
	"unicode/utf8"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/health"
	"github.com/jiva-studio/numen/libs/shared/identity"
)

func TestNotices_FitTheTextLimitOfAHealthItemWithTheLongestDate(t *testing.T) {
	t.Parallel()
	longest := time.Date(2026, time.September, 30, 0, 0, 0, 0, time.UTC)
	for name, text := range map[string]string{"revoked": getRevokedNotice(longest), "added": getAddedNotice(longest), "forked": getForkNotice(longest)} {
		if got := utf8.RuneCountInString(text); got > health.MaxTextLength {
			t.Errorf("the %s notice has %d characters, want at most %d: %q", name, got, health.MaxTextLength, text)
		}
	}
}

type notifyChain struct {
	chain    identity.Chain
	recovery identity.RecoveryKey
	devices  map[string]identity.DeviceKey
	at       time.Time
}

func newNotifyChain(t *testing.T, names ...string) notifyChain {
	t.Helper()
	words, err := identity.NewRecoveryWords(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	recovery, err := words.DeriveRecoveryKey()
	if err != nil {
		t.Fatal(err)
	}
	keys, err := identity.NewAccountKeys(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	at := time.Date(2026, time.November, 24, 12, 0, 0, 0, time.UTC)
	chain, err := identity.NewChain(recovery, keys, at)
	if err != nil {
		t.Fatal(err)
	}
	devices := map[string]identity.DeviceKey{}
	for _, name := range names {
		device, err := identity.NewDeviceKey(rand.Reader)
		if err != nil {
			t.Fatal(err)
		}
		devices[name] = device
	}
	return notifyChain{chain: chain, recovery: recovery, devices: devices, at: at}
}

func (c *notifyChain) add(t *testing.T, by identity.Signer, name string) {
	t.Helper()
	next, err := c.chain.AppendDevice(by, c.devices[name].Public(), name, c.at)
	if err != nil {
		t.Fatal(err)
	}
	c.chain = next
}

func (c *notifyChain) getSign(name string) []byte { return c.devices[name].Public().Sign }

func TestListNotices_RaisesNothingInATwoDeviceChain(t *testing.T) {
	t.Parallel()
	c := newNotifyChain(t, "A", "B")
	c.add(t, c.recovery, "A")
	c.add(t, c.devices["A"], "B")

	for _, own := range []string{"A", "B"} {
		if got := listNotices(c.chain, c.getSign(own), c.at.Add(time.Hour), signedTimes(c.chain)); len(got) != 0 {
			t.Errorf("device %s: notices = %+v, want none", own, got)
		}
	}
}

func TestListNotices_RaisesTheAddedNoticeOnlyOnTheDeviceThatNeitherAddedNorWasAdded(t *testing.T) {
	t.Parallel()
	c := newNotifyChain(t, "A", "B", "C")
	c.add(t, c.recovery, "A")
	c.add(t, c.recovery, "B")
	c.add(t, c.devices["B"], "C")
	now := c.at.Add(time.Hour)

	if got := listNotices(c.chain, c.getSign("A"), now, signedTimes(c.chain)); len(got) != 1 || !strings.Contains(got[0], "added another device") {
		t.Errorf("device A: notices = %+v, want the added notice", got)
	}
	for _, own := range []string{"B", "C"} {
		if got := listNotices(c.chain, c.getSign(own), now, signedTimes(c.chain)); len(got) != 0 {
			t.Errorf("device %s: notices = %+v, want none", own, got)
		}
	}
}

func signedTimes(chain identity.Chain) map[string]int64 {
	seen := map[string]int64{}
	for id, at := range listEntries(chain) {
		seen[id] = at.Unix()
	}
	return seen
}
