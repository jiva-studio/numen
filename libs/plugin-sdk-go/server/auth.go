package server

import (
	"context"
	"crypto/subtle"
	"errors"
	"net/http"
	"strings"
	"sync"

	"connectrpc.com/connect"
)

// ErrUnauthorized indicates missing or mismatched authentication credentials.
var ErrUnauthorized = errors.New("unauthorized")

// AuthInterceptor authenticates incoming Connect-RPC calls against an expected bearer token.
type AuthInterceptor struct {
	mu    sync.RWMutex
	token string
}

// NewAuthInterceptor creates a Connect-RPC interceptor validating bearer token credentials.
func NewAuthInterceptor(token string) *AuthInterceptor {
	return &AuthInterceptor{token: token}
}

// SetToken updates the expected bearer token.
func (a *AuthInterceptor) SetToken(token string) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.token = token
}

func (a *AuthInterceptor) checkToken(header http.Header) error {
	a.mu.RLock()
	token := a.token
	a.mu.RUnlock()

	if token == "" {
		return connect.NewError(connect.CodeUnauthenticated, ErrUnauthorized)
	}
	authHeader := header.Get("Authorization")
	if !strings.HasPrefix(authHeader, "Bearer ") {
		return connect.NewError(connect.CodeUnauthenticated, ErrUnauthorized)
	}
	received := strings.TrimPrefix(authHeader, "Bearer ")
	if subtle.ConstantTimeCompare([]byte(received), []byte(token)) != 1 {
		return connect.NewError(connect.CodeUnauthenticated, ErrUnauthorized)
	}
	return nil
}

// WrapUnary wraps unary RPC handlers with token verification.
func (a *AuthInterceptor) WrapUnary(next connect.UnaryFunc) connect.UnaryFunc {
	return func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
		if err := a.checkToken(req.Header()); err != nil {
			return nil, err
		}
		return next(ctx, req)
	}
}

// WrapStreamingClient passes through streaming client calls.
func (a *AuthInterceptor) WrapStreamingClient(next connect.StreamingClientFunc) connect.StreamingClientFunc {
	return next
}

// WrapStreamingHandler wraps streaming RPC handlers with token verification.
func (a *AuthInterceptor) WrapStreamingHandler(next connect.StreamingHandlerFunc) connect.StreamingHandlerFunc {
	return func(ctx context.Context, conn connect.StreamingHandlerConn) error {
		if err := a.checkToken(conn.RequestHeader()); err != nil {
			return err
		}
		return next(ctx, conn)
	}
}
