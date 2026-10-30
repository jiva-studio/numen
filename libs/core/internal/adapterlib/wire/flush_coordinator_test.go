package wire

import (
	"testing"
	"time"

	v1 "github.com/jiva-studio/numen/libs/protocol/gen/numen/v1"
)

// isClosed reports whether a channel has been closed, without waiting on it.
func isClosed(c <-chan struct{}) bool {
	select {
	case <-c:
		return true
	default:
		return false
	}
}

// TestAVaultNothingIsDrawnFromOwesNothing.
func TestFlushCoordinator_NoPagesNeedsNoFlush(t *testing.T) {
	var pages flushCoordinator

	request := pages.startRequest()
	if !isClosed(request.allWritten) {
		t.Error("a vault with no page open was unembedded something")
	}
	if request.isPending() {
		t.Error("a vault with no page open raised a question")
	}
}

// TestAPageSaysNothingUntilItIsAsked.
func TestFlushCoordinator_PageIdleUntilFlushRequested(t *testing.T) {
	var pages flushCoordinator

	token, notified, isDone := pages.registerClient()
	defer isDone()

	select {
	case requestVal := <-notified:
		t.Errorf("unexpected flush call for %q", requestVal)
	default:
	}

	request := pages.startRequest()
	if isClosed(request.allWritten) {
		t.Error("a page that has said nothing was counted as having written")
	}
	select {
	case requestItem := <-notified:
		if requestItem != token {
			t.Errorf("flush token mismatch: request %q, listener %q", requestItem, token)
		}
	default:
		t.Fatal("expected flush call, got none")
	}

	pages.recordFlush(token, flushWritten)
	if !isClosed(request.allWritten) {
		t.Error("a page that wrote what it unembedded is still unembedded")
	}
	if request.isPending() {
		t.Error("a page that wrote what it unembedded raised a question")
	}
}

// TestAQuestionEndsTheRoundAndNotTheWait. A page raising a question has
// answered, so nothing is waited for; and it has not written, so the window has
// no reason to go.
func TestAQuestionEndsTheRoundAndNotTheWait(t *testing.T) {
	var pages flushCoordinator

	token, notified, isDone := pages.registerClient()
	defer isDone()

	request := pages.startRequest()
	<-notified
	pages.recordFlush(token, flushAsking)

	if !request.isPending() {
		t.Error("a question was raised and the request does not say so")
	}
	if isClosed(request.allWritten) {
		t.Fatal("a page with a question standing was counted as having written")
	}

	pages.recordFlush(token, flushWritten)
	if !isClosed(request.allWritten) {
		t.Error("the question was answered and the request is still unembedded something")
	}
}

// TestOneQuestionAmongManyPagesKeepsTheWindow.
func TestOneQuestionAmongManyPagesKeepsTheWindow(t *testing.T) {
	var pages flushCoordinator

	first, notifiedFirst, doneFirst := pages.registerClient()
	defer doneFirst()
	second, notifiedSecond, doneSecond := pages.registerClient()
	defer doneSecond()

	request := pages.startRequest()
	<-notifiedFirst
	<-notifiedSecond
	pages.recordFlush(first, flushWritten)
	pages.recordFlush(second, flushAsking)

	if !request.isPending() {
		t.Error("a question was raised and the request does not say so")
	}
	if isClosed(request.allWritten) {
		t.Fatal("the window went with a question standing")
	}

	pages.recordFlush(second, flushWritten)
	if !isClosed(request.allWritten) {
		t.Error("every page has written and the request is still unembedded something")
	}
}

// TestASecondRoundAsksEveryPageAgain. A page's earlier response to the request
// is not a response to this one.
func TestASecondRoundAsksEveryPageAgain(t *testing.T) {
	var pages flushCoordinator

	token, notified, isDone := pages.registerClient()
	defer isDone()

	first := pages.startRequest()
	<-notified
	pages.recordFlush(token, flushWritten)
	if !isClosed(first.allWritten) {
		t.Fatal("the page wrote what it unembedded and the request is still unembedded something")
	}

	second := pages.startRequest()
	if !isClosed(first.superseded) {
		t.Error("the request before was left running")
	}
	if isClosed(second.allWritten) {
		t.Error("the second request took the first request's answer for its own")
	}
	select {
	case request := <-notified:
		if request != token {
			t.Errorf("flush token mismatch: request %q, listener %q", request, token)
		}
	default:
		t.Fatal("expected flush notification for second request")
	}

	pages.recordFlush(token, flushWritten)
	if !isClosed(second.allWritten) {
		t.Error("the page wrote what it unembedded and the second request is still unembedded it")
	}
}

