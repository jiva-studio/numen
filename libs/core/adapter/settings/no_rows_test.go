package settings

import "testing"

func TestNoRowsListsNoRowForAnyPluginAndOperation(t *testing.T) {
	rows, err := NoRows{}.ListRows("md.numen.tools.proofreading", "complete")

	if err != nil || len(rows) != 0 {
		t.Errorf("ListRows = %v, %v, want no row", rows, err)
	}
}
