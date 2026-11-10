package windowsession

import (
	"context"
	"errors"
	"slices"
	"testing"
)

func TestFlushingTwiceAnswersTrueTheSecondTime(t *testing.T) {
	w := newTestWindow(t, nil)
	w.open(t, garden())

	first := w.scenarios.Closing.Flush.Execute(t.Context())
	second := w.scenarios.Closing.Flush.Execute(t.Context())

	if !first || !second || w.pages.flushes != 1 {
		t.Errorf("flushes answered %v and %v after %d requests, want true, true after one", first, second, w.pages.flushes)
	}
}

func TestFlushPendingReturnsFalseWhileAPageAsks(t *testing.T) {
	w := newTestWindow(t, nil)
	w.open(t, garden())
	w.pages.isAsking = true

	if w.scenarios.Closing.Flush.Execute(t.Context()) {
		t.Fatal("the flush ended with a page awaiting an answer")
	}
	w.pages.isAsking = false

	if !w.scenarios.Closing.Flush.Execute(t.Context()) {
		t.Error("the flush after the answer did not end")
	}
}

type closing struct {
	log  *eventLog
	name string
	err  error
}

func (c closing) Close(context.Context) error {
	c.log.add("closed " + c.name)
	return c.err
}

func TestClosingShutsTheDoorFirstThenStopsThePassesThenClosesResourcesInOrder(t *testing.T) {
	log := &eventLog{}
	w := newLoggedWindow(t, log, func(deps *Deps) {
		deps.Shutdown.Closers = []Closer{closing{log: log, name: "socket"}, closing{log: log, name: "index"}}
	})
	w.open(t, garden())

	if err := w.scenarios.Closing.Close.Execute(t.Context()); err != nil {
		t.Fatal(err)
	}

	want := []string{"recorded v-garden", "start v-garden", "door closed", "retire v-garden", "closed socket", "closed index"}
	if got := log.all(); !slices.Equal(got, want) {
		t.Errorf("the window did %v, want %v", got, want)
	}
}

func TestClosingReportsEveryResourceThatFailedToClose(t *testing.T) {
	socket, index := errors.New("the socket stays"), errors.New("the index is locked")
	log := &eventLog{}
	w := newLoggedWindow(t, log, func(deps *Deps) {
		deps.Shutdown.Closers = []Closer{closing{log: log, name: "socket", err: socket}, closing{log: log, name: "index", err: index}}
	})
	w.open(t, garden())

	err := w.scenarios.Closing.Close.Execute(t.Context())

	if !errors.Is(err, socket) || !errors.Is(err, index) {
		t.Errorf("closing returned %v, want both failures", err)
	}
}
