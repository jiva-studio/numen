package editor_test

import (
	"context"
	"errors"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"connectrpc.com/connect"

	v1 "github.com/jiva-studio/numen/libs/protocol/gen/numen/v1"

	"github.com/jiva-studio/numen/libs/core/adapter/window/editor"
	"github.com/jiva-studio/numen/libs/core/internal/adapterlib/wire"
	vaults "github.com/jiva-studio/numen/libs/core/usecase/vault"
	"github.com/jiva-studio/numen/libs/notes/domain"

	"github.com/jiva-studio/numen/libs/core/port/pluginport"
)

// immediateBound is how long a call that must not wait for a page's FlushTimeout may take.
const immediateBound = editor.FlushTimeout / 2

// bindShow binds the change of vault of the state the way the desktop app does.
func (f *showing) bindShow() {
	f.installation.State.OpenVault = func(ctx context.Context, v domain.Vault) error {
		return f.installation.Show(ctx, v)
	}
}

// answerFlushes answers each flush of the page with NOTHING. Before it answers the first one, it sends
// on flushRequest and waits for release. It stops at the first answer that fails.
func (f *showing) answerFlushes(
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

// unsavedChangesText is what the unsaved-changes error of a plugin's open says while the user is asked.
const unsavedChangesText = "the user is asked about unsaved changes in the window; ask again after they answer"

// askingPage is a page that answers every flush with ASKING until the user answers its question,
// and then reports the request it was asking in as WRITTEN.
type askingPage struct {
	mu         sync.Mutex
	flushCount int
	lastToken  string
}

func (p *askingPage) getFlushCount() int {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.flushCount
}

// answerAskingUntilAnswered starts an askingPage on the stream; closing userAnswered is the user's
// answer to the page's question.
func (f *showing) answerAskingUntilAnswered(
	t *testing.T, stream *connect.ServerStreamForClient[v1.WatchQuitResponse], userAnswered <-chan struct{},
) *askingPage {
	page := &askingPage{}
	go func() {
		for stream.Receive() {
			if !stream.Msg().GetFlush() {
				continue
			}
			page.mu.Lock()
			page.flushCount++
			page.lastToken = stream.Msg().GetToken()
			page.mu.Unlock()
			result := v1.FlushResult_FLUSH_RESULT_ASKING
			select {
			case <-userAnswered:
				result = v1.FlushResult_FLUSH_RESULT_WRITTEN
			default:
			}
			if f.reportFlush(t, stream.Msg().GetToken(), result) != nil {
				return
			}
		}
	}()
	go func() {
		<-userAnswered
		page.mu.Lock()
		token := page.lastToken
		page.mu.Unlock()
		if f.reportFlush(t, token, v1.FlushResult_FLUSH_RESULT_WRITTEN) != nil {
			return
		}
	}()
	return page
}

// callOpenVault runs the plugin's open of the vault and returns its result.
func callOpenVault(ctx context.Context, list pluginport.PluginVaultList, id domain.VaultID) <-chan error {
	result := make(chan error, 1)
	go func() { result <- list.OpenVault(ctx, id) }()
	return result
}

// requireUnsavedChanges fails the test unless the plugin's open returns ErrUnsavedChanges with its
// text within immediateBound.
func requireUnsavedChanges(t *testing.T, openResult <-chan error, action string) {
	t.Helper()
	select {
	case err := <-openResult:
		if !errors.Is(err, pluginport.ErrUnsavedChanges) {
			t.Errorf("%s returned %v, want ErrUnsavedChanges", action, err)
		} else if !strings.Contains(err.Error(), unsavedChangesText) {
			t.Errorf("%s returned %q, want it to contain %q", action, err, unsavedChangesText)
		}
	case <-time.After(immediateBound):
		t.Fatalf("%s did not return within %v", action, immediateBound)
	}
}

func TestPluginOpenVaultWithUnsavedChangesInAPageAsksTheUserAndReturnsUnsavedChanges(t *testing.T) {
	f := openTwoVaults(t)
	f.bindShow()
	stream, isDone := f.watchQuit(t)
	defer isDone()
	userAnswered := make(chan struct{})
	answerUser := sync.OnceFunc(func() { close(userAnswered) })
	t.Cleanup(answerUser)
	page := f.answerAskingUntilAnswered(t, stream, userAnswered)
	list := editor.NewPluginVaultList(f.installation, f.cfg.CheckRoot())

	requireUnsavedChanges(t, callOpenVault(t.Context(), list, f.second.ID), "the plugin's open while a page holds unsaved changes")
	if got := f.installation.GetCurrentVault(); got.ID != f.first.ID {
		t.Errorf("the window shows %s after the plugin's open was rejected, want %s", got.Name, f.first.Name)
	}
	if !f.installation.State.Window.IsQuestionOpen() {
		t.Error("the plugin's open ended the flush request in which the page asks the user")
	}
	if got := page.getFlushCount(); got != 1 {
		t.Errorf("the page received %d flush requests, want 1", got)
	}

	requireUnsavedChanges(t, callOpenVault(t.Context(), list, f.second.ID), "a second plugin open while the user is asked")
	if got := page.getFlushCount(); got != 1 {
		t.Errorf("the page received %d flush requests after the second plugin open, want 1: the open sent a new one", got)
	}
	if got := f.installation.GetCurrentVault(); got.ID != f.first.ID {
		t.Errorf("the window shows %s after the second plugin open, want %s", got.Name, f.first.Name)
	}

	answerUser()
	waitCtx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
	defer cancel()
	if !f.installation.WaitForFlush(waitCtx) {
		t.Fatal("the user's answer never completed the flush request")
	}
	select {
	case err := <-callOpenVault(t.Context(), list, f.second.ID):
		if err != nil {
			t.Errorf("the plugin's open after the user answered returned %v, want nil", err)
		}
	case <-time.After(20 * time.Second):
		t.Fatal("the plugin's open after the user answered never returned")
	}
	if got := f.installation.GetCurrentVault(); got.ID != f.second.ID {
		t.Errorf("the window shows %s after the plugin's open, want %s", got.Name, f.second.Name)
	}
}

func TestPluginOpenVaultWithNoUnsavedChangesShowsTheVaultBeforeItReturns(t *testing.T) {
	f := openTwoVaults(t)
	f.bindShow()
	stream, isDone := f.watchQuit(t)
	defer isDone()
	f.answerAll(t, stream)
	list := editor.NewPluginVaultList(f.installation, f.cfg.CheckRoot())

	select {
	case err := <-callOpenVault(t.Context(), list, f.second.ID):
		if err != nil {
			t.Fatalf("the plugin's open of a window with no unsaved changes returned %v", err)
		}
	case <-time.After(20 * time.Second):
		t.Fatal("the plugin's open of a window with no unsaved changes never returned")
	}

	if got := f.installation.GetCurrentVault(); got.ID != f.second.ID {
		t.Errorf("the window shows %s when the plugin's open returned, want %s", got.Name, f.second.Name)
	}
	if got := f.state(t).GetName(); got != f.second.Name {
		t.Errorf("the window state names %q when the plugin's open returned, want %q", got, f.second.Name)
	}
	if f.installation.State.Window.IsFlushOpen() {
		t.Error("the plugin's open left a flush request open")
	}
}

// A close refused because a page asks the user waits for the user's answer (closeLikeTheApp). A
// plugin's open that starts in that wait returns ErrUnsavedChanges and does not end the user's flush.
func TestPluginOpenVaultDuringTheClosesQuestionReturnsUnsavedChangesAndKeepsTheCloseWaiting(t *testing.T) {
	f := openTwoVaults(t)
	f.bindShow()
	stream, isDone := f.watchQuit(t)
	defer isDone()
	userAnswered := make(chan struct{})
	f.answerAskingUntil(t, stream, userAnswered)

	ctx, cancel := context.WithTimeout(t.Context(), 20*time.Second)
	defer cancel()
	if f.installation.FlushPending(ctx) {
		t.Fatal("the close flushed although the page asks the user")
	}
	confirmed := make(chan bool, 1)
	go func() { confirmed <- f.installation.WaitForFlush(ctx) }()
	select {
	case isFlushed := <-confirmed:
		close(userAnswered)
		t.Fatalf("the close's wait ended with %v before the user answered", isFlushed)
	case <-time.After(200 * time.Millisecond):
	}

	list := editor.NewPluginVaultList(f.installation, f.cfg.CheckRoot())
	select {
	case err := <-callOpenVault(ctx, list, f.second.ID):
		if !errors.Is(err, pluginport.ErrUnsavedChanges) {
			t.Errorf("the plugin's open during the close's question returned %v, want ErrUnsavedChanges", err)
		}
	case <-time.After(immediateBound):
		close(userAnswered)
		t.Fatalf("the plugin's open during the close's question did not return within %v", immediateBound)
	}
	if !f.installation.State.Window.IsFlushOpen() {
		t.Error("the plugin's open ended the flush request of the user's close")
	}
	if got := f.installation.GetCurrentVault(); got.ID != f.first.ID {
		t.Errorf("the window shows %s after the plugin's open was rejected, want %s", got.Name, f.first.Name)
	}

	select {
	case isFlushed := <-confirmed:
		close(userAnswered)
		t.Fatalf("the plugin's open ended the close's wait for the user's answer with %v", isFlushed)
	case <-time.After(500 * time.Millisecond):
	}
	close(userAnswered)
	if !<-confirmed {
		t.Error("the close's wait ended unflushed after the user answered")
	}
}

func TestPluginOpenVaultDuringWindowCloseKeepsTheCloseAndIsRejected(t *testing.T) {
	f := openTwoVaults(t)
	f.bindShow()
	stream, isDone := f.watchQuit(t)
	defer isDone()
	flushRequest := make(chan struct{}, 1)
	release := make(chan struct{})
	f.answerFlushes(t, stream, flushRequest, release)

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
		t.Fatal("the close never asked the page to flush")
	}

	list := editor.NewPluginVaultList(f.installation, f.cfg.CheckRoot())
	openResult := callOpenVault(t.Context(), list, f.second.ID)

	select {
	case isFlushed := <-resolved:
		t.Errorf("the close ended with %v before the page answered: the plugin's open cancelled it", isFlushed)
		close(release)
	case <-time.After(2 * time.Second):
		close(release)
		if isFlushed := <-resolved; !isFlushed {
			t.Error("the close was cancelled by the plugin's open")
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
	if got := f.installation.GetCurrentVault(); got.ID != f.first.ID {
		t.Errorf("the window shows %s", got.Name)
	}
}

func TestTwoPluginOpenVaultCallsAtOnceDoNotReturnUnsavedChanges(t *testing.T) {
	f := openTwoVaults(t)
	f.bindShow()
	stream, isDone := f.watchQuit(t)
	defer isDone()
	flushRequest := make(chan struct{}, 1)
	release := make(chan struct{})
	f.answerFlushes(t, stream, flushRequest, release)
	list := editor.NewPluginVaultList(f.installation, f.cfg.CheckRoot())

	first := callOpenVault(t.Context(), list, f.second.ID)
	select {
	case <-flushRequest:
	case <-time.After(5 * time.Second):
		close(release)
		t.Fatal("the first open never asked the page to flush")
	}
	// The second open runs while the page has not answered the first one.
	second := callOpenVault(t.Context(), list, f.second.ID)
	select {
	case err := <-second:
		if errors.Is(err, pluginport.ErrUnsavedChanges) {
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
	if got := f.installation.GetCurrentVault(); got.ID != f.second.ID {
		t.Errorf("the window shows %s, want %s", got.Name, f.second.Name)
	}
}

func TestPluginOpenVaultOfMissingFolderReturnsUnreadable(t *testing.T) {
	f := openTwoVaults(t)
	f.bindShow()
	if err := os.RemoveAll(f.second.Path); err != nil {
		t.Fatal(err)
	}

	list := editor.NewPluginVaultList(f.installation, f.cfg.CheckRoot())
	err := list.OpenVault(t.Context(), f.second.ID)

	if !errors.Is(err, vaults.ErrUnreadable) {
		t.Errorf("the plugin's open of a vault whose folder is gone returned %v, want ErrUnreadable", err)
	}
	if got := f.installation.GetCurrentVault(); got.ID != f.first.ID {
		t.Errorf("the window is showing %s", got.Name)
	}
}

// watchFlushRequests returns a channel that receives once the page is asked to flush. The page never
// answers.
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

// startPluginOpen runs a plugin's open of the second vault and returns its result once the page has
// been asked to flush for it.
func startPluginOpen(t *testing.T, f *showing, flushRequest <-chan struct{}, release chan struct{}) <-chan error {
	t.Helper()
	list := editor.NewPluginVaultList(f.installation, f.cfg.CheckRoot())
	openResult := callOpenVault(context.WithoutCancel(t.Context()), list, f.second.ID)
	select {
	case <-flushRequest:
	case <-time.After(5 * time.Second):
		if release != nil {
			close(release)
		}
		t.Fatal("the plugin's open never asked the page to flush")
	}
	return openResult
}

// expectOpenUnavailable checks that the plugin's open gave way to the user's action.
func expectOpenUnavailable(t *testing.T, openResult <-chan error) {
	t.Helper()
	select {
	case err := <-openResult:
		if !errors.Is(err, pluginport.ErrUnavailable) {
			t.Errorf("the plugin's open returned %v, want ErrUnavailable", err)
		}
	case <-time.After(20 * time.Second):
		t.Fatal("the plugin's open never returned")
	}
}

// closeLikeTheApp is the close of apps/desktop/app: a rejected flush waits for the running flush and
// closes again once every page is written.
func closeLikeTheApp(ctx context.Context, o *editor.Installation) bool {
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
	f.bindShow()
	stream, isDone := f.watchQuit(t)
	defer isDone()
	openResult := startPluginOpen(t, f, watchFlushRequests(stream), nil)

	ctx, cancel := context.WithTimeout(t.Context(), 20*time.Second)
	defer cancel()
	if !closeLikeTheApp(ctx, f.installation) {
		t.Error("the user's close during a plugin's open was rejected: the window stays open")
	}
	expectOpenUnavailable(t, openResult)
}

func TestAQuitDuringAPluginOpenIsAccepted(t *testing.T) {
	f := openTwoVaults(t)
	f.bindShow()
	stream, isDone := f.watchQuit(t)
	defer isDone()
	flushRequest := make(chan struct{}, 1)
	release := make(chan struct{})
	f.answerFlushes(t, stream, flushRequest, release)
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
	f.bindShow()
	stream, isDone := f.watchQuit(t)
	defer isDone()
	flushRequest := make(chan struct{}, 1)
	release := make(chan struct{})
	f.answerFlushes(t, stream, flushRequest, release)
	openResult := startPluginOpen(t, f, flushRequest, release)

	switched := make(chan error, 1)
	go func() { switched <- f.installation.State.OpenVault(t.Context(), f.second) }()
	select {
	case err := <-switched:
		close(release)
		if err != nil {
			t.Errorf("the user's vault switch was rejected during a plugin's open: %v", err)
		}
	case <-time.After(2 * time.Second):
		close(release)
		if err := <-switched; err != nil {
			t.Errorf("the user's vault switch was rejected during a plugin's open: %v", err)
		}
	}
	expectOpenUnavailable(t, openResult)
	if got := f.installation.GetCurrentVault(); got.ID != f.second.ID {
		t.Errorf("the window shows %s", got.Name)
	}
}

func TestPluginOpenVaultDuringUserSwitchIsUnavailable(t *testing.T) {
	f := openTwoVaults(t)
	f.bindShow()
	stream, isDone := f.watchQuit(t)
	defer isDone()
	flushRequest := make(chan struct{}, 1)
	release := make(chan struct{})
	f.answerFlushes(t, stream, flushRequest, release)
	switched := make(chan error, 1)
	go func() { switched <- f.installation.Show(t.Context(), f.second) }()
	select {
	case <-flushRequest:
	case <-time.After(5 * time.Second):
		close(release)
		t.Fatal("the switch never asked the page to flush")
	}
	list := editor.NewPluginVaultList(f.installation, f.cfg.CheckRoot())
	err := list.OpenVault(t.Context(), f.first.ID)
	close(release)

	if !errors.Is(err, pluginport.ErrUnavailable) {
		t.Errorf("the plugin's open during the user's switch returned %v, want ErrUnavailable", err)
	}
	if err := <-switched; err != nil {
		t.Errorf("the user's switch returned %v", err)
	}
	if got := f.installation.GetCurrentVault(); got.ID != f.second.ID {
		t.Errorf("the window shows %s", got.Name)
	}
}

// reportFlush answers one flush of the page.
func (f *showing) reportFlush(t *testing.T, token string, result v1.FlushResult) error {
	ctx, cancel := context.WithTimeout(context.WithoutCancel(t.Context()), 5*time.Second)
	defer cancel()
	_, err := f.canvas.ReportFlush(ctx, connect.NewRequest(&v1.ReportFlushRequest{
		Window: wire.Editor, Token: token, Result: result,
	}))
	return err
}

// answerAskingUntil answers every flush with ASKING until userAnswered is closed, then answers the last
// token with NOTHING, as a page whose user has answered its question.
func (f *showing) answerAskingUntil(
	t *testing.T, stream *connect.ServerStreamForClient[v1.WatchQuitResponse], userAnswered <-chan struct{},
) {
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
			case <-userAnswered:
				if f.reportFlush(t, stream.Msg().GetToken(), v1.FlushResult_FLUSH_RESULT_NOTHING) != nil {
					return
				}
			default:
				if f.reportFlush(t, stream.Msg().GetToken(), v1.FlushResult_FLUSH_RESULT_ASKING) != nil {
					return
				}
			}
		}
	}()
	go func() {
		<-userAnswered
		mu.Lock()
		token := last
		mu.Unlock()
		if f.reportFlush(t, token, v1.FlushResult_FLUSH_RESULT_NOTHING) != nil {
			return
		}
	}()
}

// answerAll answers every flush of the page with NOTHING at once.
func (f *showing) answerAll(t *testing.T, stream *connect.ServerStreamForClient[v1.WatchQuitResponse]) {
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

// The user switches vault again and again while plugins keep opening vaults: every switch of the user
// is taken, none waits past its bound, and no plugin open is left holding the window.
func TestUserSwitchesAmidRepeatedPluginOpensAreAllTaken(t *testing.T) {
	f := openTwoVaults(t)
	f.bindShow()
	stream, isDone := f.watchQuit(t)
	defer isDone()
	f.answerAll(t, stream)
	list := editor.NewPluginVaultList(f.installation, f.cfg.CheckRoot())

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
				case errors.Is(err, pluginport.ErrUnavailable):
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
		go func() { switchResult <- f.installation.State.OpenVault(t.Context(), target) }()
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
		t.Errorf("a plugin open after the switches returned %v: the exclusion is still held", err)
	}
}

// Two user actions, a close and a vault switch, arrive while a plugin opens a vault. Neither hangs,
// one of them is taken, and the plugin's open gives way.
func TestTwoUserActionsDuringPluginOpenBothReturn(t *testing.T) {
	f := openTwoVaults(t)
	f.bindShow()
	stream, isDone := f.watchQuit(t)
	defer isDone()
	flushRequest := make(chan struct{}, 1)
	release := make(chan struct{})
	f.answerFlushes(t, stream, flushRequest, release)
	openResult := startPluginOpen(t, f, flushRequest, release)

	ctx, cancel := context.WithTimeout(t.Context(), 20*time.Second)
	defer cancel()
	switched := make(chan error, 1)
	closed := make(chan bool, 1)
	go func() { switched <- f.installation.State.OpenVault(ctx, f.second) }()
	go func() { closed <- closeLikeTheApp(ctx, f.installation) }()
	select {
	case err := <-openResult:
		close(release)
		if !errors.Is(err, pluginport.ErrUnavailable) {
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
		case switchErr = <-switched:
		case isClosed = <-closed:
		case <-time.After(20 * time.Second):
			t.Fatal("a user's action never returned")
		}
	}
	t.Logf("switch: %v, close: %v", switchErr, isClosed)
	if switchErr != nil && !isClosed {
		t.Error("neither the user's switch nor the user's close was taken")
	}
}

// A plugin call that ends during its open (the host endpoint closes, the plugin hangs up) returns
// unavailable and does not change the vault of the window.
func TestAPluginCallThatEndsDuringItsOpenIsUnavailable(t *testing.T) {
	f := openTwoVaults(t)
	f.bindShow()
	stream, isDone := f.watchQuit(t)
	defer isDone()
	flushRequest := watchFlushRequests(stream)
	list := editor.NewPluginVaultList(f.installation, f.cfg.CheckRoot())

	ctx, hangUp := context.WithCancel(t.Context())
	openResult := callOpenVault(ctx, list, f.second.ID)
	select {
	case <-flushRequest:
	case <-time.After(5 * time.Second):
		t.Fatal("the plugin's open never asked the page to flush")
	}
	hangUp()
	select {
	case err := <-openResult:
		if !errors.Is(err, pluginport.ErrUnavailable) {
			t.Errorf("the plugin's open whose call ended returned %v, want ErrUnavailable", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("the plugin's open whose call ended never returned")
	}
	if got := f.installation.GetCurrentVault(); got.ID != f.first.ID {
		t.Errorf("the window shows %s after a plugin call that ended", got.Name)
	}
}
