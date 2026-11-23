package settingevents

import (
	"github.com/jiva-studio/numen/libs/core/kernel/domain/settings"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapterlib/watchhub"
)

// Feed delivers the changes of the settings file at GetPath. A Feed without Events delivers
// nothing, and neither does one whose path cannot be found: the error goes to Report.
type Feed struct {
	Events  *Events
	GetPath func() (string, error)
	Read    func() (settings.Values, error)
	GetInitial func() settings.Values
	Report  func(error)
	Hub     *watchhub.Hub
}

// Subscribe starts delivering to a new reader.
func (f Feed) Subscribe() (<-chan []settings.Setting, func()) {
	if f.Events == nil {
		return createClosedFeed()
	}
	path, err := f.GetPath()
	if err != nil {
		f.Report(err)
		return createClosedFeed()
	}
	return f.Events.Subscribe(Source{
		Read:    f.Read,
		GetInitial: f.GetInitial,
		Report:  f.Report,
		Hub:     f.Hub,
		Path:    path,
	})
}

// Refresh delivers every setting that differs from what the readers were last told.
func (f Feed) Refresh() {
	if f.Events != nil {
		f.Events.Refresh(f.Read)
	}
}

func createClosedFeed() (<-chan []settings.Setting, func()) {
	closed := make(chan []settings.Setting)
	close(closed)
	return closed, func() {}
}
