package settingevents_test

import (
	"errors"
	"path/filepath"
	"slices"
	"testing"
	"time"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/settings"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/settingevents"
)

func newSource(t *testing.T, read func() (settings.Values, error)) settingevents.Source {
	t.Helper()
	return settingevents.Source{
		Read:    read,
		Initial: func() settings.Values { return settings.Values{} },
		Report:  func(error) {},
		Path:    filepath.Join(t.TempDir(), "numen.json"),
	}
}

func receive(t *testing.T, changes <-chan []settings.Setting) []settings.Setting {
	t.Helper()
	select {
	case batch := <-changes:
		return batch
	case <-time.After(2 * time.Second):
		t.Fatal("no batch arrived")
		return nil
	}
}

func TestRefreshDeliversOnlyTheSettingsWhoseValueDiffers(t *testing.T) {
	values := settings.Values{"a\x00x": "1", "a\x00y": "2"}
	events := settingevents.New()
	changes, stop := events.Subscribe(newSource(t, func() (settings.Values, error) { return values, nil }))
	defer stop()

	next := settings.Values{"a\x00x": "1", "a\x00y": "3"}
	events.Refresh(func() (settings.Values, error) { return next, nil })

	batch := receive(t, changes)
	if len(batch) != 1 || !slices.Equal(batch[0].Path, []string{"a", "y"}) || batch[0].JSON != "3" {
		t.Errorf("the batch is %v", batch)
	}
}

func TestRefreshAfterAReadFailureChangesNothing(t *testing.T) {
	var reported []error
	source := newSource(t, func() (settings.Values, error) { return settings.Values{"a\x00x": "1"}, nil })
	source.Report = func(err error) { reported = append(reported, err) }
	events := settingevents.New()
	changes, stop := events.Subscribe(source)
	defer stop()

	events.Refresh(func() (settings.Values, error) { return nil, errors.New("unreadable") })

	select {
	case batch := <-changes:
		t.Errorf("a batch arrived: %v", batch)
	case <-time.After(100 * time.Millisecond):
	}
	if len(reported) == 0 {
		t.Error("the failure was not reported")
	}
}

func TestAReaderThatLeftHasItsChannelClosed(t *testing.T) {
	events := settingevents.New()
	changes, stop := events.Subscribe(newSource(t, func() (settings.Values, error) { return settings.Values{}, nil }))

	stop()
	stop()

	select {
	case _, open := <-changes:
		if open {
			t.Error("the channel delivered after the reader left")
		}
	case <-time.After(time.Second):
		t.Error("the channel stayed open")
	}
}

func TestASettingWithNoValueIsReportedAsNull(t *testing.T) {
	events := settingevents.New()
	changes, stop := events.Subscribe(newSource(t, func() (settings.Values, error) {
		return settings.Values{"a\x00x": "1"}, nil
	}))
	defer stop()

	events.Refresh(func() (settings.Values, error) { return settings.Values{}, nil })

	batch := receive(t, changes)
	if len(batch) != 1 || !slices.Equal(batch[0].Path, []string{"a", "x"}) || batch[0].JSON != "null" {
		t.Errorf("the batch is %v", batch)
	}
}
