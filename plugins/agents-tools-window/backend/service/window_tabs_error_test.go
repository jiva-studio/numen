package service_test

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/jiva-studio/numen/libs/plugin-sdk-go/host"
)

// tabWithError is a tab as window_tabs_get answers it, with the error of a state it could not read.
type tabWithError struct {
	ID       string          `json:"id"`
	Location json.RawMessage `json:"location"`
	Error    string          `json:"error"`
}

func TestWindowTabsGetAnswersEveryTabAndNamesTheOneWhoseStateCannotBeRead(t *testing.T) {
	for name, state := range map[string]string{
		"a state that is not JSON":           `page 4`,
		"a state that is a JSON array":       `[4,120]`,
		"a state that is a JSON string":      `"page 4"`,
		"a state that is JSON null":          `null`,
		"a page that is a string":            `{"page":"four"}`,
		"a page of zero":                     `{"page":0}`,
		"a negative page":                    `{"page":-3}`,
		"a page that is a fraction":          `{"page":2.5}`,
		"a page that is null":                `{"page":null}`,
		"a page count of zero":               `{"page":4,"pageCount":0}`,
		"a page count that is a string":      `{"page":4,"pageCount":"120"}`,
		"a selection that is empty":          `{"t":[]}`,
		"a selection that is a number":       `{"t":66.5}`,
		"a selection that starts below zero": `{"t":[-1,4]}`,
		"a selection that holds a string":    `{"t":["66.5",70]}`,
		"a selection of nulls":               `{"t":[null,null]}`,
		"a selection that starts at null":    `{"t":[null,70]}`,
		"a selection that ends at null":      `{"t":[66.5,null]}`,
		"a CFI that is a number":             `{"cfi":4}`,
		"a chapter that is an array":         `{"cfi":"epubcfi(/6/4!/4/2/1:0)","chapter":["The Reed Beds"]}`,
		"a chapter that is a number":         `{"cfi":"epubcfi(/6/4!/4/2/1:0)","chapter":7}`,
	} {
		t.Run(name, func(t *testing.T) {
			w := newWindowTools(t)
			w.window.SetOpenTabs([]host.OpenTab{
				{ID: "tab-1", Kind: "reader:reader-pdf", Path: guidePath, Title: "Bird Guide", State: `{"page":4,"pageCount":120}`},
				{ID: "tab-7", Kind: "reader:reader-pdf", Path: guidePath, Title: "Bird Guide", State: state},
				{ID: "tab-9", Kind: "recording", Path: talkPath, Title: "Heron Talk", State: `{"t":[66.5,70]}`},
			}, "tab-1")

			var out struct {
				Tabs []tabWithError `json:"tabs"`
			}
			if failure := w.call(t, "window_tabs_get", `{}`, nil, &out); failure != "" {
				t.Fatalf("the call failed with %q, want every tab answered", failure)
			}

			if len(out.Tabs) != 3 || out.Tabs[0].ID != "tab-1" || out.Tabs[1].ID != "tab-7" || out.Tabs[2].ID != "tab-9" {
				t.Fatalf("answered the tabs %+v, want tab-1, tab-7 and tab-9", out.Tabs)
			}
			if broken := out.Tabs[1]; !strings.Contains(broken.Error, "tab-7") || broken.Location != nil {
				t.Errorf("tab-7 is answered %+v, want an error naming tab-7 and no location", broken)
			}
			for _, tab := range []tabWithError{out.Tabs[0], out.Tabs[2]} {
				if tab.Error != "" || tab.Location == nil {
					t.Errorf("%s is answered %+v, want its location and no error", tab.ID, tab)
				}
			}
		})
	}
}
