package pool

import (
	"context"
	"errors"
	"image"
	"slices"
	"sync"
	"testing"
	"time"

	"github.com/jiva-studio/numen/modules/libs/core/port"
	"github.com/jiva-studio/numen/modules/libs/core/usecase/document"
)

// shelf is a Scan that counts what is asked of it.
type shelf struct {
	mu       sync.Mutex
	across   float64
	sizeErr  error
	imageErr error
	sizes    int
	dpis     []int
	closes   int
}

func (s *shelf) GetPageCount() int { return 1 }

func (s *shelf) GetPageSize(int) (float64, float64, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.sizes++
	return s.across, 792, s.sizeErr
}

func (s *shelf) RenderPage(_, dpi int) (image.Image, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.dpis = append(s.dpis, dpi)
	if s.imageErr != nil {
		return nil, s.imageErr
	}
	return image.NewRGBA(image.Rect(0, 0, 1, 1)), nil
}

func (s *shelf) Close() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.closes++
}

func (s *shelf) getCloses() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.closes
}

func (s *shelf) isClosed() bool { return s.getCloses() > 0 }

func opens(s port.OpenDocument) func() (port.OpenDocument, error) {
	return func() (port.OpenDocument, error) { return s, nil }
}

func markOf(name string) document.Fingerprint {
	return document.Fingerprint{Path: name, Size: 1, Mtime: 1}
}

// waitFor holds until the condition does, or fails the test.
func waitFor(t *testing.T, what string, holds func() bool) {
	t.Helper()
	deadline := time.Now().Add(3 * time.Second)
	for !holds() {
		if time.Now().After(deadline) {
			t.Fatalf("never happened: %s", what)
		}
		time.Sleep(time.Millisecond)
	}
}

func getOpenCount(d *Documents) int {
	d.mu.Lock()
	defer d.mu.Unlock()
	return len(d.open)
}

func getOrder(d *Documents) []document.Fingerprint {
	d.mu.Lock()
	defer d.mu.Unlock()
	return slices.Clone(d.order)
}

func TestTheBoundsAWindowKeepsDocumentsUnder(t *testing.T) {
	if DocumentIdleTimeout != 2*time.Minute {
		t.Errorf("a document nobody looks at is kept %v", DocumentIdleTimeout)
	}
	if PointsDPI != 72 {
		t.Errorf("a page's own unit is 1/%d inch", PointsDPI)
	}
}

func TestAWindowKeepsMostOpenDocumentsAndClosesTheOldest(t *testing.T) {
	docs := NewDocuments(MaxOpenDocuments, DocumentIdleTimeout)
	t.Cleanup(docs.Close)
	var scans []*shelf
	for _, name := range []string{"a", "b", "c"} {
		s := &shelf{}
		scans = append(scans, s)
		_, give, err := docs.Acquire(t.Context(), markOf(name), opens(s))
		if err != nil {
			t.Fatal(err)
		}
		give()
	}
	if got := getOpenCount(docs); got != MaxOpenDocuments {
		t.Errorf("%d documents are open, want %d", got, MaxOpenDocuments)
	}
	if !scans[0].isClosed() {
		t.Error("the oldest document was kept")
	}
	if scans[1].isClosed() || scans[2].isClosed() {
		t.Error("a document within the bound was closed")
	}
}

func TestDocumentsAtTheBoundAreAllKept(t *testing.T) {
	docs := NewDocuments(2, DocumentIdleTimeout)
	t.Cleanup(docs.Close)
	a, b := &shelf{}, &shelf{}
	for name, s := range map[string]*shelf{"a": a, "b": b} {
		_, give, err := docs.Acquire(t.Context(), markOf(name), opens(s))
		if err != nil {
			t.Fatal(err)
		}
		give()
	}
	if a.isClosed() || b.isClosed() || getOpenCount(docs) != 2 {
		t.Error("a window holding exactly its bound closed a document")
	}
}

