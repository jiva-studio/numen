package theme

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"time"

	themedomain "github.com/jiva-studio/numen/libs/core/kernel/domain/theme"
	"github.com/jiva-studio/numen/libs/core/kernel/port/themeport"
)

// ErrWatchUnavailable is the failure to start watching the user's folder.
var ErrWatchUnavailable = errors.New("the themes cannot be watched")

// keepaliveInterval is the interval between keepalive reports. A failed send is how a watch detects
// that its client has disconnected.
const keepaliveInterval = time.Second

// WatchThemes reports when the user's folder or the set of plugin themes changes.
type WatchThemes struct {
	catalog catalog
	watch   themeport.OwnThemeWatch
	hold    time.Duration
}

// NewWatchThemes makes the watch over the sources. The hold is how long changes are kept before
// they are reported; zero takes the default of the watch.
func NewWatchThemes(sources Sources, watch themeport.OwnThemeWatch, hold time.Duration) *WatchThemes {
	return &WatchThemes{catalog: catalog{sources: sources}, watch: watch, hold: hold}
}

// Execute sends the names of the themes that changed until ctx ends or send fails. The plugin set is
// compared on each keepalive tick, and the names added or removed are sent in place of the empty
// report. Where the user's folder cannot be watched, a build with plugin themes watches them alone;
// any other failure to watch wraps ErrWatchUnavailable.
func (u *WatchThemes) Execute(ctx context.Context, send func(names []string) error) error {
	// The folder watch ends with this call, whichever way it ends.
	watching, stop := context.WithCancel(ctx)
	defer stop()

	changed, err := u.watchOwnThemes(watching)
	if err != nil {
		return err
	}

	previous := u.catalog.listPluginThemeNames()

	// Empty reports make a send to a disconnected client fail. The request context belongs to the
	// process and does not reflect whether the page that opened the stream is still open.
	repeat := time.NewTicker(keepaliveInterval)
	defer repeat.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-repeat.C:
			current := u.catalog.listPluginThemeNames()
			names := listDifference(previous, current)
			previous = current
			if err := send(names); err != nil {
				return err
			}
		case titles, open := <-changed:
			if !open {
				return nil
			}
			if err := send(listOwnNames(titles)); err != nil {
				return err
			}
		}
	}
}

func (u *WatchThemes) watchOwnThemes(ctx context.Context) (<-chan []string, error) {
	changed, err := u.watch.WatchOwnThemes(ctx, u.hold)
	if errors.Is(err, themedomain.ErrNoFolder) && u.catalog.sources.Plugins != nil {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrWatchUnavailable, err)
	}
	return changed, nil
}

func listOwnNames(titles []string) []string {
	names := make([]string, 0, len(titles))
	for _, title := range titles {
		names = append(names, themedomain.GetName(themedomain.Mine, title))
	}
	return names
}

func listDifference(before, after []string) []string {
	var different []string
	for _, name := range before {
		if !slices.Contains(after, name) {
			different = append(different, name)
		}
	}
	for _, name := range after {
		if !slices.Contains(before, name) {
			different = append(different, name)
		}
	}
	return different
}
