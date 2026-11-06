package editor_test

import (
	"context"
	"errors"
	"os"
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

// bindShow binds the change of vault of the state the way the desktop app does.
func (f *showing) bindShow() {
	f.installation.State.OpenVault = func(ctx context.Context, v domain.Vault) error {
		return f.installation.Show(ctx, v)
	}
}

// answerFlushes answers each flush the page is flushRequest for with nothing held, after the first one waits
// for release. It sends on flushRequest when the first flush arrives, and stops at the first answer that
// fails.
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
			ctx, cancel := context.WithTimeout(context.WithoutCancel(t.Context()), 5*time.Second)
			_, err := f.canvas.ReportFlush(ctx, connect.NewRequest(&v1.ReportFlushRequest{
				Window: wire.Editor, Token: stream.Msg().GetToken(), Result: v1.FlushResult_FLUSH_RESULT_NOTHING,
			}))
			cancel()
			if err != nil {
				// The test may have ended, so the error is not logged; the decisions report what it caused.
				return
			}
		}
	}()
}

func TestPluginOpenVaultDuringWindowCloseKeepsTheCloseAndRejectsOpen(t *testing.T) {
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
		t.Fatal("the close never flushRequest the page for what it holds")
	}

	list := editor.NewPluginVaultList(f.installation, f.cfg.CheckRoot())
	decided := make(chan error, 1)
	go func() {
		_, err := list.OpenVault(t.Context(), f.second.ID)
		decided <- err
	}()

	select {
	case isFlushed := <-resolved:
		t.Errorf("the close settled with %v before the page answered: the plugin's decision called it off", isFlushed)
		close(release)
	case <-time.After(2 * time.Second):
		close(release)
		if isFlushed := <-resolved; !isFlushed {
			t.Error("the close was called off by the plugin's decision to open a vault")
		}
	}
	select {
	case err := <-decided:
		if err == nil {
			t.Error("the plugin's decision accepted the open while the window was closing")
		}
	case <-time.After(20 * time.Second):
		t.Fatal("the plugin's decision never returned")
	}
	if got := f.installation.GetCurrentVault(); got.ID != f.first.ID {
		t.Errorf("the window is showing %s", got.Name)
	}
}

func TestTwoPluginOpenVaultDecisionsAtOnceAreNotAskingOfEachOther(t *testing.T) {
	f := openTwoVaults(t)
	f.bindShow()
	stream, isDone := f.watchQuit(t)
	defer isDone()
	flushRequest := make(chan struct{}, 1)
	release := make(chan struct{})
	f.answerFlushes(t, stream, flushRequest, release)
	list := editor.NewPluginVaultList(f.installation, f.cfg.CheckRoot())

	decided := make(chan error, 2)
	go func() {
		_, err := list.OpenVault(t.Context(), f.second.ID)
		decided <- err
	}()
	select {
	case <-flushRequest:
	case <-time.After(5 * time.Second):
		close(release)
		t.Fatal("the first decision never flushRequest the page for what it holds")
	}
	go func() {
		_, err := list.OpenVault(t.Context(), f.second.ID)
		decided <- err
	}()
	// The second decision runs while the page has not answered the first one.
	select {
	case err := <-decided:
		if errors.Is(err, pluginport.ErrVaultAsking) {
			t.Errorf("the second decision was refused as ASKING while the first one waited: %v", err)
		}
		close(release)
		if err := <-decided; err != nil {
			t.Errorf("the first decision returned %v", err)
		}
	case <-time.After(2 * time.Second):
		close(release)
		for range 2 {
			if err := <-decided; err != nil {
				t.Errorf("a decision returned %v", err)
			}
		}
	}
}

