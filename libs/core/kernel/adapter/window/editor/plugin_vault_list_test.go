package editor_test

import (
	"context"
	"errors"
	vaultapi "github.com/jiva-studio/numen/libs/core/kernel/adapter/plugin/hostendpoint/vault"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"connectrpc.com/connect"

	v1 "github.com/jiva-studio/numen/libs/core/protocol/gen/numen/v1"

	"github.com/jiva-studio/numen/libs/core/kernel/adapter/window/editor"
	vaultdomain "github.com/jiva-studio/numen/libs/core/kernel/domain/vault"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapterlib/wire"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/windowsession"
	"github.com/jiva-studio/numen/libs/note/domain"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/plugin"
)

// immediateBound is how long a call that must not wait for a page's FlushTimeout may take.
const immediateBound = windowsession.FlushTimeout / 2

func newPluginVaultList(f *showing) vaultapi.List {
	scenarios, listing := f.cases.Vaults, f.cases.Listing
	return windowsession.NewPluginVaults(&f.installation.State.Runtime, windowsession.VaultOps{
		Edit:    windowsession.VaultEdits{Add: scenarios.Change.Add, Rename: scenarios.Change.Rename, Remove: scenarios.Change.Remove},
		Find:    listing.Find,
		Missing: listing.FolderCheck,
		Open:    f.installation,
	})
}

func (f *showing) bindVaultSwitch() {
	f.installation.State.OpenVault = func(ctx context.Context, v domain.Vault, open *windowsession.PluginOpen) error {
		return f.installation.Show(ctx, v, open)
	}
}

func (f *showing) respondToFlushes(
	t *testing.T, stream *connect.ServerStreamForClient[v1.WatchQuitResponse], flushRequest chan<- struct{}, release <-chan struct{},
) {
	t.Helper()
	go func() {
		isFirst := true
		for stream.Receive() {
			if !stream.Msg().GetFlush() {
				continue
			}
			if isFirst {
				isFirst = false
				flushRequest <- struct{}{}
				<-release
			}
			if f.reportFlush(t, stream.Msg().GetToken(), v1.FlushResult_FLUSH_RESULT_NOTHING) != nil {
				// The test may have ended, so the error is not logged; the calls report what it caused.
				return
			}
		}
	}()
}

// unsavedChangesText is the text of the unsaved-changes error of a plugin's open while the window prompts the user.
const unsavedChangesText = "the window has unsaved changes and is prompting the user; retry after the prompt closes"

// promptingPage is a page that reports UNSAVED for every flush until the user responds to its prompt,
// and then reports the last flush request as WRITTEN.
type promptingPage struct {
	mu         sync.Mutex
	flushCount int
	lastToken  string
}

func (p *promptingPage) getFlushCount() int {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.flushCount
}

func (f *showing) startPromptingPage(
	t *testing.T, stream *connect.ServerStreamForClient[v1.WatchQuitResponse], userResponded <-chan struct{},
) *promptingPage {
	t.Helper()
	page := &promptingPage{}
	go func() {
		for stream.Receive() {
			if !stream.Msg().GetFlush() {
				continue
			}
			page.mu.Lock()
			page.flushCount++
			page.lastToken = stream.Msg().GetToken()
			page.mu.Unlock()
			result := v1.FlushResult_FLUSH_RESULT_UNSAVED
			select {
			case <-userResponded:
				result = v1.FlushResult_FLUSH_RESULT_WRITTEN
			default:
			}
			if f.reportFlush(t, stream.Msg().GetToken(), result) != nil {
				return
			}
		}
	}()
	go func() {
		<-userResponded
		page.mu.Lock()
		token := page.lastToken
		page.mu.Unlock()
		if f.reportFlush(t, token, v1.FlushResult_FLUSH_RESULT_WRITTEN) != nil {
			return
		}
	}()
	return page
}

func callOpenVault(ctx context.Context, list vaultapi.List, id domain.VaultID) <-chan error {
	result := make(chan error, 1)
	go func() { result <- list.OpenVault(ctx, id) }()
	return result
}

func requireUnsavedChanges(t *testing.T, openResult <-chan error, action string) {
	t.Helper()
	select {
	case err := <-openResult:
		if !errors.Is(err, plugin.ErrUnsavedChanges) {
			t.Errorf("%s returned %v, want ErrUnsavedChanges", action, err)
		} else if !strings.Contains(err.Error(), unsavedChangesText) {
			t.Errorf("%s returned %q, want it to contain %q", action, err, unsavedChangesText)
		}
	case <-time.After(immediateBound):
		t.Fatalf("%s did not return within %v", action, immediateBound)
	}
}

