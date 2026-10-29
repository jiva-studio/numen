package service

import (
	"context"
	"errors"
	"net/http"

	"github.com/jiva-studio/numen/libs/plugin-sdk-go/server"
	"github.com/jiva-studio/numen/plugins/readers-epub/backend/api"
)

// mountMethods mounts the methods of the numen.epub service on s. A method works the vault the
// Numen-Vault header of its request names.
func mountMethods(s *server.Server, books *BookServer) {
	a := s.NewAPI(api.Service)
	server.Handle(a, "GetBook", withVault(books.GetBook))
	server.Handle(a, "ReadBookMarkup", withVault(books.ReadBookMarkup))
	server.Handle(a, "ReadBookEntry", withVault(books.ReadBookEntry))
	server.Handle(a, "GetCfi", withVault(books.GetCfi))
	server.Handle(a, "FindOffset", withVault(books.FindOffset))
}

// withVault returns fn with the vault of the request header in its context and its failures as API
// errors.
func withVault[Req, Res any](fn func(ctx context.Context, req Req) (Res, error)) func(ctx context.Context, req Req) (Res, error) {
	return func(ctx context.Context, req Req) (Res, error) {
		res, err := fn(withRequestVault(ctx, server.GetHeader(ctx)), req)
		if err != nil {
			return res, toAPIError(err)
		}
		return res, nil
	}
}

// toAPIError returns err as the failure a client reads: an APIError as it is, a changed file as a
// failed precondition, and any other error as an internal failure.
func toAPIError(err error) *server.APIError {
	var failure *server.APIError
	if errors.As(err, &failure) {
		return failure
	}
	if errors.Is(err, errChanged) {
		return server.NewAPIError(server.CodeFailedPrecondition, err.Error())
	}
	return server.NewAPIError(server.CodeInternal, err.Error())
}

// getStatus returns the HTTP status that answers a failure of the code.
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
	default:
		return http.StatusInternalServerError
	}
}
