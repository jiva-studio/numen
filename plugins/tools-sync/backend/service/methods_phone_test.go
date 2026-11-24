package service

import (
	"fmt"
	"testing"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/server"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/wire/errorwire"
)

func TestGetCode_RefusesAFileWithChangesAsAFailedPrecondition(t *testing.T) {
	if got := errorwire.GetCode(fmt.Errorf("drop: %w", host.ErrChanged)); got != server.CodeFailedPrecondition {
		t.Errorf("got %q, want %q", got, server.CodeFailedPrecondition)
	}
}