func TestADocumentAskedForAgainIsTheNewestInTheOrder(t *testing.T) {
	docs := NewDocuments(2, DocumentIdleTimeout)
	t.Cleanup(docs.Close)
	a, b, c := &shelf{}, &shelf{}, &shelf{}
	take := func(name string, s *shelf) {
		_, give, err := docs.Acquire(t.Context(), markOf(name), opens(s))
		if err != nil {
			t.Fatal(err)
		}
		give()
	}
	take("a", a)
	take("b", b)
	if got := getOrder(docs); !slices.Equal(got, []document.Fingerprint{markOf("a"), markOf("b")}) {
		t.Fatalf("the order is %v", got)
	}
	take("b", b)
	if got := getOrder(docs); !slices.Equal(got, []document.Fingerprint{markOf("a"), markOf("b")}) {
		t.Errorf("asking for the newest again left the order %v", got)
	}
	take("a", a)
	if got := getOrder(docs); !slices.Equal(got, []document.Fingerprint{markOf("b"), markOf("a")}) {
		t.Errorf("the order is %v after a was asked for again", got)
	}
	take("c", c)
	if got := getOrder(docs); !slices.Equal(got, []document.Fingerprint{markOf("a"), markOf("c")}) {
		t.Errorf("the order is %v after b was closed", got)
	}
	if !b.isClosed() || a.isClosed() {
		t.Errorf("the document closed was not the least recently asked for (a closed %v, b closed %v)", a.isClosed(), b.isClosed())
	}
}

func TestADocumentInUseIsNotClosedForTheBound(t *testing.T) {
	docs := NewDocuments(1, DocumentIdleTimeout)
	t.Cleanup(docs.Close)
	a, b := &shelf{}, &shelf{}
	_, giveA, err := docs.Acquire(t.Context(), markOf("a"), opens(a))
	if err != nil {
		t.Fatal(err)
	}
	_, giveB, err := docs.Acquire(t.Context(), markOf("b"), opens(b))
	if err != nil {
		t.Fatal(err)
	}
	if a.isClosed() || b.isClosed() || getOpenCount(docs) != 2 {
		t.Error("a document somebody holds was closed")
	}
	giveA()
	giveB()
}

func TestADocumentThatCannotBeOpenedIsReportedAndNotKept(t *testing.T) {
	docs := NewDocuments(MaxOpenDocuments, DocumentIdleTimeout)
	t.Cleanup(docs.Close)
	cause := errors.New("not a document")
	good := &shelf{}
	_, giveGood, err := docs.Acquire(t.Context(), markOf("good"), opens(good))
	if err != nil {
		t.Fatal(err)
	}
	defer giveGood()

	attempts := 0
	broken := func() (port.OpenDocument, error) {
		attempts++
		return nil, cause
	}
	for range 2 {
		doc, give, err := docs.Acquire(t.Context(), markOf("bad"), broken)
		if !errors.Is(err, cause) {
			t.Fatalf("a document that cannot be opened was answered %v", err)
		}
		if doc != nil || give != nil {
			t.Error("a document that cannot be opened was handed over")
		}
	}
	if attempts != 2 {
		t.Errorf("the file was read %d times, want 2", attempts)
	}
	if getOpenCount(docs) != 1 || !slices.Equal(getOrder(docs), []document.Fingerprint{markOf("good")}) {
		t.Errorf("the failed document is still held: %d open, order %v", getOpenCount(docs), getOrder(docs))
	}
}

func TestADocumentAskedAfterTheWindowWentDoesNotLockTheWindow(t *testing.T) {
	docs := NewDocuments(MaxOpenDocuments, DocumentIdleTimeout)
	docs.Close()
	done := make(chan struct{})
	go func() {
		defer close(done)
		for range 2 {
			if _, _, err := docs.Acquire(t.Context(), markOf("a"), opens(&shelf{})); !errors.Is(err, ErrBusy) {
				t.Errorf("answered %v", err)
			}
		}
	}()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("the second ask waited on a lock the first kept")
	}
}

func TestACallerThatRanOutOfTimeIsToldTheDocumentIsBusy(t *testing.T) {
	docs := NewDocuments(MaxOpenDocuments, DocumentIdleTimeout)
	t.Cleanup(docs.Close)
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	release := make(chan struct{})
	slow := func() (port.OpenDocument, error) {
		<-release
		return &shelf{}, nil
	}
	_, _, err := docs.Acquire(ctx, markOf("a"), slow)
	if !errors.Is(err, ErrBusy) {
		t.Fatalf("answered %v", err)
	}
	docs.mu.Lock()
	uses := docs.open[markOf("a")].uses
	docs.mu.Unlock()
	if uses != 0 {
		t.Errorf("a caller that gave up still holds the document (%d uses)", uses)
	}
	close(release)
}

