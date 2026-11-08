//go:build !linux

package keychain

// isSessionBusReachable is true off Linux: only go-keyring's ErrUnsupportedPlatform and the D-Bus
// errors name an absent keychain there.
func isSessionBusReachable() bool { return true }