func TestPluginOpenVaultOfMissingFolderReturnsUnreadable(t *testing.T) {
	f := openTwoVaults(t)
	f.bindShow()
	if err := os.RemoveAll(f.second.Path); err != nil {
		t.Fatal(err)
	}

	list := editor.NewPluginVaultList(f.installation, f.cfg.CheckRoot())
	swap, err := list.OpenVault(t.Context(), f.second.ID)

	if !errors.Is(err, vaults.ErrUnreadable) {
		if err == nil {
			err = swap(t.Context())
		}
		t.Errorf("the decision on a vault whose folder is gone returned no ErrUnreadable; the swap then returned %v", err)
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

// startPluginDecision runs a plugin's decision to open the second vault and returns its result once
// the page has been asked to flush for it.
func startPluginDecision(t *testing.T, f *showing, flushRequest <-chan struct{}, release chan struct{}) <-chan error {
	t.Helper()
	list := editor.NewPluginVaultList(f.installation, f.cfg.CheckRoot())
	decided := make(chan error, 1)
	go func() {
		_, err := list.OpenVault(context.WithoutCancel(t.Context()), f.second.ID)
		decided <- err
	}()
	select {
	case <-flushRequest:
	case <-time.After(5 * time.Second):
		if release != nil {
			close(release)
		}
		t.Fatal("the plugin's decision never asked the page to flush")
	}
	return decided
}

// expectDecisionUnavailable checks that the plugin's decision gave way to the user's action.
func expectDecisionUnavailable(t *testing.T, decided <-chan error) {
	t.Helper()
	select {
	case err := <-decided:
		if !errors.Is(err, pluginport.ErrUnavailable) {
			t.Errorf("the plugin's decision returned %v, want ErrUnavailable", err)
		}
	case <-time.After(20 * time.Second):
		t.Fatal("the plugin's decision never returned")
	}
}

// closeLikeTheApp is the close of apps/desktop/app: a refused flush waits for the running flush and
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

func TestACloseDuringAPluginDecisionWithASilentPageClosesTheWindow(t *testing.T) {
	f := openTwoVaults(t)
	f.bindShow()
	stream, isDone := f.watchQuit(t)
	defer isDone()
	decided := startPluginDecision(t, f, watchFlushRequests(stream), nil)

	ctx, cancel := context.WithTimeout(t.Context(), 20*time.Second)
	defer cancel()
	if !closeLikeTheApp(ctx, f.installation) {
		t.Error("the user's close during a plugin's decision was refused: the window stays open")
	}
	expectDecisionUnavailable(t, decided)
}

func TestAQuitDuringAPluginDecisionIsAccepted(t *testing.T) {
	f := openTwoVaults(t)
	f.bindShow()
	stream, isDone := f.watchQuit(t)
	defer isDone()
	flushRequest := make(chan struct{}, 1)
	release := make(chan struct{})
	f.answerFlushes(t, stream, flushRequest, release)
	decided := startPluginDecision(t, f, flushRequest, release)

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
			t.Error("the user's quit was refused while a plugin decided")
		}
	case <-time.After(2 * time.Second):
		close(release)
		if !<-resolved {
			t.Error("the user's quit was refused while a plugin decided")
		}
	}
	expectDecisionUnavailable(t, decided)
}

func TestUserVaultSwitchDuringPluginDecisionIsAccepted(t *testing.T) {
	f := openTwoVaults(t)
	f.bindShow()
	stream, isDone := f.watchQuit(t)
	defer isDone()
	flushRequest := make(chan struct{}, 1)
	release := make(chan struct{})
	f.answerFlushes(t, stream, flushRequest, release)
	decided := startPluginDecision(t, f, flushRequest, release)

	switched := make(chan error, 1)
	go func() { switched <- f.installation.State.OpenVault(t.Context(), f.second) }()
	select {
	case err := <-switched:
		close(release)
		if err != nil {
			t.Errorf("the user's own vault switch was refused while a plugin decided: %v", err)
		}
	case <-time.After(2 * time.Second):
		close(release)
		if err := <-switched; err != nil {
			t.Errorf("the user's own vault switch was refused while a plugin decided: %v", err)
		}
	}
	expectDecisionUnavailable(t, decided)
	if got := f.installation.GetCurrentVault(); got.ID != f.second.ID {
		t.Errorf("the window shows %s", got.Name)
	}
}

