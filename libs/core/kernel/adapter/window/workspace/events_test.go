package workspace

import (
	"strconv"
	"testing"
	"time"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/tablink"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/tablinks"
)

// A watch that reads late receives the last sync of each group.
func TestSlowWatchReceivesLastSyncOfEachGroup(t *testing.T) {
	links := tablinks.New(LinkGrace, scheduleNothing, newLinkEvents())
	watch := links.Windows.Watch("window-a")
	defer watch.Close()
	const groups = 20
	var tabs []tablink.Tab
	for index := range groups * 2 {
		tabs = append(tabs, tablink.Tab{ID: "plex:" + strconv.Itoa(index), Kind: "plex"})
	}
	links.Windows.WriteTabs(tablinks.Window{ID: "window-a"}, tabs)
	for index := 0; index < groups*2; index += 2 {
		links.Links.Link(tabs[index].ID, tabs[index+1].ID)
	}

	for index := 0; index < groups*2; index += 2 {
		links.Links.SyncPath(tabs[index].ID, "first.md", "")
		links.Links.SyncPath(tabs[index].ID, "last.md", "")
	}

	paths := map[string]string{}
	for len(paths) < groups {
		select {
		case event := <-watch.Events:
			if event.Sync != nil {
				paths[event.Sync.GroupID] = event.Sync.Path
			}
		case <-time.After(5 * time.Second):
			t.Fatalf("syncs arrived for %d groups, want %d", len(paths), groups)
		}
	}
	for group, path := range paths {
		if path != "last.md" {
			t.Errorf("group %s last moved to %q, want last.md", group, path)
		}
	}
}