func TestADocumentIsHeldByOneCallAtATime(t *testing.T) {
	docs := NewDocuments(MaxOpenDocuments, DocumentIdleTimeout)
	t.Cleanup(docs.Close)
	doc, give, err := docs.Acquire(t.Context(), markOf("a"), opens(&shelf{}))
	if err != nil {
		t.Fatal(err)
	}
	defer give()

	first, cancelFirst := context.WithTimeout(t.Context(), 2*time.Second)
	defer cancelFirst()
	if !doc.Lock(first) {
		t.Fatal("a free document could not be held")
	}
	second, cancelSecond := context.WithTimeout(t.Context(), 30*time.Millisecond)
	defer cancelSecond()
	if doc.Lock(second) {
		t.Fatal("a document held by one call was held by another")
	}
	doc.Unlock()
	third, cancelThird := context.WithTimeout(t.Context(), 2*time.Second)
	defer cancelThird()
	if !doc.Lock(third) {
		t.Error("a document let go could not be held")
	}
	doc.Unlock()
}

func TestAskingAgainStopsTheClockOnClosingTheDocument(t *testing.T) {
	docs := NewDocuments(MaxOpenDocuments, time.Hour)
	t.Cleanup(docs.Close)
	s := &shelf{}
	_, give, err := docs.Acquire(t.Context(), markOf("a"), opens(s))
	if err != nil {
		t.Fatal(err)
	}
	give()
	doc := docs.open[markOf("a")]
	clock := doc.idle
	if clock == nil {
		t.Fatal("giving a document back started no clock")
	}

	_, give, err = docs.Acquire(t.Context(), markOf("a"), opens(s))
	if err != nil {
		t.Fatal(err)
	}
	if doc.idle != nil {
		t.Error("a document in use is still on the clock")
	}
	if clock.Stop() {
		t.Error("the clock of a document in use was left running")
	}
	give()
}

func TestADocumentStillHeldByAnotherCallIsNotOnTheClock(t *testing.T) {
	docs := NewDocuments(MaxOpenDocuments, time.Hour)
	t.Cleanup(docs.Close)
	s := &shelf{}
	_, giveOne, err := docs.Acquire(t.Context(), markOf("a"), opens(s))
	if err != nil {
		t.Fatal(err)
	}
	_, giveTwo, err := docs.Acquire(t.Context(), markOf("a"), opens(s))
	if err != nil {
		t.Fatal(err)
	}
	doc := docs.open[markOf("a")]

	giveOne()
	if doc.idle != nil {
		t.Error("a document still in use was put on the clock")
	}
	giveTwo()
	if doc.idle == nil {
		t.Error("a document nobody holds was not put on the clock")
	}
}

func TestADocumentNobodyLooksAtIsClosedAndForgotten(t *testing.T) {
	docs := NewDocuments(MaxOpenDocuments, 10*time.Millisecond)
	t.Cleanup(docs.Close)
	s := &shelf{}
	_, give, err := docs.Acquire(t.Context(), markOf("a"), opens(s))
	if err != nil {
		t.Fatal(err)
	}
	give()
	waitFor(t, "the idle document is closed", s.isClosed)
	waitFor(t, "the idle document is forgotten", func() bool {
		return getOpenCount(docs) == 0 && len(getOrder(docs)) == 0
	})
}

func TestADocumentGivenBackAfterItWasEvictedIsClosedOnce(t *testing.T) {
	docs := NewDocuments(1, DocumentIdleTimeout)
	t.Cleanup(docs.Close)
	a, b := &shelf{}, &shelf{}
	_, giveA, err := docs.Acquire(t.Context(), markOf("a"), opens(a))
	if err != nil {
		t.Fatal(err)
	}
	giveA()
	_, giveB, err := docs.Acquire(t.Context(), markOf("b"), opens(b))
	if err != nil {
		t.Fatal(err)
	}
	giveB()
	if got := a.getCloses(); got != 1 {
		t.Errorf("the evicted document was closed %d times", got)
	}
}