func TestADecisionRefusedAsAskingReleasesTheFlushExclusion(t *testing.T) {
	f := openTwoVaults(t)
	f.bindShow()
	stream, isDone := f.watchQuit(t)
	defer isDone()
	go func() {
		isFirst := true
		for stream.Receive() {
			if !stream.Msg().GetFlush() {
				continue
			}
			result := v1.FlushResult_FLUSH_RESULT_NOTHING
			if isFirst {
				isFirst = false
				result = v1.FlushResult_FLUSH_RESULT_ASKING
			}
			ctx, cancel := context.WithTimeout(context.WithoutCancel(t.Context()), 5*time.Second)
			_, err := f.canvas.ReportFlush(ctx, connect.NewRequest(&v1.ReportFlushRequest{
				Window: wire.Editor, Token: stream.Msg().GetToken(), Result: result,
			}))
			cancel()
			if err != nil {
				return
			}
		}
	}()
	list := editor.NewPluginVaultList(f.installation, f.cfg.CheckRoot())
	if _, err := list.OpenVault(t.Context(), f.second.ID); !errors.Is(err, pluginport.ErrVaultAsking) {
		t.Fatalf("the decision returned %v, want ErrVaultAsking", err)
	}

	if err := f.installation.Show(t.Context(), f.second); err != nil {
		t.Errorf("the switch after a refused decision returned %v", err)
	}
	if got := f.installation.GetCurrentVault(); got.ID != f.second.ID {
		t.Errorf("the window shows %s", got.Name)
	}
}

func TestACloseBetweenADecisionAndItsSwapKeepsTheWindowOnItsVault(t *testing.T) {
	f := openTwoVaults(t)
	f.bindShow()
	list := editor.NewPluginVaultList(f.installation, f.cfg.CheckRoot())
	swap, err := list.OpenVault(t.Context(), f.second.ID)
	if err != nil {
		t.Fatal(err)
	}
	if !f.installation.FlushPending(t.Context()) {
		t.Fatal("the close after the decision was refused")
	}
	if err := swap(t.Context()); err == nil {
		t.Error("the swap after the window closed showed another vault")
	}
	if got := f.installation.GetCurrentVault(); got.ID != f.first.ID {
		t.Errorf("the closed window shows %s", got.Name)
	}
}

func TestTheSwapAfterADecisionShowsTheVault(t *testing.T) {
	f := openTwoVaults(t)
	f.bindShow()
	list := editor.NewPluginVaultList(f.installation, f.cfg.CheckRoot())
	swap, err := list.OpenVault(t.Context(), f.second.ID)
	if err != nil {
		t.Fatal(err)
	}
	if err := swap(t.Context()); err != nil {
		t.Fatalf("the swap returned %v", err)
	}
	if got := f.installation.GetCurrentVault(); got.ID != f.second.ID {
		t.Errorf("the window shows %s", got.Name)
	}
}

