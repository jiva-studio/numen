//go:build !linux && !darwin

package performance

// SetNice does nothing on a system without a niceness the process can set.
func SetNice(int) error { return nil }
