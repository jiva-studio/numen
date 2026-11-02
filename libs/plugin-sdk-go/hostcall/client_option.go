package hostcall

import "connectrpc.com/connect"

// WithBearer is the client option that presents the token on every call, unary or streaming.
func WithBearer(token string) connect.ClientOption {
	return connect.WithInterceptors(Bearer(token))
}
