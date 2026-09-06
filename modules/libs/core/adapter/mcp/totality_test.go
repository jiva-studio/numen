package mcp

import (
	"testing"

	v1 "github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1"

	"github.com/jiva-studio/numen/modules/libs/core/internal/testsupport"
)

// TestEveryRefusalIsWorded. A refusal the sentences do not name is answered
// with the schema's own spelling of it, which is the one thing an agent cannot
// act on: it says what the value is called and not what to do next.
func TestEveryRefusalIsWorded(t *testing.T) {
	testsupport.Handled(t, func(reason v1.Refusal) bool { return said(reason) != reason.String() })
}
