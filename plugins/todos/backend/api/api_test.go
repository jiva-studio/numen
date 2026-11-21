package api

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func readFixture(t *testing.T, name string) []byte {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("..", "..", "testdata", "api", name+".json"))
	if err != nil {
		t.Fatal(err)
	}
	return data
}

func sampleTasks() []Task {
	return []Task{
		{
			Path: "Projects/Home.md", Line: 7, Raw: "- [ ] (A) buy milk due:2026-11-13 rec:!1w +home @phone", Text: "buy milk",
			Status: "open", Priority: "A", Due: "2026-11-13", DoneOn: "", Recurrence: "rec:!1w",
			Projects: []string{"home"}, Contexts: []string{"phone"}, Issues: []string{}, Depth: 0,
		},
		{
			Path: "Daily/2026-11-12.md", Line: 3, Raw: "  - [ ] call due:2026-02-30", Text: "call due:2026-02-30",
			Status: "open", Priority: "", Due: "", DoneOn: "", Recurrence: "",
			Projects: []string{}, Contexts: []string{}, Issues: []string{"invalidDue"}, Depth: 1,
		},
	}
}

// Each message encodes to the JSON of its fixture and the fixture decodes to the message, so the
// TypeScript tests that read the same fixture test the same keys.
func TestEveryMessageMatchesItsFixture(t *testing.T) {
	cases := []struct {
		fixture string
		message any
		decoded any
	}{
		{"GetSummary.request", GetSummaryRequest{Today: "2026-11-14"}, new(GetSummaryRequest)},
		{"GetSummary.response", Summary{IsReady: true, OverdueCount: 2, TodayCount: 1, NoDateCount: 3, UnreadableCount: 1}, new(Summary)},
		{"ListTasks.request", ListTasksRequest{
			Today: "2026-11-14",
			Filter: Filter{
				Status: []string{"open"}, Due: "overdue", From: "", To: "", Projects: []string{"home"},
				Contexts: []string{"phone"}, Priorities: []string{"A"}, Path: "Projects/", Text: "milk",
			},
			Limit: 20,
		}, new(ListTasksRequest)},
		{"ListTasks.response", ListTasksResponse{Tasks: sampleTasks(), Total: 31, Revision: 12}, new(ListTasksResponse)},
		{"RefreshIndex.request", RefreshIndexRequest{
			Paths:   []string{"Daily/2026-11-14.md", "Projects/Home.md"},
			Renamed: []Rename{{From: "Inbox.md", To: "Projects/Inbox.md"}},
		}, new(RefreshIndexRequest)},
		{"RefreshIndex.response", RefreshIndexResponse{}, new(RefreshIndexResponse)},
		{"WatchTasks.request", WatchTasksRequest{}, new(WatchTasksRequest)},
		{"WatchTasks.response", WatchTasksMessage{Revision: 12, IsReady: true}, new(WatchTasksMessage)},
		{"SetTaskDone.request", SetTaskDoneRequest{
			Path: "Projects/Home.md", Line: 7, Raw: "- [ ] fix the tap +home ^tap", IsDone: true, Today: "2026-11-14",
		}, new(SetTaskDoneRequest)},
		{"SetTaskDone.response", SetTaskDoneResponse{
			Task: Task{
				Path: "Projects/Home.md", Line: 7, Raw: "- [x] fix the tap +home done:2026-11-14 ^tap", Text: "fix the tap ^tap",
				Status: "done", Priority: "", Due: "", DoneOn: "2026-11-14", Recurrence: "",
				Projects: []string{"home"}, Contexts: []string{}, Issues: []string{}, Depth: 0,
			},
			Change: "completed",
			Error:  "",
		}, new(SetTaskDoneResponse)},
	}
	for _, tc := range cases {
		t.Run(tc.fixture, func(t *testing.T) {
			fixture := readFixture(t, tc.fixture)

			encoded, err := json.Marshal(tc.message)
			if err != nil {
				t.Fatal(err)
			}
			var got, want any
			if err := json.Unmarshal(encoded, &got); err != nil {
				t.Fatal(err)
			}
			if err := json.Unmarshal(fixture, &want); err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(got, want) {
				t.Errorf("encoded = %s, fixture = %s", encoded, fixture)
			}

			if err := json.Unmarshal(fixture, tc.decoded); err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(reflect.ValueOf(tc.decoded).Elem().Interface(), tc.message) {
				t.Errorf("decoded = %+v, want %+v", tc.decoded, tc.message)
			}
		})
	}
}

func TestTheServiceNameIsTheOneTheManifestDeclares(t *testing.T) {
	if Service != "md.numen.todos" {
		t.Errorf("Service = %q", Service)
	}
}
