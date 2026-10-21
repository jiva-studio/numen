package app

import (
	"context"
	"slices"
	"sync"
	"testing"
	"time"
)

// screen records the window going out of sight and coming back into it.
type screen struct {
	mu   sync.Mutex
	seen []string
}

func makeScreen() *screen { return &screen{} }

func (s *screen) sight() visibility {
	return visibility{hide: func() { s.mark("hide") }, show: func() { s.mark("show") }}
}

func (s *screen) mark(what string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.seen = append(s.seen, what)
}

// was is what has happened to the window so far.
func (s *screen) was() []string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return slices.Clone(s.seen)
}

// settler is a vault whose settling a test writes the answers for, and which
// records how many times it was asked.
type settler struct {
	mu      sync.Mutex
	answers []bool
	asked   int
	// begun is written to as each settling starts.
	begun chan struct{}
	// until holds each settling there until a test lets it through.
	until chan struct{}
}

func newSettler(answers ...bool) *settler {
	return &settler{answers: answers, begun: make(chan struct{}, len(answers)+1)}
}

func (s *settler) settle(context.Context) bool {
	s.mu.Lock()
	held := s.until
	answer := true
	if s.asked < len(s.answers) {
		answer = s.answers[s.asked]
	}
	s.asked++
	s.mu.Unlock()

	select {
	case s.begun <- struct{}{}:
	default:
	}
	if held != nil {
		<-held
	}
	return answer
}

func (s *settler) getSettledCount() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.asked
}

// TestACloseCalledOffCanBeAskedForAgain. A settling that ends with a question
// standing leaves the window where it is, and every later ask settles the vault
// again.
func TestACloseCalledOffCanBeAskedForAgain(t *testing.T) {
	vault := newSettler(false, false, true)
	g := &vaultFlush{flush: vault.settle}

	if g.flushVault() {
		t.Fatal("a question standing let the window go")
	}
	if g.isFlushed() {
		t.Error("a called-off close left the vault settled")
	}
	if g.flushVault() {
		t.Fatal("the second ask let the window go with a question standing")
	}
	if !g.flushVault() {
		t.Fatal("the ask after the question was answered did not let the window go")
	}
	if !g.isFlushed() {
		t.Error("the vault settled and does not say so")
	}
	if vault.getSettledCount() != 3 {
		t.Errorf("the vault was settled %d times", vault.getSettledCount())
	}
}

// TestASecondQuitStillFlushes. A vault that settled is not settled again; one
// that did not is, and that is what writes what a page holds.
func TestASecondQuitStillFlushes(t *testing.T) {
	vault := newSettler(false, true)
	g := &vaultFlush{flush: vault.settle}

	g.flushVault()
	if !g.flushVault() {
		t.Fatal("the second quit did not settle the vault")
	}
	if vault.getSettledCount() != 2 {
		t.Fatalf("the vault was settled %d times", vault.getSettledCount())
	}

	// Settled, and asking again costs nothing.
	if !g.flushVault() {
		t.Error("a settled vault answered that it had not")
	}
	if vault.getSettledCount() != 2 {
		t.Errorf("a settled vault was settled again, %d times in all", vault.getSettledCount())
	}
}

// TestConcurrentFlushesShareOneRun. The window's hook and a quit
// asked for elsewhere both wait, and the vault settles once for the two.
func TestConcurrentFlushesShareOneRun(t *testing.T) {
	vault := newSettler(true)
	vault.until = make(chan struct{})
	g := &vaultFlush{flush: vault.settle}

	answers := make(chan bool, 2)
	for range 2 {
		go func() { answers <- g.flushVault() }()
	}

	select {
	case <-vault.begun:
	case <-time.After(5 * time.Second):
		t.Fatal("the vault was never settled")
	}
	close(vault.until)

	for range 2 {
		select {
		case answer := <-answers:
			if !answer {
				t.Error("a waiter was told the vault had not settled")
			}
		case <-time.After(5 * time.Second):
			t.Fatal("a waiter was never answered")
		}
	}
	if vault.getSettledCount() != 1 {
		t.Errorf("the vault was settled %d times", vault.getSettledCount())
	}
}

