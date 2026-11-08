//go:build !linux

package keychain

func isSessionBusReachable() bool { return true }
