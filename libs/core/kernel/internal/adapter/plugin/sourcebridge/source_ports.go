package sourcebridge

import "github.com/jiva-studio/numen/libs/core/kernel/port/searchsourceport"

// WatchedSource is a Source whose description can change while the window is open, as it does when the
// settings of its plugin change.
type WatchedSource interface {
	searchsourceport.Source
	searchsourceport.DescriptionWatcher
}
