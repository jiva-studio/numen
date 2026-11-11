package theme

import "time"

// SetKeepaliveInterval shortens the interval between keepalives until the returned function restores it.
func SetKeepaliveInterval(interval time.Duration) (restore func()) {
	before := keepaliveInterval
	keepaliveInterval = interval
	return func() { keepaliveInterval = before }
}