// TestAQuitAskedForElsewhereDoesNotGoOnAQuestion. The goroutine a quit leaves
// behind may refuse the quit and nothing else: a person is answering, and the
// application is not to go out from under them.
func TestAQuitAskedForElsewhereDoesNotGoOnAQuestion(t *testing.T) {
	vault := newSettler(false)
	g := &vaultFlush{flush: vault.settle}

	quit := make(chan struct{}, 1)
	if requestQuit(g, makeScreen().sight(), func() { quit <- struct{}{} }) {
		t.Fatal("the quit went before the vault settled")
	}

	select {
	case <-quit:
		t.Fatal("a question standing quit the application")
	case <-time.After(300 * time.Millisecond):
	}
}

// TestRequestedQuitProceedsOnceVaultIsFlushed.
func TestRequestedQuitProceedsOnceVaultIsFlushed(t *testing.T) {
	vault := newSettler(true)
	g := &vaultFlush{flush: vault.settle}

	quit := make(chan struct{}, 1)
	if requestQuit(g, makeScreen().sight(), func() { quit <- struct{}{} }) {
		t.Fatal("the quit went before the vault settled")
	}

	select {
	case <-quit:
	case <-time.After(5 * time.Second):
		t.Fatal("the quit was never asked for again")
	}
	if !requestQuit(g, makeScreen().sight(), func() { t.Error("a settled vault was quit twice") }) {
		t.Error("a settled vault rejected the quit")
	}
}

// TestAQuestionCallsTheCloseOffAndTheWindowIsAskedForAgain. The person answers,
// the page says it has nothing left, and the close the person asked for
// happens.
func TestAQuestionCallsTheCloseOffAndTheWindowIsAskedForAgain(t *testing.T) {
	vault := newSettler(false)
	g := &vaultFlush{flush: vault.settle}

	person := make(chan struct{})
	answered := func(context.Context) bool {
		<-person
		return true
	}
	again := make(chan struct{}, 1)

	if closeWindow(t.Context(), g, makeScreen().sight(), answered, func() { again <- struct{}{} }) {
		t.Fatal("a question standing let the window go")
	}
	select {
	case <-again:
		t.Fatal("the close was asked for again before the person answered")
	case <-time.After(200 * time.Millisecond):
	}

	close(person)

	select {
	case <-again:
	case <-time.After(5 * time.Second):
		t.Fatal("the close was never asked for again")
	}
}

// TestAWindowNobodyAnswersForIsNotAskedForAgain. A page that raised a question
// and then went says nothing more, and the window stays where the person left
// it.
func TestAWindowNobodyAnswersForIsNotAskedForAgain(t *testing.T) {
	vault := newSettler(false)
	g := &vaultFlush{flush: vault.settle}

	again := make(chan struct{}, 1)
	if closeWindow(t.Context(), g, makeScreen().sight(), func(context.Context) bool { return false }, func() {
		again <- struct{}{}
	}) {
		t.Fatal("a question standing let the window go")
	}

	select {
	case <-again:
		t.Fatal("the close was asked for again with the question unanswered")
	case <-time.After(300 * time.Millisecond):
	}
}

// TestWindowWithFlushedVaultIsDestroyed.
func TestWindowWithFlushedVaultIsDestroyed(t *testing.T) {
	g := &vaultFlush{flush: newSettler(true).settle}

	if !closeWindow(t.Context(), g, makeScreen().sight(), func(context.Context) bool {
		t.Error("a settled vault was waited on for an answer")
		return false
	}, func() { t.Error("a settled vault asked for the close again") }) {
		t.Fatal("a settled vault did not let the window go")
	}
}

// TestWindowIsHiddenBeforeFlushBegins. The window goes from the
// screen when the close is asked for, and everything owed lands behind it.
func TestWindowIsHiddenBeforeFlushBegins(t *testing.T) {
	seen := makeScreen()
	begun := make(chan []string, 1)
	g := &vaultFlush{flush: func(context.Context) bool {
		begun <- seen.was()
		return true
	}}

	if !closeWindow(t.Context(), g, seen.sight(), func(context.Context) bool {
		t.Error("a settled vault was waited on for an answer")
		return false
	}, func() { t.Error("a settled vault asked for the close again") }) {
		t.Fatal("a settled vault did not let the window go")
	}

	select {
	case was := <-begun:
		if !slices.Equal(was, []string{"hide"}) {
			t.Errorf("the window was %v when the settling began", was)
		}
	default:
		t.Fatal("the vault was never settled")
	}
	if was := seen.was(); !slices.Equal(was, []string{"hide"}) {
		t.Errorf("the window that went was %v", was)
	}
}

