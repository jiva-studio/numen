package theme

import (
	"testing"

	v1 "github.com/jiva-studio/numen/libs/protocol/gen/numen/v1"

	"github.com/jiva-studio/numen/libs/core/internal/testsupport"
)

// TestEveryOriginIsWrittenFromOne. An origin that the schema defines and that
// has no mapping arrives at a window as the unspecified value, and the window
// draws the theme under neither heading.
func TestEveryOriginIsSavedFromOne(t *testing.T) {
	testsupport.CheckProduced(t, map[v1.Origin]Origin{
		v1.Origin_ORIGIN_PRESET: Preset,
		v1.Origin_ORIGIN_MINE:   Mine,
	}, encodeOrigin)
}
