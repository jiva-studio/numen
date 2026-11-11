package wiretest

import (
	"context"
	"net/http"
)

// NewRootedHandler serves the handler the way a window does: the request
// context is the process's own, and ends when the application ends and at no
// other moment.
func NewRootedHandler(handler http.Handler, entered, returned chan<- struct{}) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		entered <- struct{}{}
		defer func() { returned <- struct{}{} }()
		handler.ServeHTTP(w, r.WithContext(context.WithoutCancel(r.Context())))
	})
}
