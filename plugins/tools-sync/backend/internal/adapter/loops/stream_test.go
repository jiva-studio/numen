package loops_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/jiva-studio/numen/libs/shared/syncprotocol"
	syncclient "github.com/jiva-studio/numen/libs/shared/syncprotocol/client"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/internal/adapter/loops"
)

func TestNewStreamHTTP_HasNoTimeoutThatEndsTheReadOfABody(t *testing.T) {
	t.Parallel()

	if got := loops.NewStreamHTTP(time.Second).Timeout; got != 0 {
		t.Errorf("Timeout = %v, want none", got)
	}
}

func TestNewStreamHTTP_FailsAnOpenWhoseHeadersNeverComeAsUnreachable(t *testing.T) {
	t.Parallel()
	release := make(chan struct{})
	silent := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { <-release }))
	t.Cleanup(func() {
		close(release)
		silent.Close()
	})
	ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
	defer cancel()
	client := syncclient.NewWatchClient(silent.URL, loops.NewStreamHTTP(50*time.Millisecond), func() string { return "t" })

	_, err := client.WatchChanges(ctx)

	if !errors.Is(err, syncprotocol.ErrUnreachable) {
		t.Errorf("error = %v, want ErrUnreachable at the header timeout", err)
	}
}
