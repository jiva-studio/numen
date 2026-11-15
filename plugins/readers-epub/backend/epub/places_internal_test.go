package epub

import (
	"testing"

	"github.com/jiva-studio/numen/plugins/readers-epub/backend/epub/cfi"
	"github.com/jiva-studio/numen/plugins/readers-epub/backend/epub/text"
)

func TestFindDocumentOffsetAdvancesAcrossSplitRunBoundary(t *testing.T) {
	runs := []placedRun{
		{Run: text.Run{Start: 0, End: 6, Steps: []int{4, 2, 1}, Base: 0}, offset: 0},
		{Run: text.Run{Start: 7, End: 10, Steps: []int{4, 2, 1}, Base: 6}, offset: 0},
	}
	got := findDocumentOffset(Document{Offset: 0, Length: 10}, runs, cfi.CFI{Document: []int{4, 2, 1}, Offset: 6})
	if got != 10 {
		t.Errorf("findDocumentOffset at split run Base = %d, want 10", got)
	}
}
