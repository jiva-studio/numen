package service

import (
	"context"
	"errors"
	"strings"
	"sync"

	"github.com/jiva-studio/numen/libs/plugin-sdk-go/server"
	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/api"
)

// codeOf returns the API error code of err, or the empty code when err is not an API error.
func codeOf(err error) server.ErrorCode {
	var failure *server.APIError
	if errors.As(err, &failure) {
		return failure.Code
	}
	return ""
}

// messageStream collects the messages a stream method sends. Receive blocks until the next message
// or the end of the method; Err returns the error the method ended with.
type messageStream[Res any] struct {
	messages chan *Res
	cancel   context.CancelFunc
	finished chan struct{}
	current  *Res
	mu       sync.Mutex
	err      error
}

// openStream runs the stream method under a cancellable context and delivers its messages to the
// returned stream one at a time.
func openStream[Req, Res any](
	ctx context.Context, req *Req, method func(context.Context, *Req, func(*Res) error) error,
) (*messageStream[Res], error) {
	ctx, cancel := context.WithCancel(ctx)
	s := &messageStream[Res]{messages: make(chan *Res, 256), cancel: cancel, finished: make(chan struct{})}
	go func() {
		defer close(s.finished)
		defer close(s.messages)
		err := method(ctx, req, func(res *Res) error {
			select {
			case s.messages <- res:
				return nil
			case <-ctx.Done():
				return ctx.Err()
			}
		})
		s.mu.Lock()
		s.err = err
		s.mu.Unlock()
	}()
	return s, nil
}

// Receive waits for the next message and reports whether there is one.
func (s *messageStream[Res]) Receive() bool {
	res, open := <-s.messages
	s.current = res
	return open
}

// Msg returns the message that Receive reported.
func (s *messageStream[Res]) Msg() *Res { return s.current }

// Err returns the error the method ended with, after Receive reports false.
func (s *messageStream[Res]) Err() error {
	<-s.finished
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.err
}

// Close cancels the method and waits for it to return.
func (s *messageStream[Res]) Close() {
	s.cancel()
	for range s.messages {
	}
	<-s.finished
}

// reviewClient calls the review service the way the window does: a unary method
// returns directly and a stream method returns a messageStream.
type reviewClient struct {
	*reviewHandler
}

// WatchCardsDue opens the stream of vault counts.
func (c reviewClient) WatchCardsDue(
	ctx context.Context, req *api.WatchCardsDueRequest,
) (*messageStream[api.WatchCardsDueResponse], error) {
	return openStream(ctx, req, c.reviewHandler.WatchCardsDue)
}

// WatchReloads opens the stream of reload messages.
func (c reviewClient) WatchReloads(
	ctx context.Context, req *api.WatchReloadsRequest,
) (*messageStream[api.WatchReloadsResponse], error) {
	return openStream(ctx, req, c.reviewHandler.WatchReloads)
}

// hasCause reports whether err is an API error whose message carries the text of target.
func hasCause(err error, target error) bool {
	var failure *server.APIError
	return errors.As(err, &failure) && strings.Contains(failure.Message, target.Error())
}
