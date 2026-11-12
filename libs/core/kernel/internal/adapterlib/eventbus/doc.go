// Package eventbus debounces and throttles the events sent to plugins, so that a burst of changes in the
// application does not become a burst of calls into a plugin process. It holds only the scheduling of
// the events and reaches the plugins through the Trigger it is given.
//
// Events that share a debounce key are collapsed to the last one published, and a key is delivered at
// most one throttle interval after the first event of its burst, and sooner when a timer that was
// stopped too late fires. Each key is delivered on its own goroutine, so the order of two events with
// different keys is not kept: a plugin must not rely on one arriving before the other.
package eventbus
