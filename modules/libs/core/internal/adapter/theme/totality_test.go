package theme

import (
	"testing"

	v1 "github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1"

	"github.com/jiva-studio/numen/modules/libs/core/internal/testsupport"
)

// TestEveryShelfIsWrittenFromOne. A shelf the schema names and nothing is put
// on reaches a window as the unspecified value, and the window draws the theme
// under neither heading.
func TestEveryShelfIsWrittenFromOne(t *testing.T) {
	testsupport.Produced(t, map[v1.Shelf]Shelf{
		v1.Shelf_SHELF_PRESET: Preset,
		v1.Shelf_SHELF_MINE:   Mine,
	}, shelved)
}
