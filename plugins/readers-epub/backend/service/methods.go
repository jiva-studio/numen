package service

import (
	"context"
	"errors"
	"net/http"

	"github.com/jiva-studio/numen/libs/plugin-sdk-go/server"
	"github.com/jiva-studio/numen/plugins/readers-epub/backend/api"
)

func mountMethods(s *server.Server, books *BookServer) error {
	a := s.NewAPI(api.Service)
	server.Handle(a, "GetBook", withVault(books.GetBook))
	server.Handle(a, "ReadBookMarkup", withVault(books.ReadBookMarkup))
	server.Handle(a, "ReadBookEntry", withVault(books.ReadBookEntry))
	server.Handle(a, "GetCfi", withVault(books.GetCfi))
	server.Handle(a, "FindOffset", withVault(books.FindOffset))
	// gomutants:disable-next-line RETURN_ERROR_NIL reason="mountMethods returns any handler registration error accumulated on a"
	return a.Err()
}

func withVault[Req, Res any](fn func(ctx context.Context, req Req) (Res, error)) func(ctx context.Context, req Req) (Res, error) {
	return func(ctx context.Context, req Req) (Res, error) {
		res, err := fn(withRequestVault(ctx, server.GetHeader(ctx)), req)
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
	if errors.Is(err, errChanged) {
		return server.NewAPIError(server.CodeFailedPrecondition, err.Error())
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
