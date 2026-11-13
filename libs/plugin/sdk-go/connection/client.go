package connection

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"connectrpc.com/connect"

	v1 "github.com/jiva-studio/numen/libs/core/protocol/gen/numen/v1"
	"github.com/jiva-studio/numen/libs/core/protocol/gen/numen/v1/numenv1connect"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/hostcall"
)

var (
	// ErrNoConnection means the settings of the plugin hold no row that can serve the operation.
	ErrNoConnection = errors.New("connection: no connection serves the operation")
	// ErrStreamEnded means the stream of a call ended with neither a done nor a failure.
	ErrStreamEnded = errors.New("connection: the stream ended before the call did")
)

// Failure is one row that did not answer: the name of its connection and the reason.
type Failure struct{ ConnectionName, Reason string }

// AllFailedError means every row of the plugin was set aside or failed.
type AllFailedError struct{ Failures []Failure }

// Error names each connection and its reason, in the order of the rows.
func (e *AllFailedError) Error() string {
	parts := make([]string, len(e.Failures))
	for i, one := range e.Failures {
		parts[i] = one.ConnectionName + ": " + one.Reason
	}
	return "connection: every connection failed: " + strings.Join(parts, "; ")
}

// StoppedError means a connection found the request itself wrong, so no other connection was tried.
type StoppedError struct{ ConnectionName, Reason string }

// Error names the connection and the reason.
func (e *StoppedError) Error() string {
	return "connection: " + e.ConnectionName + " stopped the request: " + e.Reason
}

// Priority says whose action a call serves. A call that sets none is background, so it waits behind the
// calls of a person's actions when its connection runs as many calls as it may.
//
//numen:api
type Priority int

const (
	// PriorityBackground is work that serves no person's action, such as indexing.
	PriorityBackground Priority = iota
	// PriorityInteractive is a person's action, such as a search or a check of a line.
	PriorityInteractive
)

// Frames are the frames of one call. A reader calls Next until it returns false, then reads Err, and
// reads Done where Err is nil.
type Frames interface {
	// Next moves to the next data frame. It returns false at the end of the call and on an error.
	Next() bool
	// Data is the data frame Next moved to.
	Data() []byte
	// Done is the done payload, once Next returned false and Err is nil.
	Done() []byte
	// Err returns why the call ended: nil after a done, ErrNoConnection, *AllFailedError or *StoppedError
	// for a call the core could not serve, the error of the context for a canceled call, or ErrStreamEnded.
	Err() error
	// Close ends the call and cancels it if it has not ended.
	Close() error
}

// Caller carries out one operation through the rows of the plugin. A Client is the Caller of a plugin
// process; a plugin compiled into the core is handed one of the core.
type Caller interface {
	Call(ctx context.Context, operation string, payload []byte, priority Priority) (Frames, error)
}

// Client asks the core to carry out operations for one plugin. The core takes the connections from the
// rows of the settings of that plugin; a request names a connection only as the row it prefers, and a
// result holds no key.
type Client struct {
	host numenv1connect.HostConnectionServiceClient
}

// NewClient creates a client for the host at baseURL, authenticating with token.
func NewClient(httpClient connect.HTTPClient, baseURL, token string) *Client {
	return &Client{host: numenv1connect.NewHostConnectionServiceClient(
		httpClient, baseURL, connect.WithInterceptors(hostcall.Bearer(token)),
	)}
}

// NewClientFromEnv creates a client for the host named by NUMEN_HOST_PORT, authenticating with
// NUMEN_PLUGIN_TOKEN.
func NewClientFromEnv() (*Client, error) {
	endpoint, err := hostcall.FromEnv()
	if err != nil {
		return nil, err
	}
	return newClientAt(endpoint), nil
}

// NewClientFromVars creates a client for the host named by the NUMEN_HOST_PORT entry of vars,
// authenticating with its NUMEN_PLUGIN_TOKEN entry. They are the variables a host gives a plugin it links
// into its own process.
func NewClientFromVars(vars map[string]string) (*Client, error) {
	endpoint, err := hostcall.FromVars(vars)
	if err != nil {
		return nil, err
	}
	return newClientAt(endpoint), nil
}