func TestPluginDecisionDuringUserSwitchIsUnavailable(t *testing.T) {
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
	_, err := list.OpenVault(t.Context(), f.first.ID)
	close(release)

	if !errors.Is(err, pluginport.ErrUnavailable) {
		t.Errorf("the decision during the user's switch returned %v, want ErrUnavailable", err)
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

// A close refused because a page asks the user waits for the user's answer (closeLikeTheApp). A
// plugin's decision that starts in that wait must not end the user's flush.
func TestAPluginDecisionDuringTheClosesQuestionKeepsTheCloseWaiting(t *testing.T) {
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
	_, err := list.OpenVault(ctx, f.second.ID)
	if !errors.Is(err, pluginport.ErrUnavailable) {
		t.Errorf("the plugin's decision during the close's question returned %v, want ErrUnavailable", err)
	}

	select {
	case isFlushed := <-confirmed:
		close(userAnswered)
		t.Fatalf("the plugin's decision ended the close's wait for the user's answer with %v: the window stays open after the user answers", isFlushed)
	case <-time.After(500 * time.Millisecond):
	}
	close(userAnswered)
	if !<-confirmed {
		t.Error("the close's wait ended unflushed after the user answered")
	}
}

// The user switches vault again and again while plugins keep deciding: every switch of the user
// is taken, none waits past its bound, and no decision is left holding the window.
func TestUserSwitchesAmidRepeatedPluginDecisionsAreAllTaken(t *testing.T) {
	f := openTwoVaults(t)
	f.bindShow()
	stream, isDone := f.watchQuit(t)
	defer isDone()
	f.answerAll(t, stream)
	list := editor.NewPluginVaultList(f.installation, f.cfg.CheckRoot())

	stop := make(chan struct{})
	var deciders sync.WaitGroup
	var unexpected atomic.Value
	var given, unavailable atomic.Int64
	for range 4 {
		deciders.Go(func() {
			for {
				select {
				case <-stop:
					return
				default:
				}
				_, err := list.OpenVault(context.WithoutCancel(t.Context()), f.second.ID)
				switch {
				case err == nil:
					given.Add(1)
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
				t.Errorf("switch %d of the user was refused amid plugin decisions: %v", i, err)
			}
		case <-time.After(15 * time.Second):
			close(stop)
			t.Fatalf("switch %d of the user never returned amid plugin decisions", i)
		}
	}
	close(stop)
	deciders.Wait()
	t.Logf("decisions given %d, unavailable %d", given.Load(), unavailable.Load())
	if v := unexpected.Load(); v != nil {
		t.Errorf("a decision returned other than ErrUnavailable or nil: %v", v)
	}
	if _, err := list.OpenVault(t.Context(), f.second.ID); err != nil {
		t.Errorf("a decision after the storm returned %v: the exclusion is still held", err)
	}
}

// Two user actions, a close and a vault switch, arrive while a plugin decides. Neither hangs, one of
// them is taken, and the decision gives way.
func TestTwoUserActionsDuringPluginDecisionBothReturn(t *testing.T) {
	f := openTwoVaults(t)
	f.bindShow()
	stream, isDone := f.watchQuit(t)
	defer isDone()
	flushRequest := make(chan struct{}, 1)
	release := make(chan struct{})
	f.answerFlushes(t, stream, flushRequest, release)
	decided := startPluginDecision(t, f, flushRequest, release)

	ctx, cancel := context.WithTimeout(t.Context(), 20*time.Second)
	defer cancel()
	switched := make(chan error, 1)
	closed := make(chan bool, 1)
	go func() { switched <- f.installation.State.OpenVault(ctx, f.second) }()
	go func() { closed <- closeLikeTheApp(ctx, f.installation) }()
	select {
	case err := <-decided:
		close(release)
		if !errors.Is(err, pluginport.ErrUnavailable) {
			t.Errorf("the plugin's decision returned %v, want ErrUnavailable", err)
		}
	case <-time.After(time.Second):
		close(release)
		expectDecisionUnavailable(t, decided)
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

// A plugin call that ends during its decision (the host endpoint closes, the plugin hangs up) is
// returns unavailable and does not move the window.
func TestAPluginCallThatEndsDuringItsDecisionIsUnavailable(t *testing.T) {
	f := openTwoVaults(t)
	f.bindShow()
	stream, isDone := f.watchQuit(t)
	defer isDone()
	flushRequest := watchFlushRequests(stream)
	list := editor.NewPluginVaultList(f.installation, f.cfg.CheckRoot())

	ctx, hangUp := context.WithCancel(t.Context())
	decided := make(chan error, 1)
	go func() {
		_, err := list.OpenVault(ctx, f.second.ID)
		decided <- err
	}()
	select {
	case <-flushRequest:
	case <-time.After(5 * time.Second):
		t.Fatal("the decision never asked the page to flush")
	}
	hangUp()
	select {
	case err := <-decided:
		if !errors.Is(err, pluginport.ErrUnavailable) {
			t.Errorf("the decision whose call ended returned %v, want ErrUnavailable", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("the decision whose call ended never returned")
	}
	if got := f.installation.GetCurrentVault(); got.ID != f.first.ID {
		t.Errorf("the window shows %s after a plugin call that ended", got.Name)
	}
}
