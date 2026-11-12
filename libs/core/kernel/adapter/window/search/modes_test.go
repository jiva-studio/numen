package search

import (
	"testing"

	v1 "github.com/jiva-studio/numen/libs/core/protocol/gen/numen/v1"

	"github.com/jiva-studio/numen/libs/core/kernel/internal/testsupport"
)

func TestEverySearchModeIsRead(t *testing.T) {
	testsupport.CheckHandled(t, func(mode v1.SearchMode) bool {
		_, named := ConvertSearchModeFromWire(mode)
		return named
	})
}
