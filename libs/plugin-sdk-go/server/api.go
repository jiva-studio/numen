package server

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
)

// ErrorCode names why an API method failed. The codes are the Connect codes in snake case, so a
// client reads the error of a method the same way whether the method is declared by a schema or not.
type ErrorCode string

const (
	CodeInvalidArgument    ErrorCode = "invalid_argument"
	CodeNotFound           ErrorCode = "not_found"
	CodeAlreadyExists      ErrorCode = "already_exists"
	CodePermissionDenied   ErrorCode = "permission_denied"
	CodeFailedPrecondition ErrorCode = "failed_precondition"
	CodeUnavailable        ErrorCode = "unavailable"
	CodeInternal           ErrorCode = "internal"
)

// getStatus returns the HTTP status that answers a method failing with the code.
func (c ErrorCode) getStatus() int {
	switch c {
	case CodeInvalidArgument:
		return http.StatusBadRequest
	case CodeNotFound:
		return http.StatusNotFound
	case CodeAlreadyExists:
		return http.StatusConflict
	case CodePermissionDenied:
		return http.StatusForbidden
	case CodeFailedPrecondition:
		return http.StatusPreconditionFailed
	case CodeUnavailable:
		return http.StatusServiceUnavailable
	default:
		return http.StatusInternalServerError
	}
}

// APIError is the failure of an API method, as the client reads it.
type APIError struct {
	Code    ErrorCode `json:"code"`
	Message string    `json:"message"`
}

// NewAPIError creates the failure of a method with the code and the message the client reads.
func NewAPIError(code ErrorCode, message string) *APIError {
	return &APIError{Code: code, Message: message}
}

func (e *APIError) Error() string {
	return string(e.Code) + ": " + e.Message
}

// toAPIError returns err as the failure a client reads: an APIError as it is, any other error as an
// internal failure with its text.
func toAPIError(err error) *APIError {
	var failure *APIError
	if errors.As(err, &failure) {
		return failure
	}
	return NewAPIError(CodeInternal, err.Error())
}

// API serves the JSON methods of a plugin under one service name. The name is declared in the
// manifest `services`; the host forwards every request whose path starts with it to the plugin with
// the host token, and a method is mounted at POST /<service>/<method>.
type API struct {
	server  *Server
	service string
}

// NewAPI creates the API of the service name on the server.
func (s *Server) NewAPI(service string) *API {
	if service == "" || strings.ContainsAny(service, "/ {}") {
		panic(fmt.Sprintf("server: invalid API service name %q", service))
	}
	return &API{server: s, service: service}
}

func (a *API) mount(method string, handler http.Handler) {
	if method == "" || strings.ContainsAny(method, "/ {}") {
		panic(fmt.Sprintf("server: invalid API method name %q", method))
	}
	a.server.Register(http.MethodPost+" /"+a.service+"/"+method, a.server.interceptor.RequireToken(handler))
}

// Handle mounts a method that reads one request and answers one response, both JSON. Unknown
// request fields are ignored and an empty body is a zero request. A method failing with an
// APIError answers its code; any other error answers internal.
func Handle[Req, Res any](a *API, method string, fn func(ctx context.Context, req Req) (Res, error)) {
	a.mount(method, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req Req
		if err := readRequest(r.Body, &req); err != nil {
			writeFailure(w, NewAPIError(CodeInvalidArgument, err.Error()))
			return
		}
		res, err := fn(r.Context(), req)
		if err != nil {
			writeFailure(w, toAPIError(err))
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(res)
	}))
}

// Stream mounts a method that reads one request and sends any number of responses as
// Server-Sent Events: each response is a `data:` event holding its JSON, a failure is a final
// `error` event holding an APIError, and a method returning nil ends with an `end` event.
func Stream[Req, Res any](a *API, method string, fn func(ctx context.Context, req Req, send func(Res) error) error) {
	a.mount(method, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req Req
		if err := readRequest(r.Body, &req); err != nil {
			writeFailure(w, NewAPIError(CodeInvalidArgument, err.Error()))
			return
		}
		flusher := http.NewResponseController(w)
		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.WriteHeader(http.StatusOK)
		_ = flusher.Flush()

		send := func(res Res) error {
			return writeEvent(w, flusher, "", res)
		}
		if err := fn(r.Context(), req, send); err != nil {
			if r.Context().Err() == nil {
				_ = writeEvent(w, flusher, "error", toAPIError(err))
			}
			return
		}
		_ = writeEvent(w, flusher, "end", struct{}{})
	}))
}

func readRequest(body io.Reader, req any) error {
	err := json.NewDecoder(body).Decode(req)
	if errors.Is(err, io.EOF) {
		return nil
	}
	return err
}

func writeFailure(w http.ResponseWriter, failure *APIError) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(failure.Code.getStatus())
	_ = json.NewEncoder(w).Encode(failure)
}

func writeEvent(w http.ResponseWriter, flusher *http.ResponseController, name string, payload any) error {
	data, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	if name != "" {
		if _, err := fmt.Fprintf(w, "event: %s\n", name); err != nil {
			return err
		}
	}
	if _, err := fmt.Fprintf(w, "data: %s\n\n", data); err != nil {
		return err
	}
	return flusher.Flush()
}
