package syncprotocol

import (
	"context"
	"errors"
	"strconv"

	"connectrpc.com/connect"

	syncv1 "github.com/jiva-studio/numen/libs/shared/syncprotocol/gen/numen/sync/v1"
)

type versionClientInterceptor struct{}

// NewVersionInterceptor returns the client interceptor that sends Version in every request and turns the
// answer of a server of another version into an error that wraps ErrUpdateRequired.
func NewVersionInterceptor() connect.Interceptor { return versionClientInterceptor{} }

func (versionClientInterceptor) WrapUnary(next connect.UnaryFunc) connect.UnaryFunc {
	return func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
		req.Header().Set(HeaderVersion, strconv.FormatUint(uint64(Version), 10))
		response, err := next(ctx, req)
		return response, readUpdateRequired(err)
	}
}

func (versionClientInterceptor) WrapStreamingClient(next connect.StreamingClientFunc) connect.StreamingClientFunc {
	return func(ctx context.Context, spec connect.Spec) connect.StreamingClientConn {
		conn := next(ctx, spec)
		conn.RequestHeader().Set(HeaderVersion, strconv.FormatUint(uint64(Version), 10))
		return versionClientConn{StreamingClientConn: conn}
	}
}

// versionClientConn turns the errors of a stream that name another version into ErrUpdateRequired.
type versionClientConn struct{ connect.StreamingClientConn }

func (c versionClientConn) Receive(message any) error {
	return readUpdateRequired(c.StreamingClientConn.Receive(message))
}

func (c versionClientConn) CloseResponse() error {
	return readUpdateRequired(c.StreamingClientConn.CloseResponse())
}

func (versionClientInterceptor) WrapStreamingHandler(next connect.StreamingHandlerFunc) connect.StreamingHandlerFunc {
	return next
}

func readUpdateRequired(err error) error {
	var failure *connect.Error
	if !errors.As(err, &failure) || failure.Code() != connect.CodeFailedPrecondition {
		return err
	}
	for _, detail := range failure.Details() {
		value, valueErr := detail.Value()
		if valueErr != nil {
			continue
		}
		if _, isRequired := value.(*syncv1.UpdateRequired); isRequired {
			required := connect.NewError(connect.CodeFailedPrecondition, ErrUpdateRequired)
			for _, kept := range failure.Details() {
				required.AddDetail(kept)
			}
			return required
		}
	}
	return err
}

type versionHandlerInterceptor struct{ version uint32 }

// NewVersionHandlerInterceptor returns the server interceptor that answers a request whose header holds
// another version than version, or none, or something that is no number, with FAILED_PRECONDITION and an
// UpdateRequired detail, before the call runs.
func NewVersionHandlerInterceptor(version uint32) connect.Interceptor {
	return versionHandlerInterceptor{version: version}
}

func (i versionHandlerInterceptor) WrapUnary(next connect.UnaryFunc) connect.UnaryFunc {
	return func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
		if err := i.check(req.Header().Get(HeaderVersion)); err != nil {
			return nil, err
		}
		return next(ctx, req)
	}
}

func (versionHandlerInterceptor) WrapStreamingClient(next connect.StreamingClientFunc) connect.StreamingClientFunc {
	return next
}

func (i versionHandlerInterceptor) WrapStreamingHandler(next connect.StreamingHandlerFunc) connect.StreamingHandlerFunc {
	return func(ctx context.Context, conn connect.StreamingHandlerConn) error {
		if err := i.check(conn.RequestHeader().Get(HeaderVersion)); err != nil {
			return err
		}
		return next(ctx, conn)
	}
}

func (i versionHandlerInterceptor) check(header string) error {
	if sent, err := strconv.ParseUint(header, 10, 32); err == nil && uint32(sent) == i.version {
		return nil
	}
	failure := connect.NewError(connect.CodeFailedPrecondition, ErrUpdateRequired)
	if detail, err := connect.NewErrorDetail(&syncv1.UpdateRequired{ServerVersion: i.version}); err == nil {
		failure.AddDetail(detail)
	}
	return failure
}