func TestAWindowGoingClosesWhatEveryoneHasGivenBack(t *testing.T) {
	docs := NewDocuments(100, time.Hour)
	var idle, busy []*shelf
	var clocks []*time.Timer
	var gives []func()
	for i := range 12 {
		s := &shelf{}
		name := string(rune('a' + i))
		_, give, err := docs.Acquire(t.Context(), markOf(name), opens(s))
		if err != nil {
			t.Fatal(err)
		}
		if i%2 == 0 {
			give()
			idle = append(idle, s)
			clocks = append(clocks, docs.open[markOf(name)].idle)
		} else {
			busy = append(busy, s)
			gives = append(gives, give)
		}
	}

	gone := make(chan struct{})
	go func() {
		docs.Close()
		close(gone)
	}()

	for i, s := range idle {
		waitFor(t, "an idle document is closed", s.isClosed)
		if clocks[i].Stop() {
			t.Error("the clock of a closed document was left running")
		}
	}
	for _, s := range busy {
		if s.isClosed() {
			t.Error("a document in use was closed from under its reader")
		}
	}
	select {
	case <-gone:
		t.Fatal("the window went with documents in use")
	default:
	}

	for _, give := range gives {
		give()
	}
	select {
	case <-gone:
	case <-time.After(2 * time.Second):
		t.Fatal("the window never went")
	}
	for _, s := range busy {
		if got := s.getCloses(); got != 1 {
			t.Errorf("a document in use was closed %d times once given back", got)
		}
	}
	if len(getOrder(docs)) != 0 {
		t.Error("a window that went still orders documents")
	}
}

func TestShuttingADocumentThatNeverOpenedClosesNothing(t *testing.T) {
	ready := make(chan struct{})
	close(ready)
	(&Document{ready: ready}).Close()
}

func TestShuttingADocumentStillOpeningClosesItOnceOpened(t *testing.T) {
	s := &shelf{}
	doc := &Document{ready: make(chan struct{})}
	doc.Close()
	time.Sleep(20 * time.Millisecond)
	if s.isClosed() {
		t.Fatal("closed before it was open")
	}
	doc.Scan = s
	close(doc.ready)
	waitFor(t, "the document is closed once open", s.isClosed)

	never := &Document{ready: make(chan struct{})}
	never.Close()
	close(never.ready)
	time.Sleep(10 * time.Millisecond)
}

func TestShuttingAnOpenDocumentClosesItsScan(t *testing.T) {
	s := &shelf{}
	ready := make(chan struct{})
	close(ready)
	(&Document{ready: ready, Scan: s}).Close()
	if got := s.getCloses(); got != 1 {
		t.Errorf("the scan was closed %d times", got)
	}
}

func TestAPageIsDrawnAtTheResolutionItsWidthNeeds(t *testing.T) {
	for _, c := range []struct {
		name          string
		across        float64
		width, wantAt int
	}{
		{"as wide as the page", 612, 612, 72},
		{"narrower than the page", 612, 100, 12},
		{"twice the page", 612, 1224, 144},
		{"just over a resolution", 600, 601, 73},
		{"nothing asked", 612, 0, 1},
		{"a page one unit wide", 1, 5, 360},
	} {
		t.Run(c.name, func(t *testing.T) {
			s := &shelf{across: c.across}
			doc := &Document{Scan: s, points: map[int]int{}}
			if _, err := doc.Render(0, c.width); err != nil {
				t.Fatal(err)
			}
			if len(s.dpis) != 1 || s.dpis[0] != c.wantAt {
				t.Errorf("drawn at %v, want %d", s.dpis, c.wantAt)
			}
		})
	}
}

func TestAPagesWidthIsMeasuredOnce(t *testing.T) {
	s := &shelf{across: 612}
	doc := &Document{Scan: s, points: map[int]int{}}
	for range 3 {
		if _, err := doc.Render(4, 612); err != nil {
			t.Fatal(err)
		}
	}
	if s.sizes != 1 {
		t.Errorf("the page was measured %d times", s.sizes)
	}
	if _, err := doc.Render(5, 612); err != nil {
		t.Fatal(err)
	}
	if s.sizes != 2 {
		t.Errorf("another page was not measured (%d measures)", s.sizes)
	}
}

