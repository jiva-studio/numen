package webui

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

	_, err := api.Attending(t.Context(), connect.NewRequest(&v1.AttendingRequest{
		Front: "two",
		Tabs: []*v1.Tab{
			{Id: "one", Kind: "plex", Path: "Entropy.md", Title: "Entropy"},
			{Id: "two", Kind: "recording", Path: "Talk.mp3", Title: "Talk.mp3", At: 1000, Of: 4000},
		},
	}))
	if err != nil {
		t.Fatal(err)
	}

	want := domain.Attention{
		FrontID: "two",
		Tabs: []domain.Tab{
			{ID: "one", Kind: "plex", Path: "Entropy.md", Title: "Entropy"},
			{ID: "two", Kind: "recording", Path: "Talk.mp3", Title: "Talk.mp3", At: 1000, Of: 4000},
		},
	}
	if got := api.Attended(); !reflect.DeepEqual(got, want) {
		t.Errorf("the window has %+v open", got)
	}
}

func TestAttendingTellsWhoeverIsListening(t *testing.T) {
	var heard []domain.Attention
	api := &API{Attends: func(open domain.Attention) { heard = append(heard, open) }}

	_, err := api.Attending(t.Context(), connect.NewRequest(&v1.AttendingRequest{
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
		open domain.Attention
		want string
	}{
		"a tab in front": {
			open: domain.Attention{
				FrontID: "two",
				Tabs:    []domain.Tab{{ID: "one", Path: "Entropy.md"}, {ID: "two", Path: "Talk.mp3"}},
			},
			want: "Talk.mp3",
		},
		"nothing open":     {open: domain.Attention{}},
		"a tab it lets go": {open: domain.Attention{FrontID: "three", Tabs: []domain.Tab{{ID: "one"}}}},
		"a tab with no identity": {
			open: domain.Attention{Tabs: []domain.Tab{{Path: "Entropy.md"}}},
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