// TestFlushCoordinator_PageRegisteredDuringCloseReceivesFlush.
func TestFlushCoordinator_PageRegisteredDuringCloseReceivesFlush(t *testing.T) {
	var pages flushCoordinator

	request := pages.startRequest()
	if !isClosed(request.allWritten) {
		t.Fatal("a vault with no page open was unembedded something")
	}

	token, notified, isDone := pages.registerClient()
	defer isDone()

	select {
	case requestVal := <-notified:
		if requestVal != token {
			t.Errorf("flush token mismatch: request %q, listener %q", requestVal, token)
		}
	default:
		t.Error("a page that opened while the window was going was never asked")
	}
}

// TestAPageThatGoesWithNothingStandingStopsBeingOwed.
func TestAPageThatGoesWithNothingActiveStopsBeingPending(t *testing.T) {
	var pages flushCoordinator

	_, _, isDone := pages.registerClient()

	request := pages.startRequest()
	if isClosed(request.allWritten) {
		t.Fatal("a page that has said nothing was counted as having written")
	}

	isDone()
	if !isClosed(request.allWritten) {
		t.Error("a page that stopped listening is still unembedded")
	}
}

// TestAPageThatGoesWithAQuestionStandingIsStillOwed. The text is in that
// webview's buffer and nowhere else, and the stream ending gives no information
// about where it should end up.
func TestAPageThatGoesWithAQuestionActiveIsStillPending(t *testing.T) {
	var pages flushCoordinator

	token, notified, isDone := pages.registerClient()
	first := pages.startRequest()
	<-notified
	pages.recordFlush(token, flushAsking)

	isDone()

	if isClosed(first.allWritten) {
		t.Fatal("the window went with the buffer of a page that had a question standing")
	}

	// The client reconnects under a token of its own, and is queried anew.
	second := pages.startRequest()
	if isClosed(second.allWritten) {
		t.Fatal("a request with the question still standing was unembedded nothing")
	}
	again, notifiedAgain, doneAgain := pages.registerClient()
	defer doneAgain()
	if again == token {
		t.Fatalf("the page that came back listens under the token that went, %q", again)
	}
	select {
	case request := <-notifiedAgain:
		if request != again {
			t.Errorf("flush token mismatch: request %q, listener %q", request, again)
		}
	default:
		t.Fatal("the page that came back was never asked")
	}

	pages.recordFlush(again, flushAsking)
	if !second.isPending() {
		t.Error("the page raised its question again and the request does not say so")
	}
	pages.recordFlush(again, flushWritten)
	if !isClosed(second.allWritten) {
		t.Error("the question was answered and the request is still unembedded something")
	}
}

// TestFlushCoordinator_DepartedPageWithQuestionIsNotPending tests timeout
// handling when an active page disconnects during a flush cycle.
func TestFlushCoordinator_DepartedPageWithQuestionIsNotPending(t *testing.T) {
	var pages flushCoordinator

	token, notified, isDone := pages.registerClient()
	pages.startRequest()
	<-notified
	pages.recordFlush(token, flushAsking)
	isDone()

	request := pages.startRequest()
	if isClosed(request.allWritten) {
		t.Error("a request took a page that went with a question standing as written")
	}
	if request.isPending() {
		t.Error("a page that is no longer there was counted as raising a question")
	}

	// A page that returns nothing is what the bound exists for, and nothing here
	// ends the request before it.
	select {
	case <-request.allWritten:
		t.Error("the request ended without the page saying anything")
	case <-time.After(50 * time.Millisecond):
	}
}

// TestAPageWithNothingLeftAndOneThatHasWrittenBothLetTheWindowGo.
func TestAPageWithNothingLeftAndOneThatHasSavedBothLetTheWindowGo(t *testing.T) {
	for name, response := range map[string]flushState{
		"nothing unembedded": parseFlushResult(v1.FlushResult_FLUSH_RESULT_NOTHING),
		"everything written": parseFlushResult(v1.FlushResult_FLUSH_RESULT_WRITTEN),
	} {
		t.Run(name, func(t *testing.T) {
			var pages flushCoordinator

			token, notified, isDone := pages.registerClient()
			defer isDone()

			request := pages.startRequest()
			<-notified
			pages.recordFlush(token, response)

			if !isClosed(request.allWritten) {
				t.Error("window remained blocked by empty page")
			}
		})
	}
}
