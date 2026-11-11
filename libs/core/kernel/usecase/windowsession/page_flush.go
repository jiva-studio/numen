package windowsession

import "context"

// PageFlush has every page write what only it holds, then waits for the writes already taken.
type PageFlush struct {
	pages  Pages
	writes Writes
}

func NewPageFlush(pages Pages, writes Writes) *PageFlush {
	return &PageFlush{pages: pages, writes: writes}
}

// Execute flushes the pages and returns false, with no write discarded, when a page holds work that
// needs user confirmation.
func (f *PageFlush) Execute(ctx context.Context) bool {
	if !f.pages.RequestFlush(ctx) {
		return false
	}
	<-f.writes.Seal()
	return true
}

// ReopenWrites accepts writes again.
func (f *PageFlush) ReopenWrites() { f.writes.Open() }

// EndRound ends the round the pages were settled in, so what a page holds from here is new.
func (f *PageFlush) EndRound() { f.pages.CancelFlush() }
