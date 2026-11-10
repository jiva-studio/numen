package workspace

import (
	"time"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/tablink"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapterlib/broadcast"
)

// maxPendingEvents is the queue size of one watch.
const maxPendingEvents = 1024

// linkEvents delivers the events of the tab links to the windows that watch them. A watch keeps the latest
// event of a topic, in order.
type linkEvents struct {
	*broadcast.Broadcaster[tablink.Event]
}

func newLinkEvents() linkEvents {
	return linkEvents{&broadcast.Broadcaster[tablink.Event]{
		IsLatest:  true,
		IsOrdered: true,
		GetTopic:  getEventTopic,
		// A subscriber queues at most one snapshot and one sync per group, so
		// no event is dropped below this many groups.
		Capacity: maxPendingEvents,
	}}
}

func getEventTopic(e tablink.Event) string {
	switch {
	case e.Sync != nil:
		return "sync:" + e.Sync.GroupID
	case e.Move != nil:
		return "move:" + e.Move.Tab.ID
	case e.Highlight != nil:
		return "highlight:" + e.Highlight.WindowID
	default:
		return "snapshot"
	}
}

// scheduleAfter runs f once after d and returns the function that stops the run.
func scheduleAfter(d time.Duration, f func()) func() bool { return time.AfterFunc(d, f).Stop }
