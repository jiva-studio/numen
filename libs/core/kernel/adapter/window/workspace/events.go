package workspace

import (
	"time"

	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapterlib/broadcast"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/tablinks"
)

// maxPendingEvents is the queue size of one watch.
const maxPendingEvents = 1024

// linkEvents delivers the events of the tab links to the windows that watch them. A watch keeps the latest
// event of a topic, in order.
type linkEvents struct {
	*broadcast.Broadcaster[tablinks.Event]
}

func newLinkEvents() linkEvents {
	return linkEvents{&broadcast.Broadcaster[tablinks.Event]{
		IsLatest:  true,
		IsOrdered: true,
		GetTopic:  getEventTopic,
		// A subscriber queues at most one snapshot and one sync per group, so
		// no event is dropped below this many groups.
		Capacity: maxPendingEvents,
	}}
}

func getEventTopic(e tablinks.Event) string {
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

func scheduleAfter(d time.Duration, f func()) func() bool { return time.AfterFunc(d, f).Stop }
