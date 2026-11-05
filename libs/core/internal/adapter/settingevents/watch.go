package settingevents

import (
	"os"
	"path/filepath"
	"time"

	"github.com/rjeczalik/notify"

	"github.com/jiva-studio/numen/libs/core/internal/adapterlib/watchhub"
)

// saveDelay is how long the settings file is left alone after an event of the system before it is
// read. A save is several events: the file is truncated, written and renamed.
const saveDelay = 100 * time.Millisecond

// backlog is how many events of the system wait for the delay.
const backlog = 64

// watchFile calls changed after the file is saved by anyone, until the returned function is called.
// The folder is watched and not the file: a save replaces the file. The files beside it, such as
// the lock, are no event.
func watchFile(hub *watchhub.Hub, path string, report func(error), changed func()) func() {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		report(err)
		return func() {}
	}
	raw := make(chan notify.EventInfo, backlog)
	stop, err := hub.Watch(dir, false, raw)
	if err != nil {
		report(err)
		return func() {}
	}
	quit := make(chan struct{})
	go joinSaves(filepath.Base(path), raw, quit, changed)
	return func() {
		stop()
		close(quit)
	}
}

// joinSaves calls changed once for the events of one save of the file named. The delay runs from
// the first event, so a file written continuously is still reported.
func joinSaves(name string, raw <-chan notify.EventInfo, quit <-chan struct{}, changed func()) {
	var due <-chan time.Time
	for {
		select {
		case <-quit:
			return
		case event := <-raw:
			if filepath.Base(event.Path()) == name && due == nil {
				due = time.After(saveDelay)
			}
		case <-due:
			due = nil
			changed()
		}
	}
}
