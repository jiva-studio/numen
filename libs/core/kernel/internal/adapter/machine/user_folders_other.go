//go:build !linux && !windows

package machine

import "path/filepath"

func readPlatformDocuments(home string) string { return filepath.Join(home, "Documents") }
