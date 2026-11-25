// Package stopafter counts the bytes of one direction against a budget and cuts the connection when it ends.
package stopafter

import (
	"log/slog"
	"net/http"
)

// Limit counts the bytes of one direction since the admin call that armed it.
type Limit struct {
	isArmed bool
	left    int64
}

// Start arms the limit with a budget of bytes.
func (a *Limit) Start(bytes int64) {
	a.isArmed, a.left = true, bytes
}

// Take spends bytes of the budget and returns how many of them are allowed; isEnd is true when the budget ends.
func (a *Limit) Take(bytes int64) (allowed int64, isEnd bool) {
	if !a.isArmed {
		return bytes, false
	}
	if bytes < a.left {
		a.left -= bytes
		return bytes, false
	}
	allowed = a.left
	a.isArmed, a.left = false, 0
	return allowed, true
}

// CutConnection sends what is written to w, calls stop, and closes the connection of the response.
func CutConnection(w http.ResponseWriter, stop func()) {
	controller := http.NewResponseController(w)
	flushErr := controller.Flush()
	stop()
	if flushErr != nil {
		return
	}
	conn, _, err := controller.Hijack()
	if err != nil {
		return
	}
	if err := conn.Close(); err != nil {
		slog.Debug("closing the cut download connection", "error", err)
	}
}
