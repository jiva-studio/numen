package windowsession

import (
	"cmp"
	"context"
	"fmt"
	windowdomain "github.com/jiva-studio/numen/libs/core/kernel/domain/window"
	"slices"
	"time"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/plugin"
)

// defaultTabBound is how long ShowTab waits when the bound is zero.
const defaultTabBound = 2 * time.Second

// ShowTab asks the window to make an open tab the active one.
type ShowTab struct {
	reports  *TabReports
	tabFront TabFront
	bound    time.Duration
}

// NewShowTab makes the use case over the tab reports of the window and the front of its tabs. A zero
// bound waits for defaultTabBound.
func NewShowTab(reports *TabReports, tabFront TabFront, bound time.Duration) ShowTab {
	return ShowTab{reports: reports, tabFront: tabFront, bound: bound}
}

// Execute asks the window to make the open tab with the id the active one and returns the open tabs
// as the window reports them once it is. An id with no open tab returns fs.ErrNotExist, and a tab the
// window does not report active in time returns plugin.ErrTabNotShown.
func (u ShowTab) Execute(ctx context.Context, id string) (windowdomain.OpenTabs, error) {
	report := u.reports.signal.wait()
	open := u.reports.GetOpenTabs()
	if !slices.ContainsFunc(open.Tabs, func(tab windowdomain.Tab) bool { return tab.ID == id }) {
		return windowdomain.OpenTabs{}, fmt.Errorf("editor: %q: %w", id, plugin.ErrTabNotOpen)
	}
	u.tabFront.Publish(id)
	if open.ActiveID == id {
		return open, nil
	}
	wait := cmp.Or(u.bound, defaultTabBound)
	bound, stop := context.WithTimeout(ctx, wait)
	defer stop()
	for {
		select {
		case <-report:
			report = u.reports.signal.wait()
			if open := u.reports.GetOpenTabs(); open.ActiveID == id {
				return open, nil
			}
		case <-bound.Done():
			if err := ctx.Err(); err != nil {
				return windowdomain.OpenTabs{}, err
			}
			return windowdomain.OpenTabs{}, fmt.Errorf("editor: activate tab %q within %s: %w", id, wait, plugin.ErrTabNotShown)
		}
	}
}
