package hostendpoint

import (
	"context"
	"errors"
	"fmt"
	vaultapi "github.com/jiva-studio/numen/libs/core/kernel/adapter/plugin/hostendpoint/vault"
	"net/http"
	"strings"
	"sync"
	"testing"
	"time"

	"connectrpc.com/connect"

	v1 "github.com/jiva-studio/numen/libs/core/protocol/gen/numen/v1"
	"github.com/jiva-studio/numen/libs/core/protocol/gen/numen/v1/numenv1connect"
	"github.com/jiva-studio/numen/libs/note/domain"
	"github.com/jiva-studio/numen/libs/plugin/catalog/manifest"

	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/plugin/transport"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/plugin"
)

func openListEndpoint(t *testing.T, list vaultapi.List) (*Endpoint, numenv1connect.HostVaultServiceClient) {
	t.Helper()
	hostCallers := newCallers(nil)
	hostCallers.vault = vaultOf(newFakeNoteVault())
	endpoint, err := Open(t.Context(), Address, hostCallers.over(list, manifest.PermissionVaultManage))
	if err != nil {
		t.Fatal(err)
	}
	url := fmt.Sprintf("http://127.0.0.1:%d", endpoint.GetPort())
	return endpoint, numenv1connect.NewHostVaultServiceClient(http.DefaultClient, url, connect.WithInterceptors(transport.AuthInterceptor("token-a")))
}

func TestOpenVaultWithUnsavedChangesReturnsUnsavedWithTheDetail(t *testing.T) {
	list := newRecordingList()
	unsaved := fmt.Errorf("open vault v2: %w", plugin.ErrUnsavedChanges)
	list.err = unsaved
	endpoint, client := openListEndpoint(t, list)
	t.Cleanup(func() {
		if err := endpoint.Close(context.WithoutCancel(t.Context())); err != nil {
			t.Error(err)
		}
	})

	res, err := client.OpenVault(t.Context(), connect.NewRequest(&v1.HostVaultServiceOpenVaultRequest{Id: "v2"}))
	if err != nil {
		t.Fatal(err)
	}

	if res.Msg.GetError() != v1.VaultsErrorCode_VAULTS_ERROR_CODE_UNSAVED {
		t.Errorf("OpenVault of a window with unsaved changes returned %v, want error UNSAVED", res.Msg)
	}
	const question = "the window has unsaved changes and is prompting the user; retry after the prompt closes"
	if detail := res.Msg.GetDetail(); !strings.Contains(detail, unsaved.Error()) || !strings.Contains(detail, question) {
		t.Errorf("OpenVault of a window with unsaved changes has the detail %q, want it to contain %q and %q", detail, unsaved.Error(), question)
	}
	if got := list.getCalls(); len(got) != 0 {
		t.Errorf("the list opened %v, want nothing", got)
	}
}

func TestTwoOpenVaultCallsAtOnceBothReturnAndBothOpen(t *testing.T) {
	list := newRecordingList()
	endpoint, client := openListEndpoint(t, list)
	var wg sync.WaitGroup
	for range 2 {
		wg.Go(func() {
			res, err := client.OpenVault(t.Context(), connect.NewRequest(&v1.HostVaultServiceOpenVaultRequest{Id: "v2"}))
			if err != nil || res.Msg.GetError() != 0 {
				t.Errorf("OpenVault of v2 returned %v (%v)", res, err)
			}
		})
	}
	wg.Wait()
	// Both calls have returned, so both opens have ended before the endpoint closes.
	if got := list.getCalls(); len(got) != 2 {
		t.Errorf("the opens that ran were %v, want 2", got)
	}
	if err := endpoint.Close(context.WithoutCancel(t.Context())); err != nil {
		t.Fatal(err)
	}
}

func TestOpenVaultWhileTheWindowFlushesIsUnavailable(t *testing.T) {
	list := newRecordingList()
	list.err = fmt.Errorf("open vault v2: %w: %w", domain.ErrUnavailable, errors.New("the window is flushing pending writes"))
	endpoint, client := openListEndpoint(t, list)
	defer func() {
		if err := endpoint.Close(context.WithoutCancel(t.Context())); err != nil {
			t.Error(err)
		}
	}()

	_, err := client.OpenVault(t.Context(), connect.NewRequest(&v1.HostVaultServiceOpenVaultRequest{Id: "v2"}))

	var failure *connect.Error
	if !errors.As(err, &failure) || failure.Code() != connect.CodeUnavailable {
		t.Errorf("OpenVault while the window flushes returned %v, want unavailable", err)
	}
	if got := list.getCalls(); len(got) != 0 {
		t.Errorf("the opens that ran were %v, want none", got)
	}
}

// A call still running when the shutdown tolerance ends is aborted by Close, which returns an error.
func TestCloseWithCallRunningPastShutdownToleranceReturnsError(t *testing.T) {
	t.Parallel()
	list := newRecordingList()
	list.release = make(chan struct{})
	t.Cleanup(func() { close(list.release) })
	endpoint, client := openListEndpoint(t, list)
	callResult := make(chan error, 1)
	go func() {
		_, err := client.OpenVault(context.WithoutCancel(t.Context()), connect.NewRequest(&v1.HostVaultServiceOpenVaultRequest{Id: "v2"}))
		callResult <- err
	}()
	select {
	case <-list.openCh:
	case <-time.After(10 * time.Second):
		t.Fatal("the window received no call to open the vault")
	}

	if err := endpoint.Close(context.WithoutCancel(t.Context())); err == nil {
		t.Error("Close returned nil although a call ran past the shutdown tolerance")
	}
	select {
	case err := <-callResult:
		if err == nil {
			t.Error("the call aborted by Close returned nil, want an error")
		}
	case <-time.After(10 * time.Second):
		t.Fatal("the call aborted by Close never returned")
	}
}