func TestPluginOpenVaultWithUnsavedChangesPromptsTheUser(t *testing.T) {
	f := openTwoVaults(t)
	f.bindVaultSwitch()
	stream, isDone := f.watchQuit(t)
	defer isDone()
	userResponded := make(chan struct{})
	respondAsUser := sync.OnceFunc(func() { close(userResponded) })
	t.Cleanup(respondAsUser)
	page := f.startPromptingPage(t, stream, userResponded)
	list := newPluginVaultList(f)

	requireUnsavedChanges(t, callOpenVault(t.Context(), list, f.second.ID), "the plugin's open while a page holds unsaved changes")
	if got := f.installation.State.GetCurrentVault(); got.ID != f.first.ID {
		t.Errorf("the window shows %s after the plugin's open was rejected, want %s", got.Name, f.first.Name)
	}
	if !f.installation.State.Window.IsQuestionOpen() {
		t.Error("the plugin's open ended the flush request in which the page prompts the user")
	}
	if got := page.getFlushCount(); got != 1 {
		t.Errorf("the page received %d flush requests, want 1", got)
	}

	requireUnsavedChanges(t, callOpenVault(t.Context(), list, f.second.ID), "a second plugin open during the prompt")
	if got := page.getFlushCount(); got != 1 {
		t.Errorf("the page received %d flush requests after the second plugin open, want 1: the open sent a new one", got)
	}
	if got := f.installation.State.GetCurrentVault(); got.ID != f.first.ID {
		t.Errorf("the window shows %s after the second plugin open, want %s", got.Name, f.first.Name)
	}

	respondAsUser()
	waitCtx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
	defer cancel()
	if !f.installation.WaitForFlush(waitCtx) {
		t.Fatal("closing the prompt never completed the flush request")
	}
	select {
	case err := <-callOpenVault(t.Context(), list, f.second.ID):
		if err != nil {
			t.Errorf("the plugin's open after the prompt closed returned %v, want nil", err)
		}
	case <-time.After(20 * time.Second):
		t.Fatal("the plugin's open after the prompt closed never returned")
	}
	if got := f.installation.State.GetCurrentVault(); got.ID != f.second.ID {
		t.Errorf("the window shows %s after the plugin's open, want %s", got.Name, f.second.Name)
	}
}

func TestPluginOpenVaultWithoutUnsavedChangesShowsVaultFirst(t *testing.T) {
	f := openTwoVaults(t)
	f.bindVaultSwitch()
	stream, isDone := f.watchQuit(t)
	defer isDone()
	f.respondToAllFlushes(t, stream)
	list := newPluginVaultList(f)

	select {
	case err := <-callOpenVault(t.Context(), list, f.second.ID):
		if err != nil {
			t.Fatalf("the plugin's open of a window with no unsaved changes returned %v", err)
		}
	case <-time.After(20 * time.Second):
		t.Fatal("the plugin's open of a window with no unsaved changes never returned")
	}

	if got := f.installation.State.GetCurrentVault(); got.ID != f.second.ID {
		t.Errorf("the window shows %s when the plugin's open returned, want %s", got.Name, f.second.Name)
	}
	if got := f.state(t).GetName(); got != f.second.Name {
		t.Errorf("the window state names %q when the plugin's open returned, want %q", got, f.second.Name)
	}
	if f.isFlushRequestedOfANewPage(t, 500*time.Millisecond) {
		t.Error("the plugin's open left a flush request open")
	}
}

