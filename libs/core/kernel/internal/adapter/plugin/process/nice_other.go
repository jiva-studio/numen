//go:build !linux && !darwin && !windows

package process

// SetNice leaves process scheduling to the platform.
func SetNice(_, _ int) error { return nil }
