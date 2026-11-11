package theme

import "time"

// SetKeepalive shortens the interval between the empty reports of a watch.
func (s *Service) SetKeepalive(interval time.Duration) { s.keepalive = interval }
