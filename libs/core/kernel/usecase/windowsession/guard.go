package windowsession

import (
	"context"
	"errors"
	"sync"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/window"
)

// Guard is whether a flush of pending writes is running and whether the window has closed. Vault
// swaps and window close both flush, one at a time. The zero value is a window that is open and idle.
//
// A window that has closed shows no other vault.
type Guard struct {
	mu         sync.Mutex
	isBusy     bool
	isClosing  bool
	pluginOpen *PluginOpen

	switching sync.Mutex
}

// PluginOpen tracks a running plugin-initiated vault open; a user flush cancels it. A vault switch
// given the running open shares the open's lock; a user vault switch is given nil.
type PluginOpen struct {
	cancel context.CancelCauseFunc
	end    chan struct{}
}

// errCancelledByUser is the cause a plugin's vault open is canceled with when the user closes the window,
// quits or switches its vault.
var errCancelledByUser = errors.New("the user closed the window or switched its vault")

func (s *Guard) tryBegin() error {
	s.mu.Lock()
	for s.pluginOpen != nil {
		current := s.pluginOpen
		s.mu.Unlock()
		current.cancel(errCancelledByUser)
		<-current.end
		s.mu.Lock()
	}
	defer s.mu.Unlock()

	switch {
	case s.isClosing:
		return window.ErrClosing
	case s.isBusy:
		return window.ErrFlushing
	}
	s.isBusy = true
	return nil
}

func (s *Guard) tryBeginPluginOpen(ctx context.Context) (context.Context, *PluginOpen, func(), error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	switch {
	case s.isClosing:
		return nil, nil, nil, window.ErrClosing
	case s.isBusy:
		return nil, nil, nil, window.ErrFlushing
	}
	ctx, cancel := context.WithCancelCause(ctx)
	open := &PluginOpen{cancel: cancel, end: make(chan struct{})}
	s.isBusy, s.pluginOpen = true, open
	return ctx, open, func() {
		s.mu.Lock()
		s.isBusy, s.pluginOpen = false, nil
		s.mu.Unlock()
		cancel(nil)
		close(open.end)
	}, nil
}

func (s *Guard) tryBeginSwitch(open *PluginOpen) (func(), bool, error) {
	s.mu.Lock()
	isInsidePluginOpen := open != nil && s.pluginOpen == open
	s.mu.Unlock()
	if isInsidePluginOpen {
		return func() {}, true, nil
	}
	if err := s.tryBegin(); err != nil {
		return nil, false, err
	}
	return s.endKeepOpen, false, nil
}

func (s *Guard) isClosed() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.isClosing
}

func (s *Guard) endKeepOpen() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.isBusy = false
}

func (s *Guard) end(closed bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.isBusy = false
	s.isClosing = closed
}
