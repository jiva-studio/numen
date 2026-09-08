package editor

import (
	"reflect"
	"testing"

	"connectrpc.com/connect"

	v1 "github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1"

	"github.com/jiva-studio/numen/modules/libs/core/domain"
)

// The window says what the person has open, and whoever answers on their behalf
// reads the last of it.

func TestAttendingIsWhatTheWindowLastSaid(t *testing.T) {
	api := &API{}

	if open := api.Attended(); len(open.Tabs) != 0 || open.FrontID != "" {
		t.Fatalf("a window that has said nothing has %+v open", open)
	}

	_, err := api.WriteOpenTabs(t.Context(), connect.NewRequest(&v1.WriteOpenTabsRequest{
		Front: "two",
		Tabs: []*v1.Tab{
			{Id: "one", Kind: "plex", Path: "Entropy.md", Title: "Entropy"},
			{Id: "two", Kind: "recording", Path: "Talk.mp3", Title: "Talk.mp3",
				Recording: &v1.RecordingProgress{TranscribedDuration: 1000, Duration: 4000}},
		},
	}))
	if err != nil {
		t.Fatal(err)
	}

	want := domain.OpenTabs{
		FrontID: "two",
		Tabs: []domain.Tab{
			{ID: "one", Kind: "plex", Path: "Entropy.md", Title: "Entropy"},
			{ID: "two", Kind: "recording", Path: "Talk.mp3", Title: "Talk.mp3",
				Recording: &domain.RecordingProgress{TranscribedDuration: 1000, Duration: 4000}},
		},
	}
	if got := api.Attended(); !reflect.DeepEqual(got, want) {
		t.Errorf("the window has %+v open", got)
	}
}

func TestAttendingTellsWhoeverIsListening(t *testing.T) {
	var heard []domain.OpenTabs
	api := &API{Attends: func(open domain.OpenTabs) { heard = append(heard, open) }}

	_, err := api.WriteOpenTabs(t.Context(), connect.NewRequest(&v1.WriteOpenTabsRequest{
		Front: "one",
		Tabs:  []*v1.Tab{{Id: "one", Kind: "note", Path: "Entropy.md", Title: "Entropy"}},
	}))
	if err != nil {
		t.Fatal(err)
	}

	if len(heard) != 1 {
		t.Fatalf("what the window has open was said %d times", len(heard))
	}
	front, held := heard[0].Fronted()
	if !held || front.Path != "Entropy.md" {
		t.Errorf("the tab in front is %+v", front)
	}
}

func TestFrontedIsTheTabThePersonIsLookingAt(t *testing.T) {
	for name, c := range map[string]struct {
		open domain.OpenTabs
		want string
	}{
		"a tab in front": {
			open: domain.OpenTabs{
				FrontID: "two",
				Tabs:    []domain.Tab{{ID: "one", Path: "Entropy.md"}, {ID: "two", Path: "Talk.mp3"}},
			},
			want: "Talk.mp3",
		},
		"nothing open":     {open: domain.OpenTabs{}},
		"a tab it lets go": {open: domain.OpenTabs{FrontID: "three", Tabs: []domain.Tab{{ID: "one"}}}},
		"a tab with no identity": {
			open: domain.OpenTabs{Tabs: []domain.Tab{{Path: "Entropy.md"}}},
		},
	} {
		t.Run(name, func(t *testing.T) {
			front, held := c.open.Fronted()
			if held != (c.want != "") {
				t.Fatalf("a tab in front is %v", held)
			}
			if front.Path != c.want {
				t.Errorf("the tab in front holds %q", front.Path)
			}
		})
	}
}
