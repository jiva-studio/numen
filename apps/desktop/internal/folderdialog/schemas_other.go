//go:build !linux

package folderdialog

// SetSchemasDir does nothing: only Linux needs the schemas.
func SetSchemasDir() error { return nil }

func hasSchemas() bool { return true }
