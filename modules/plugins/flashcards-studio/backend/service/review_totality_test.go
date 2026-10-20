package service

import (
	"testing"

	v1 "github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1"

	"github.com/jiva-studio/numen/modules/libs/protocol/schematest"
)

// TestEveryRatingIsRead. A rating the reading does not name comes through as
// zero, which the use case refuses; a person who pressed a button would be told
// the answer was not one.
func TestEveryRatingIsRead(t *testing.T) {
	schematest.CheckHandled(t, func(r v1.Rating) bool { return parseRating(r) != 0 })
}