// A close rejected because a page prompts the user waits until the prompt closes (closeAsDesktopApp). A
// plugin's open that starts in that wait returns ErrUnsavedChanges and does not end the user's flush.
func TestPluginOpenDuringClosePromptFailsAndCloseKeepsWaiting(t *testing.T) {
	f := openTwoVaults(t)
	f.bindVaultSwitch()
	stream, isDone := f.watchQuit(t)
	defer isDone()
	userResponded := make(chan struct{})
	f.respondPromptUntil(t, stream, userResponded)

	ctx, cancel := context.WithTimeout(t.Context(), 20*time.Second)
	defer cancel()
	if f.installation.FlushPending(ctx) {
		t.Fatal("the close flushed although the page prompts the user")
	}
	confirmCh := make(chan bool, 1)
	go func() { confirmCh <- f.installation.WaitForFlush(ctx) }()
	select {
	case isFlushed := <-confirmCh:
		close(userResponded)
		t.Fatalf("the close's wait ended with %v before the prompt closed", isFlushed)
	case <-time.After(200 * time.Millisecond):
	}

	list := newPluginVaultList(f)
	select {
	case err := <-callOpenVault(ctx, list, f.second.ID):
		if !errors.Is(err, plugin.ErrUnsavedChanges) {
			t.Errorf("the plugin's open during the close prompt returned %v, want ErrUnsavedChanges", err)
		}
	case <-time.After(immediateBound):
		close(userResponded)
		t.Fatalf("the plugin's open during the close prompt did not return within %v", immediateBound)
	}
	if !f.isFlushRequestedOfANewPage(t, 5*time.Second) {
		t.Error("the plugin's open ended the flush request of the user's close")
	}
	if got := f.installation.State.GetCurrentVault(); got.ID != f.first.ID {
		t.Errorf("the window shows %s after the plugin's open was rejected, want %s", got.Name, f.first.Name)
	}

	select {
	case isFlushed := <-confirmCh:
		close(userResponded)
		t.Fatalf("the plugin's open ended the close's wait for the prompt with %v", isFlushed)
	case <-time.After(500 * time.Millisecond):
	}
	close(userResponded)
	if !<-confirmCh {
		t.Error("the close's wait ended unflushed after the prompt closed")
	}
}

func TestPluginOpenVaultDuringWindowCloseIsRejected(t *testing.T) {
	f := openTwoVaults(t)
	f.bindVaultSwitch()
	stream, isDone := f.watchQuit(t)
	defer isDone()
	flushRequest := make(chan struct{}, 1)
	release := make(chan struct{})
	f.respondToFlushes(t, stream, flushRequest, release)

	resolved := make(chan bool, 1)
	go func() {
		ctx, cancel := context.WithTimeout(t.Context(), 20*time.Second)
		defer cancel()
		resolved <- f.installation.FlushPending(ctx)
	}()
	select {
	case <-flushRequest:
	case <-time.After(5 * time.Second):
		close(release)
		t.Fatal("the close never requested a flush from the page")
	}

	list := newPluginVaultList(f)
	openResult := callOpenVault(t.Context(), list, f.second.ID)

	select {
	case isFlushed := <-resolved:
		t.Errorf("the close ended with %v before the page reported: the plugin's open canceled it", isFlushed)
		close(release)
	case <-time.After(2 * time.Second):
		close(release)
		if isFlushed := <-resolved; !isFlushed {
			t.Error("the close was canceled by the plugin's open")
		}
	}
	select {
	case err := <-openResult:
		if err == nil {
			t.Error("the plugin's open succeeded while the window was closing")
		}
	case <-time.After(20 * time.Second):
		t.Fatal("the plugin's open never returned")
	}
	if got := f.installation.State.GetCurrentVault(); got.ID != f.first.ID {
		t.Errorf("the window shows %s", got.Name)
	}
}

func TestTwoPluginOpenVaultCallsAtOnceDoNotReturnUnsavedChanges(t *testing.T) {
	f := openTwoVaults(t)
	f.bindVaultSwitch()
	stream, isDone := f.watchQuit(t)
	defer isDone()
	flushRequest := make(chan struct{}, 1)
	release := make(chan struct{})
	f.respondToFlushes(t, stream, flushRequest, release)
	list := newPluginVaultList(f)

	first := callOpenVault(t.Context(), list, f.second.ID)
	select {
	case <-flushRequest:
	case <-time.After(5 * time.Second):
		close(release)
		t.Fatal("the first open never requested a flush from the page")
	}
	// The second open runs before the page has reported the first one.
	second := callOpenVault(t.Context(), list, f.second.ID)
	select {
	case err := <-second:
		if errors.Is(err, plugin.ErrUnsavedChanges) {
			t.Errorf("the second open returned ErrUnsavedChanges while the first one waited for a page with nothing to save: %v", err)
		}
		close(release)
	case <-time.After(2 * time.Second):
		close(release)
		if err := <-second; err != nil {
			t.Errorf("the second open returned %v", err)
		}
	}
	if err := <-first; err != nil {
		t.Errorf("the first open returned %v", err)
	}
	if got := f.installation.State.GetCurrentVault(); got.ID != f.second.ID {
		t.Errorf("the window shows %s, want %s", got.Name, f.second.Name)
	}
}

