package pluginport_test

import (
	"maps"
	"testing"

	"github.com/jiva-studio/numen/libs/core/port/pluginport"
)

func TestTheValuesOfATargetAreDerivedFromItsPath(t *testing.T) {
	cases := map[string]struct {
		path        string
		contentType string
		noteType    string
		want        map[string]any
	}{
		"the root folder": {"", "", "", map[string]any{
			"target.isRoot": true, "target.isFolder": true, "target.isMultiple": false, "target.isNote": false,
			"target.isFile": false, "target.contentType": "", "target.noteType": "",
		}},
		"a note": {"notes/Bees.md", "text/markdown", "link", map[string]any{
			"target.isRoot": false, "target.isFolder": false, "target.isMultiple": false, "target.isNote": true,
			"target.isFile": false, "target.contentType": "text/markdown", "target.noteType": "link",
		}},
		"a file that is no note": {"papers/Hive.pdf", "application/pdf", "", map[string]any{
			"target.isRoot": false, "target.isFolder": false, "target.isMultiple": false, "target.isNote": false,
			"target.isFile": true, "target.contentType": "application/pdf", "target.noteType": "",
		}},
		"a path of no known type, which may be a folder": {"papers", "application/octet-stream", "", map[string]any{
			"target.isRoot": false, "target.isMultiple": false, "target.isNote": false, "target.contentType": "", "target.noteType": "",
		}},
	}
	for name, one := range cases {
		if got := pluginport.GetTargetValues(one.path, one.contentType, one.noteType); !maps.Equal(got, one.want) {
			t.Errorf("%s: %v, want %v", name, got, one.want)
		}
	}
}
