package link

import "fmt"

// GetSizeText returns a size in bytes below one megabyte as bytes and any other as whole megabytes.
func GetSizeText(bytes int64) string {
	if bytes < 1<<20 {
		return fmt.Sprintf("%d bytes", bytes)
	}
	return fmt.Sprintf("%d MB", bytes>>20)
}
