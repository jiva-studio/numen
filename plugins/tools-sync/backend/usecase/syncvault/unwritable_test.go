package syncvault_test

import (
	"errors"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/health"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/domain/healthkey"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/testsupport/synctest"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/usecase/pull"
	"github.com/jiva-studio/numen/tools/syncserver/fakeserver"
)

const lockedNote = "Locked/Plan.md"

var errDenied = errors.New("permission denied")

func findUnsavedItem(c *synctest.Computer) (health.Item, bool) {
	at := slices.IndexFunc(c.Health.List(), func(item health.Item) bool { return item.Key == healthkey.Create("unsaved-files", string(target)) })
	if at < 0 {
		return health.Item{}, false
	}
	return c.Health.List()[at], true
}

func TestSyncVault_AFileTheHostCannotWriteDoesNotStopTheRestOfThePull(t *testing.T) {
	p := newPair(t, fakeserver.Config{})
	p.a.WriteNote(source, lockedNote, "locked")
	p.a.WriteNote(source, "Other.md", "other")
	sync(t, p.a, source)
	p.b.Vaults.With(target).RefuseWriting(lockedNote, errDenied)

	sync(t, p.b, target)

	wantNote(t, p.b, target, "Other.md", "other")
	wantNoFile(t, p.b, target, lockedNote)
}

func TestSyncVault_AFileTheHostCannotWriteDoesNotStopThePushOfLocalChanges(t *testing.T) {
	p := newPair(t, fakeserver.Config{})
	p.a.WriteNote(source, lockedNote, "locked")
	sync(t, p.a, source)
	p.b.Vaults.With(target).RefuseWriting(lockedNote, errDenied)
	p.b.WriteNote(target, "Mine.md", "mine")

	sync(t, p.b, target)
	sync(t, p.a, source)

	wantNote(t, p.a, source, "Mine.md", "mine")
}

func TestSyncVault_AFileTheHostCannotWriteIsNamedInAHealthItem(t *testing.T) {
	p := newPair(t, fakeserver.Config{})
	p.a.WriteNote(source, lockedNote, "locked")
	sync(t, p.a, source)
	p.b.Vaults.With(target).RefuseWriting(lockedNote, errDenied)

	sync(t, p.b, target)

	item, isRaised := findUnsavedItem(p.b)
	if !isRaised || item.Text != pull.GetUnsavedText(1) || !strings.Contains(item.Detail, lockedNote) {
		t.Errorf("health item = %+v (raised: %v), want %q naming %s", item, isRaised, pull.GetUnsavedText(1), lockedNote)
	}
}

func TestSyncVault_AFileTheHostCannotWriteIsTriedAgainLaterAndTheItemGoes(t *testing.T) {
	p := newPair(t, fakeserver.Config{})
	p.a.WriteNote(source, lockedNote, "locked")
	sync(t, p.a, source)
	p.b.Vaults.With(target).RefuseWriting(lockedNote, errDenied)
	sync(t, p.b, target)
	p.b.Vaults.With(target).RefuseWriting(lockedNote, nil)

	sync(t, p.b, target)
	wantNoFile(t, p.b, target, lockedNote)

	p.b.Clock.Advance(time.Hour)
	sync(t, p.b, target)

	wantNote(t, p.b, target, lockedNote, "locked")
	if item, isRaised := findUnsavedItem(p.b); isRaised {
		t.Errorf("health item %+v still raised after the file was saved", item)
	}
}
