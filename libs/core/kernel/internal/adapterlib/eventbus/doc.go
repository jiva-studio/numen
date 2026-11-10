// Package eventbus debounces and throttles the events sent to plugins, so that a burst of changes in the
// application does not become a burst of calls into a plugin process. It holds only the scheduling of
// the events and reaches the plugins through the Trigger it is given.
package eventbus
