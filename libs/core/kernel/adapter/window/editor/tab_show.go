package editor

import (
	"cmp"
	"context"
	"fmt"
	"io/fs"
	"slices"
	"time"

	"github.com/jiva-studio/numen/libs/note/domain"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/plugin"
)

// defaultTabActivationBound is the wait of ShowTab when State.TabActivationBound is zero.
const defaultTabActivationBound = 2 * time.Second

// ShowTab asks the window to make the open tab with the id the active one and returns the open tabs
// as the window reports them once it is. An id with no open tab returns fs.ErrNotExist, and a tab the
// window does not report active in time returns plugin.ErrTabNotShown.
func (s *State) ShowTab(ctx context.Context, id string) (domain.OpenTabs, error) {
	report := s.openTabsReports.wait()
	open := s.GetOpenTabs()
	if !slices.ContainsFunc(open.Tabs, func(tab domain.Tab) bool { return tab.ID == id }) {
		return domain.OpenTabs{}, fmt.Errorf("editor: no open tab has id %q: %w", id, fs.ErrNotExist)
	}
	s.TabIDs.publish(id)
	if open.ActiveID == id {
		return open, nil
	}
	wait := cmp.Or(s.TabActivationBound, defaultTabActivationBound)
	bound, stop := context.WithTimeout(ctx, wait)
	defer stop()
	for {
		select {
		case <-report:
			report = s.openTabsReports.wait()
			if open := s.GetOpenTabs(); open.ActiveID == id {
				return open, nil
			}
		case <-bound.Done():
			if err := ctx.Err(); err != nil {
				return domain.OpenTabs{}, err
			}
			return domain.OpenTabs{}, fmt.Errorf("editor: activate tab %q within %s: %w", id, wait, plugin.ErrTabNotShown)
		}
	}
}
