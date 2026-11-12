// Package window holds the values the core hands to the window it runs in.
package window

// ErrorHandler receives errors from work the core continues past. A caller that
// sets none receives no notification.
type ErrorHandler func(error)
