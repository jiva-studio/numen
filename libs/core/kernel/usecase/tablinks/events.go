package tablinks

import "time"

// Events delivers the events of the tab links to the watching windows.
type Events interface {
	// Publish queues the event for every subscriber.
	Publish(event Event)
	// Subscribe starts a subscription: the channel of the events published from now on, and the function
	// that ends it and closes the channel.
	Subscribe() (<-chan Event, func())
}

// Schedule runs f once after d and returns a function that cancels the run. The cancel function returns
// whether the run was prevented.
type Schedule func(d time.Duration, f func()) (cancel func() bool)
