package container_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"connectrpc.com/connect"

	"github.com/jiva-studio/numen/libs/core/kernel/container"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/filesystem/vaultconfig"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapterlib/wire"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/testsupport"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/windowsession"
	v1 "github.com/jiva-studio/numen/libs/core/protocol/gen/numen/v1"
	"github.com/jiva-studio/numen/libs/core/protocol/gen/numen/v1/numenv1connect"
	"github.com/jiva-studio/numen/libs/note/domain"
)

// switchSpy remembers the context and the plugin open of the first switch a window runs.
type switchSpy struct {
	next windowsession.VaultSwitch
	mu   sync.Mutex
	ctx  context.Context
	open *windowsession.PluginOpen
}

func (s *switchSpy) Execute(ctx context.Context, v domain.Vault, open *windowsession.PluginOpen) error {
	s.mu.Lock()
	if s.ctx == nil {
		s.ctx, s.open = ctx, open
	}
	s.mu.Unlock()
	return s.next.Execute(ctx, v, open)
}

func (s *switchSpy) getFirst() (context.Context, *windowsession.PluginOpen) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.ctx, s.open
}

func newRegisteredVault(t *testing.T, cfg container.Config, name string) domain.Vault {
	t.Helper()
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, name+".md"), []byte("# "+name+"\n\nInvented text.\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := vaultconfig.Initialize(root, vaultconfig.DefaultServiceDir, testsupport.GetEpoch()); err != nil {
		t.Fatal(err)
	}
	v, err := cfg.AddVault(root, name)
	if err != nil {
		t.Fatal(err)
	}
	return v
}

// A vault switch given a plugin's open that has ended takes the exclusion like
// any other switch: during the user's close it is rejected and the close completes.
func TestVaultSwitchAfterPluginOpenFailsDuringUserClose(t *testing.T) {
	cfg := container.NewConfig(container.Config{
		Paths: container.Paths{IndexPath: filepath.Join(t.TempDir(), "index.db"), RegistryPath: filepath.Join(t.TempDir(), "vaults.json")},
	})
	first := newRegisteredVault(t, cfg, "one")
	second := newRegisteredVault(t, cfg, "two")
	db, err := container.OpenIndex(cfg, t.Context())
	if err != nil {
		t.Fatal(err)
	}
	spy := &switchSpy{}
	cases := &container.WindowUseCases{}
	installation, err := container.OpenWindowSwitchingThrough(cfg, t.Context(),
		container.WindowOpening{DB: db, Named: "one", Out: os.Stderr, UseCases: cases},
		func(next windowsession.VaultSwitch) windowsession.VaultSwitch { spy.next = next; return spy })
	if err != nil {
		t.Fatal(err)
	}
	testsupport.CloseAfter(t, installation)

	path, handler := numenv1connect.NewWindowServiceHandler(installation.State.Window)
	mux := http.NewServeMux()
	mux.Handle(path, handler)
	server := httptest.NewUnstartedServer(mux)
	server.EnableHTTP2 = true
	server.StartTLS()
	t.Cleanup(server.CloseClientConnections)
	t.Cleanup(server.Close)
	canvas := numenv1connect.NewWindowServiceClient(server.Client(), server.URL)

	listening, hangUp := context.WithCancel(t.Context())
	defer hangUp()
	stream, err := canvas.WatchQuit(listening, connect.NewRequest(&v1.WatchQuitRequest{Window: wire.Editor}))
	if err != nil {
		t.Fatal(err)
	}
	defer testsupport.Close(t, stream)
	// The token before the first request.
	if !stream.Receive() {
		t.Fatalf("the stream never opened: %v", stream.Err())
	}
	reportFlush := func(token string) error {
		ctx, cancel := context.WithTimeout(context.WithoutCancel(t.Context()), 5*time.Second)
		defer cancel()
		_, err := canvas.ReportFlush(ctx, connect.NewRequest(&v1.ReportFlushRequest{
			Window: wire.Editor, Token: token, Result: v1.FlushResult_FLUSH_RESULT_NOTHING,
		}))
		return err
	}
	secondFlush := make(chan struct{}, 1)
	release := make(chan struct{})
	go func() {
		count := 0
		for stream.Receive() {
			if !stream.Msg().GetFlush() {
				continue
			}
			count++
			if count == 2 {
				secondFlush <- struct{}{}
				<-release
			}
			if reportFlush(stream.Msg().GetToken()) != nil {
				return
			}
		}
	}()

	list := windowsession.NewPluginVaults(&installation.State.Runtime, windowsession.VaultOps{
		Edit: windowsession.VaultEdits{
			Add: cases.Vaults.Change.Add, Rename: cases.Vaults.Change.Rename, Remove: cases.Vaults.Change.Remove,
		},
		Find:    cases.Listing.Find,
		Missing: cases.Listing.FolderCheck,
		Open:    installation,
	})
	opened := make(chan error, 1)
	go func() { opened <- list.OpenVault(t.Context(), second.ID) }()
	select {
	case err := <-opened:
		if err != nil {
			t.Fatalf("the plugin's open returned %v", err)
		}
	case <-time.After(20 * time.Second):
		t.Fatal("the plugin's open never returned")
	}

	closeCtx, cancel := context.WithTimeout(t.Context(), 20*time.Second)
	defer cancel()
	closeResult := make(chan bool, 1)
	go func() { closeResult <- installation.FlushPending(closeCtx) }()
	select {
	case <-secondFlush:
	case <-time.After(5 * time.Second):
		close(release)
		t.Fatal("the user's close never requested a flush from the page")
	}
	switchCtx, switchOpen := spy.getFirst()
	err = installation.Show(switchCtx, first, switchOpen)
	close(release)
	if err == nil {
		t.Error("a switch under the context of an ended plugin open ran during the user's close")
	}
	select {
	case isFlushed := <-closeResult:
		if !isFlushed {
			t.Error("the user's close did not complete")
		}
	case <-time.After(20 * time.Second):
		t.Fatal("the user's close never returned")
	}
	if got := installation.State.GetCurrentVault(); got.ID != second.ID {
		t.Errorf("the window shows %s, want %s", got.Name, second.Name)
	}
}
