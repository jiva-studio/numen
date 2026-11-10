package theme

import (
	"testing"

	v1 "github.com/jiva-studio/numen/libs/core/protocol/gen/numen/v1"

	themedomain "github.com/jiva-studio/numen/libs/core/kernel/domain/theme"

	"github.com/jiva-studio/numen/libs/core/kernel/internal/testsupport"
)

// TestEveryOriginIsWrittenFromOne. An origin that the schema defines and that
// has no mapping arrives at a window as the unspecified value, and the window
// draws the theme under neither heading.
func TestEveryOriginIsSavedFromOne(t *testing.T) {
	testsupport.CheckProduced(t, map[v1.Origin]themedomain.Origin{
		v1.Origin_ORIGIN_PRESET: themedomain.Preset,
		v1.Origin_ORIGIN_MINE:   themedomain.Mine,
		v1.Origin_ORIGIN_PLUGIN: themedomain.Plugin,
	}, encodeOrigin)
}