// TestQuestionRestoresWindowAndBlocksQuit. A settling that ends with a
// question standing is a window on the screen the person answers on, and an
// application that has not gone.
func TestQuestionRestoresWindowAndBlocksQuit(t *testing.T) {
	seen := makeScreen()
	g := &vaultFlush{flush: newSettler(false).settle}

	quit := make(chan struct{}, 1)
	if requestQuit(g, seen.sight(), func() { quit <- struct{}{} }) {
		t.Fatal("the quit went before the vault settled")
	}

	select {
	case <-quit:
		t.Fatal("a question standing quit the application")
	case <-time.After(300 * time.Millisecond):
	}
	if was := seen.was(); !slices.Equal(was, []string{"hide", "show"}) {
		t.Errorf("the window a question was asked in was %v", was)
	}
}

// TestWindowTracker tracks window counts and identifies the last remaining window.
func TestWindowTracker(t *testing.T) {
	tracker := newWindowTracker()
	if !tracker.isLast() {
		t.Error("an empty tracker should report isLast")
	}
	if tracker.countWindows() != 0 {
		t.Errorf("expected 0 windows, got %d", tracker.countWindows())
	}

	tracker.addWindow()
	if !tracker.isLast() {
		t.Error("a single window should report isLast")
	}
	if tracker.countWindows() != 1 {
		t.Errorf("expected 1 window, got %d", tracker.countWindows())
	}

	tracker.addWindow()
	if tracker.isLast() {
		t.Error("two windows should not report isLast")
	}
	if tracker.countWindows() != 2 {
		t.Errorf("expected 2 windows, got %d", tracker.countWindows())
	}

	if tracker.removeWindow() {
		t.Error("removing one of two windows should not report 0 left")
	}
	if !tracker.isLast() {
		t.Error("one remaining window should report isLast")
	}
	if !tracker.removeWindow() {
		t.Error("removing last window should report 0 left")
	}

	tracker.addWindow()
	tracker.addWindow()
	if !tracker.closeSecondaryWindow() {
		t.Error("closeSecondaryWindow should succeed when more than one window remains")
	}
	if tracker.countWindows() != 1 {
		t.Errorf("expected 1 window, got %d", tracker.countWindows())
	}
	if tracker.closeSecondaryWindow() {
		t.Error("closeSecondaryWindow should report false for the last window")
	}
	if tracker.countWindows() != 1 {
		t.Errorf("expected window count to remain 1, got %d", tracker.countWindows())
	}
}

// TestHandleWindowClosingSecondaryWindowClosesWithoutFlushing. Closing a secondary
// window leaves the vault untouched, while closing the last window flushes it.
func TestHandleWindowClosingSecondaryWindowClosesWithoutFlushing(t *testing.T) {
	tracker := newWindowTracker()
	tracker.addWindow()
	tracker.addWindow()

	settler := newSettler(true)
	g := &vaultFlush{flush: settler.settle}
	seen := makeScreen()

	closed := handleWindowClosing(t.Context(), tracker, g, seen.sight(), func(context.Context) bool {
		t.Error("secondary window close should not wait for flush answers")
		return false
	}, func() {
		t.Error("secondary window close should not retry")
	})

	if !closed {
		t.Fatal("secondary window closing should succeed immediately")
	}
	if settler.getSettledCount() != 0 {
		t.Errorf("vault should not have flushed on secondary window close, got %d flushes", settler.getSettledCount())
	}
	if !tracker.isLast() {
		t.Error("tracker should now report last window remaining")
	}

	// Now close the last remaining window
	lastClosed := handleWindowClosing(t.Context(), tracker, g, seen.sight(), func(context.Context) bool {
		return false
	}, func() {})

	if !lastClosed {
		t.Fatal("last window closing should succeed when vault flushes")
	}
	if settler.getSettledCount() != 1 {
		t.Errorf("vault should have flushed once on last window close, got %d flushes", settler.getSettledCount())
	}
}