func TestPluginOpenVaultOfMissingFolderReturnsUnreadable(t *testing.T) {
	f := openTwoVaults(t)
	f.bindVaultSwitch()
	if err := os.RemoveAll(f.second.Path); err != nil {
		t.Fatal(err)
	}

	list := newPluginVaultList(f)
	err := list.OpenVault(t.Context(), f.second.ID)

	if !errors.Is(err, vaultdomain.ErrUnreadable) {
		t.Errorf("the plugin's open of a vault whose folder is gone returned %v, want ErrUnreadable", err)
	}
	if got := f.installation.State.GetCurrentVault(); got.ID != f.first.ID {
		t.Errorf("the window is showing %s", got.Name)
	}
}

func watchFlushRequests(stream *connect.ServerStreamForClient[v1.WatchQuitResponse]) <-chan struct{} {
	flushRequest := make(chan struct{}, 1)
	go func() {
		for stream.Receive() {
			if stream.Msg().GetFlush() {
				select {
				case flushRequest <- struct{}{}:
				default:
				}
			}
		}
	}()
	return flushRequest
}

func startPluginOpen(t *testing.T, f *showing, flushRequest <-chan struct{}, release chan struct{}) <-chan error {
	t.Helper()
	list := newPluginVaultList(f)
	openResult := callOpenVault(context.WithoutCancel(t.Context()), list, f.second.ID)
	select {
	case <-flushRequest:
	case <-time.After(5 * time.Second):
		if release != nil {
			close(release)
		}
		t.Fatal("the plugin's open never requested a flush from the page")
	}
	return openResult
}

func expectOpenUnavailable(t *testing.T, openResult <-chan error) {
	t.Helper()
	select {
	case err := <-openResult:
		if !errors.Is(err, domain.ErrUnavailable) {
			t.Errorf("the plugin's open returned %v, want ErrUnavailable", err)
		}
	case <-time.After(20 * time.Second):
		t.Fatal("the plugin's open never returned")
	}
}

func closeAsDesktopApp(ctx context.Context, o *editor.Installation) bool {
	if o.FlushPending(ctx) {
		return true
	}
	if o.WaitForFlush(ctx) {
		return o.FlushPending(ctx)
	}
	return false
}

func TestACloseDuringAPluginOpenWithASilentPageClosesTheWindow(t *testing.T) {
	f := openTwoVaults(t)
	f.bindVaultSwitch()
	stream, isDone := f.watchQuit(t)
	defer isDone()
	openResult := startPluginOpen(t, f, watchFlushRequests(stream), nil)

	ctx, cancel := context.WithTimeout(t.Context(), 20*time.Second)
	defer cancel()
	if !closeAsDesktopApp(ctx, f.installation) {
		t.Error("the user's close during a plugin's open was rejected: the window stays open")
	}
	expectOpenUnavailable(t, openResult)
}

func TestAQuitDuringAPluginOpenIsAccepted(t *testing.T) {
	f := openTwoVaults(t)
	f.bindVaultSwitch()
	stream, isDone := f.watchQuit(t)
	defer isDone()
	flushRequest := make(chan struct{}, 1)
	release := make(chan struct{})
	f.respondToFlushes(t, stream, flushRequest, release)
	openResult := startPluginOpen(t, f, flushRequest, release)

	resolved := make(chan bool, 1)
	go func() {
		ctx, cancel := context.WithTimeout(t.Context(), 20*time.Second)
		defer cancel()
		resolved <- f.installation.FlushPending(ctx)
	}()
	select {
	case isFlushed := <-resolved:
		close(release)
		if !isFlushed {
			t.Error("the user's quit was rejected during a plugin's open")
		}
	case <-time.After(2 * time.Second):
		close(release)
		if !<-resolved {
			t.Error("the user's quit was rejected during a plugin's open")
		}
	}
	expectOpenUnavailable(t, openResult)
}

func TestUserVaultSwitchDuringPluginOpenIsAccepted(t *testing.T) {
	f := openTwoVaults(t)
	f.bindVaultSwitch()
	stream, isDone := f.watchQuit(t)
	defer isDone()
	flushRequest := make(chan struct{}, 1)
	release := make(chan struct{})
	f.respondToFlushes(t, stream, flushRequest, release)
	openResult := startPluginOpen(t, f, flushRequest, release)

	switchCh := make(chan error, 1)
	go func() { switchCh <- f.installation.State.OpenVault(t.Context(), f.second, nil) }()
	select {
	case err := <-switchCh:
		close(release)
		if err != nil {
			t.Errorf("the user's vault switch was rejected during a plugin's open: %v", err)
		}
	case <-time.After(2 * time.Second):
		close(release)
		if err := <-switchCh; err != nil {
			t.Errorf("the user's vault switch was rejected during a plugin's open: %v", err)
		}
	}
	expectOpenUnavailable(t, openResult)
	if got := f.installation.State.GetCurrentVault(); got.ID != f.second.ID {
		t.Errorf("the window shows %s", got.Name)
	}
}

