//go:build !linux

package app

func setSchemasDir() error { return nil }

func hasSchemas() bool { return true }
