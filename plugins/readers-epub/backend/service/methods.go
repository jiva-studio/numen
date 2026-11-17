package service

import (
	"context"
	"errors"
	"net/http"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/server"
	"github.com/jiva-studio/numen/plugins/readers-epub/backend/api"
	"github.com/jiva-studio/numen/plugins/readers-epub/backend/usecase/books"
	"github.com/jiva-studio/numen/plugins/readers-epub/backend/wire/bookwire"
)

func mountMethods(s *server.Server, methods *bookwire.Methods) error {
	a := s.NewAPI(api.Service)
	server.Handle(a, "GetBook", withVault(methods.GetBook))
	server.Handle(a, "ReadBookMarkup", withVault(methods.ReadBookMarkup))
	server.Handle(a, "ReadBookEntry", withVault(methods.ReadBookEntry))
	server.Handle(a, "GetCfi", withVault(methods.GetCfi))
	server.Handle(a, "FindOffset", withVault(methods.FindOffset))
	// gomutants:disable-next-line RETURN_ERROR_NIL reason="mountMethods returns any handler registration error accumulated on a"
	return a.Err()
}

func withVault[Req, Res any](fn func(ctx context.Context, req Req) (Res, error)) func(ctx context.Context, req Req) (Res, error) {
	return func(ctx context.Context, req Req) (Res, error) {
		res, err := fn(books.WithVault(ctx, host.GetRequestVault(server.GetHeader(ctx))), req)
		if err != nil {
			// gomutants:disable-next-line RETURN_ZERO reason="res is ignored when an error is returned"
			return res, toAPIError(err)
		}
		return res, nil
	}
}

func toAPIError(err error) *server.APIError {
	if failure, ok := errors.AsType[*server.APIError](err); ok {
		return failure
	}
	return server.NewAPIError(server.CodeInternal, err.Error())
}

func getStatus(code server.ErrorCode) int {
	switch code {
	case server.CodeInvalidArgument:
		return http.StatusBadRequest
	case server.CodeNotFound:
		return http.StatusNotFound
	case server.CodeAlreadyExists:
		return http.StatusConflict
	case server.CodePermissionDenied:
		return http.StatusForbidden
	case server.CodeFailedPrecondition:
		return http.StatusConflict
	case server.CodeUnavailable:
		return http.StatusServiceUnavailable
	case server.CodeInternal:
		return http.StatusInternalServerError
	default:
		return http.StatusInternalServerError
	}
}