func TestPluginOpenVaultDuringUserSwitchIsUnavailable(t *testing.T) {
	f := openTwoVaults(t)
	f.bindVaultSwitch()
	stream, isDone := f.watchQuit(t)
	defer isDone()
	flushRequest := make(chan struct{}, 1)
	release := make(chan struct{})
	f.respondToFlushes(t, stream, flushRequest, release)
	switchCh := make(chan error, 1)
	go func() { switchCh <- f.installation.Show(t.Context(), f.second, nil) }()
	select {
	case <-flushRequest:
	case <-time.After(5 * time.Second):
		close(release)
		t.Fatal("the switch never requested a flush from the page")
	}
	list := newPluginVaultList(f)
	err := list.OpenVault(t.Context(), f.first.ID)
	close(release)

	if !errors.Is(err, domain.ErrUnavailable) {
		t.Errorf("the plugin's open during the user's switch returned %v, want ErrUnavailable", err)
	}
	if err := <-switchCh; err != nil {
		t.Errorf("the user's switch returned %v", err)
	}
	if got := f.installation.State.GetCurrentVault(); got.ID != f.second.ID {
		t.Errorf("the window shows %s", got.Name)
	}
}

func (f *showing) reportFlush(t *testing.T, token string, result v1.FlushResult) error {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.WithoutCancel(t.Context()), 5*time.Second)
	defer cancel()
	_, err := f.canvas.ReportFlush(ctx, connect.NewRequest(&v1.ReportFlushRequest{
		Window: wire.Editor, Token: token, Result: result,
	}))
	return err
}

func (f *showing) respondPromptUntil(
	t *testing.T, stream *connect.ServerStreamForClient[v1.WatchQuitResponse], userResponded <-chan struct{},
) {
	t.Helper()
	var mu sync.Mutex
	last := ""
	go func() {
		for stream.Receive() {
			if !stream.Msg().GetFlush() {
				continue
			}
			mu.Lock()
			last = stream.Msg().GetToken()
			mu.Unlock()
			select {
			case <-userResponded:
				if f.reportFlush(t, stream.Msg().GetToken(), v1.FlushResult_FLUSH_RESULT_NOTHING) != nil {
					return
				}
			default:
				if f.reportFlush(t, stream.Msg().GetToken(), v1.FlushResult_FLUSH_RESULT_UNSAVED) != nil {
					return
				}
			}
		}
	}()
	go func() {
		<-userResponded
		mu.Lock()
		token := last
		mu.Unlock()
		if f.reportFlush(t, token, v1.FlushResult_FLUSH_RESULT_NOTHING) != nil {
			return
		}
	}()
}

func (f *showing) respondToAllFlushes(t *testing.T, stream *connect.ServerStreamForClient[v1.WatchQuitResponse]) {
	t.Helper()
	go func() {
		for stream.Receive() {
			if !stream.Msg().GetFlush() {
				continue
			}
			if f.reportFlush(t, stream.Msg().GetToken(), v1.FlushResult_FLUSH_RESULT_NOTHING) != nil {
				return
			}
		}
	}()
}

// The user switches vault again and again while plugins keep opening vaults: every user switch
// completes within its bound and no plugin open blocks the window.
func TestUserVaultSwitchesSucceedDuringRepeatedPluginOpens(t *testing.T) {
	f := openTwoVaults(t)
	f.bindVaultSwitch()
	stream, isDone := f.watchQuit(t)
	defer isDone()
	f.respondToAllFlushes(t, stream)
	list := newPluginVaultList(f)

	stop := make(chan struct{})
	var openers sync.WaitGroup
	var unexpected atomic.Value
	var succeeded, unavailable atomic.Int64
	for range 4 {
		openers.Go(func() {
			for {
				select {
				case <-stop:
					return
				default:
				}
				err := list.OpenVault(context.WithoutCancel(t.Context()), f.second.ID)
				switch {
				case err == nil:
					succeeded.Add(1)
				case errors.Is(err, domain.ErrUnavailable):
					unavailable.Add(1)
				default:
					unexpected.Store(err.Error())
				}
			}
		})
	}
	for i := range 6 {
		target := f.second
		if i%2 == 1 {
			target = f.first
		}
		switchResult := make(chan error, 1)
		go func() { switchResult <- f.installation.State.OpenVault(t.Context(), target, nil) }()
		select {
		case err := <-switchResult:
			if err != nil {
				t.Errorf("switch %d of the user was rejected amid plugin opens: %v", i, err)
			}
		case <-time.After(15 * time.Second):
			close(stop)
			t.Fatalf("switch %d of the user never returned amid plugin opens", i)
		}
	}
	close(stop)
	openers.Wait()
	t.Logf("plugin opens succeeded %d, unavailable %d", succeeded.Load(), unavailable.Load())
	if v := unexpected.Load(); v != nil {
		t.Errorf("a plugin open returned other than ErrUnavailable or nil: %v", v)
	}
	if err := list.OpenVault(t.Context(), f.second.ID); err != nil {
		t.Errorf("a plugin open after the switches returned %v: the exclusion is still locked", err)
	}
}