func TestAPageThatCannotBeDrawnIsReported(t *testing.T) {
	cause := errors.New("the library said no")

	measure := &shelf{across: 612, sizeErr: cause}
	if _, err := (&Document{Scan: measure, points: map[int]int{}}).Render(0, 100); !errors.Is(err, cause) {
		t.Errorf("a page that cannot be measured was answered %v", err)
	}
	if len(measure.dpis) != 0 {
		t.Error("a page that cannot be measured was drawn")
	}

	empty := &shelf{across: 0.5}
	doc := &Document{Scan: empty, points: map[int]int{}}
	if _, err := doc.Render(0, 100); err == nil {
		t.Error("a page with no width was drawn")
	}
	if len(empty.dpis) != 0 || len(doc.points) != 0 {
		t.Error("a page with no width was drawn or remembered")
	}

	draw := &shelf{across: 612, imageErr: cause}
	img, err := (&Document{Scan: draw, points: map[int]int{}}).Render(0, 100)
	if !errors.Is(err, cause) || img != nil {
		t.Errorf("a page the library cannot draw was answered %v, %v", img, err)
	}
}

func TestAnEvictedDocumentFromTheMiddleOfTheOrderLeavesTheRestInOrder(t *testing.T) {
	docs := NewDocuments(1, time.Hour)
	t.Cleanup(docs.Close)
	_, giveA, err := docs.Acquire(t.Context(), markOf("a"), opens(&shelf{}))
	if err != nil {
		t.Fatal(err)
	}
	defer giveA()
	b := &shelf{}
	_, giveB, err := docs.Acquire(t.Context(), markOf("b"), opens(b))
	if err != nil {
		t.Fatal(err)
	}
	giveB()
	clock := docs.open[markOf("b")].idle

	_, giveC, err := docs.Acquire(t.Context(), markOf("c"), opens(&shelf{}))
	if err != nil {
		t.Fatal(err)
	}
	defer giveC()
	if !b.isClosed() {
		t.Error("the idle document was not closed")
	}
	if clock.Stop() {
		t.Error("the clock of an evicted document was left running")
	}
	if got := getOrder(docs); !slices.Equal(got, []document.Fingerprint{markOf("a"), markOf("c")}) {
		t.Errorf("the order is %v", got)
	}
}

func TestADocumentThatFailedToOpenIsClosedOnceGivenBack(t *testing.T) {
	docs := NewDocuments(MaxOpenDocuments, DocumentIdleTimeout)
	t.Cleanup(docs.Close)
	s := &shelf{}
	cause := errors.New("half open")
	if _, _, err := docs.Acquire(t.Context(), markOf("a"), func() (port.OpenDocument, error) { return s, cause }); !errors.Is(err, cause) {
		t.Fatalf("answered %v", err)
	}
	if got := s.getCloses(); got != 1 {
		t.Errorf("the scan was closed %d times", got)
	}
}

func TestRetiringClosesOnlyTheDocumentNobodyHolds(t *testing.T) {
	docs := NewDocuments(MaxOpenDocuments, time.Hour)
	t.Cleanup(docs.Close)
	s := &shelf{}
	doc, give, err := docs.Acquire(t.Context(), markOf("a"), opens(s))
	if err != nil {
		t.Fatal(err)
	}

	docs.CloseIfIdle(markOf("a"), doc)
	if s.isClosed() || getOpenCount(docs) != 1 {
		t.Fatal("a document in use was retired")
	}

	give()
	docs.CloseIfIdle(markOf("a"), &Document{ready: make(chan struct{})})
	if s.isClosed() || getOpenCount(docs) != 1 {
		t.Fatal("a document was retired for another's clock")
	}

	docs.CloseIfIdle(markOf("a"), doc)
	if !s.isClosed() || getOpenCount(docs) != 0 || len(getOrder(docs)) != 0 {
		t.Error("an idle document was not retired")
	}
}

func TestADocumentGivenBackAfterItWasReplacedLeavesTheReplacementHeld(t *testing.T) {
	docs := NewDocuments(MaxOpenDocuments, DocumentIdleTimeout)
	mark := markOf("a")
	failed := &Document{uses: 1}
	replacement := &Document{uses: 1}
	docs.open[mark] = replacement
	docs.isClosing = true

	docs.Release(mark, failed)

	if docs.open[mark] != replacement {
		t.Error("giving back the document that failed removed the one that replaced it")
	}
	select {
	case <-docs.empty:
		t.Error("the window was told nothing is open while the replacement is held")
	default:
	}
}