func newClientAt(endpoint hostcall.Endpoint) *Client {
	return NewClient(hostcall.HTTPClient(), endpoint.BaseURL, endpoint.Token)
}

// Call asks the rows of the plugin, in order, to carry out the operation. The core goes to the next row
// only when a row fails before its first data frame. The caller closes the frames.
func (c *Client) Call(ctx context.Context, operation string, payload []byte, priority Priority) (Frames, error) {
	return c.call(ctx, &v1.HostConnectionServiceCallRequest{Operation: operation, Payload: payload, Priority: writePriority(priority)})
}

func (c *Client) call(ctx context.Context, request *v1.HostConnectionServiceCallRequest) (Frames, error) {
	stream, err := c.host.Call(ctx, connect.NewRequest(request))
	if err != nil {
		if ctxErr := ctx.Err(); ctxErr != nil {
			return nil, ctxErr
		}
		return nil, fmt.Errorf("connection: call %s: %w", request.GetOperation(), err)
	}
	return &callStream{stream: stream, contextErr: ctx.Err, operation: request.GetOperation()}, nil
}

type callStream struct {
	stream     *connect.ServerStreamForClient[v1.HostConnectionServiceCallResponse]
	contextErr func() error
	operation  string
	data       []byte
	done       []byte
	err        error
	isDone     bool
}

func (s *callStream) Next() bool {
	for !s.isDone {
		if !s.stream.Receive() {
			s.isDone = true
			s.err = s.readEnd()
			return false
		}
		switch frame := s.stream.Msg().GetFrame().(type) {
		case *v1.HostConnectionServiceCallResponse_Data:
			s.data = frame.Data
			return true
		case *v1.HostConnectionServiceCallResponse_Done:
			s.isDone, s.done = true, frame.Done
		case *v1.HostConnectionServiceCallResponse_Failure:
			s.isDone = true
			s.err = readFailure(frame.Failure)
		}
	}
	return false
}

func (s *callStream) Data() []byte { return s.data }

func (s *callStream) Done() []byte { return s.done }

func (s *callStream) Err() error { return s.err }

func (s *callStream) Close() error { return s.stream.Close() }

func (s *callStream) readEnd() error {
	if err := s.stream.Err(); err != nil {
		if ctxErr := s.contextErr(); ctxErr != nil {
			return ctxErr
		}
		return fmt.Errorf("connection: call %s: %w", s.operation, err)
	}
	return ErrStreamEnded
}

func readFailure(failure *v1.CallFailure) error {
	failures := failure.GetFailures()
	switch failure.GetError() {
	case v1.CallErrorCode_CALL_ERROR_CODE_NO_CONNECTION:
		return ErrNoConnection
	case v1.CallErrorCode_CALL_ERROR_CODE_ALL_FAILED:
		all := &AllFailedError{Failures: make([]Failure, len(failures))}
		for i, one := range failures {
			all.Failures[i] = Failure{ConnectionName: one.GetConnectionName(), Reason: one.GetReason()}
		}
		return all
	case v1.CallErrorCode_CALL_ERROR_CODE_STOPPED:
		stopped := &StoppedError{}
		if len(failures) > 0 {
			stopped.ConnectionName, stopped.Reason = failures[0].GetConnectionName(), failures[0].GetReason()
		}
		return stopped
	case v1.CallErrorCode_CALL_ERROR_CODE_UNSPECIFIED:
	}
	return fmt.Errorf("connection: unknown error code %d: %s", failure.GetError(), failure.GetDetail())
}

// Collect reads frames to their end and returns the data frames in order with the done payload. It closes
// the frames.
func Collect(frames Frames) (data [][]byte, done []byte, err error) {
	defer func() { err = errors.Join(err, frames.Close()) }()
	for frames.Next() {
		data = append(data, frames.Data())
	}
	if err := frames.Err(); err != nil {
		return nil, nil, err
	}
	return data, frames.Done(), nil
}

func writePriority(priority Priority) v1.Priority {
	if priority == PriorityInteractive {
		return v1.Priority_PRIORITY_INTERACTIVE
	}
	return v1.Priority_PRIORITY_BACKGROUND
}