// Two user actions, a close and a vault switch, arrive while a plugin opens a vault. Neither hangs,
// one succeeds and the plugin open is canceled.
func TestTwoUserActionsDuringPluginOpenBothReturn(t *testing.T) {
	f := openTwoVaults(t)
	f.bindVaultSwitch()
	stream, isDone := f.watchQuit(t)
	defer isDone()
	flushRequest := make(chan struct{}, 1)
	release := make(chan struct{})
	f.respondToFlushes(t, stream, flushRequest, release)
	openResult := startPluginOpen(t, f, flushRequest, release)

	ctx, cancel := context.WithTimeout(t.Context(), 20*time.Second)
	defer cancel()
	switchCh := make(chan error, 1)
	closed := make(chan bool, 1)
	go func() { switchCh <- f.installation.State.OpenVault(ctx, f.second, nil) }()
	go func() { closed <- closeAsDesktopApp(ctx, f.installation) }()
	select {
	case err := <-openResult:
		close(release)
		if !errors.Is(err, domain.ErrUnavailable) {
			t.Errorf("the plugin's open returned %v, want ErrUnavailable", err)
		}
	case <-time.After(time.Second):
		close(release)
		expectOpenUnavailable(t, openResult)
	}
	var switchErr error
	var isClosed bool
	for range 2 {
		select {
		case switchErr = <-switchCh:
		case isClosed = <-closed:
		case <-time.After(20 * time.Second):
			t.Fatal("a user's action never returned")
		}
	}
	t.Logf("switch: %v, close: %v", switchErr, isClosed)
	if switchErr != nil && !isClosed {
		t.Error("neither the user's switch nor the user's close succeeded")
	}
}

// A plugin call that ends during its open (the host endpoint closes, the plugin hangs up) returns
// unavailable and does not change the vault of the window.
func TestAPluginCallThatEndsDuringItsOpenIsUnavailable(t *testing.T) {
	f := openTwoVaults(t)
	f.bindVaultSwitch()
	stream, isDone := f.watchQuit(t)
	defer isDone()
	flushRequest := watchFlushRequests(stream)
	list := newPluginVaultList(f)

	ctx, hangUp := context.WithCancel(t.Context())
	openResult := callOpenVault(ctx, list, f.second.ID)
	select {
	case <-flushRequest:
	case <-time.After(5 * time.Second):
		t.Fatal("the plugin's open never requested a flush from the page")
	}
	hangUp()
	select {
	case err := <-openResult:
		if !errors.Is(err, domain.ErrUnavailable) {
			t.Errorf("the plugin's open whose call ended returned %v, want ErrUnavailable", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("the plugin's open whose call ended never returned")
	}
	if got := f.installation.State.GetCurrentVault(); got.ID != f.first.ID {
		t.Errorf("the window shows %s after a plugin call that ended", got.Name)
	}
}

func (f *showing) isFlushRequestedOfANewPage(t *testing.T, wait time.Duration) bool {
	t.Helper()
	stream, isDone := f.watchQuit(t)
	defer isDone()
	tokens := make(chan string, 1)
	go func() {
		for stream.Receive() {
			if stream.Msg().GetFlush() {
				tokens <- stream.Msg().GetToken()
				return
			}
		}
	}()
	select {
	case token := <-tokens:
		if err := f.reportFlush(t, token, v1.FlushResult_FLUSH_RESULT_NOTHING); err != nil {
			t.Fatalf("report the flush of the new page: %v", err)
		}
		return true
	case <-time.After(wait):
		return false
	}
}
