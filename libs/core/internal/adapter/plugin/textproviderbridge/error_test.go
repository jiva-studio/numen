package textproviderbridge

import (
	"testing"

	v1 "github.com/jiva-studio/numen/libs/protocol/gen/numen/v1"

	"github.com/jiva-studio/numen/libs/core/internal/testsupport"
)

func TestEveryErrorAProviderAnswersIsAnError(t *testing.T) {
	testsupport.CheckHandled(t, func(code v1.TextErrorCode) bool { return readError(code) != nil })
}
