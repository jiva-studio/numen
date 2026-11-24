package service

import (
	"errors"
	"fmt"
	"testing"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/server"
	"github.com/jiva-studio/numen/libs/shared/syncprotocol"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/usecase/fetchfile"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/wire/errorwire"
)

func TestGetWarning_NamesTheFailuresOfADownload(t *testing.T) {
	for name, one := range map[string]struct {
		err  error
		want string
	}{
		"server away":      {err: fmt.Errorf("fetch: %w", syncprotocol.ErrUnreachable), want: "Could not reach the sync server"},
		"blob missing":     {err: fmt.Errorf("fetch: %w", fetchfile.ErrBlobMissing), want: "Could not download the file"},
		"file has changes": {err: fmt.Errorf("drop: %w", host.ErrChanged), want: "The file has changes that are not sent"},
		"another error":    {err: errors.New("broken"), want: ""},
	} {
		if got := errorwire.GetWarning(one.err); got != one.want {
			t.Errorf("%s: got %q, want %q", name, got, one.want)
		}
	}
}

func TestGetCode_RefusesAFileWithChangesAsAFailedPrecondition(t *testing.T) {
	if got := errorwire.GetCode(fmt.Errorf("drop: %w", host.ErrChanged)); got != server.CodeFailedPrecondition {
		t.Errorf("got %q, want %q", got, server.CodeFailedPrecondition)
	}
}
