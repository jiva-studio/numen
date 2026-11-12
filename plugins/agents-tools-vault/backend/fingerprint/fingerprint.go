// Package fingerprint writes the stat of a file as the token the tools hand to an agent, and reads it back.
package fingerprint

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/domain"
)

// Format writes the size and the modification time of the file as "<size>-<nanoseconds>".
func Format(ref domain.FileStat) string {
	var nanos int64
	if !ref.ModTime.IsZero() {
		nanos = ref.ModTime.UnixNano()
	}
	return strconv.FormatInt(ref.Size, 10) + "-" + strconv.FormatInt(nanos, 10)
}

// Parse reads a token Format wrote; the path and kind of the stat stay empty.
func Parse(token string) (domain.FileStat, error) {
	if token == "" {
		return domain.FileStat{}, errors.New("fingerprint is required")
	}
	sizeText, timeText, found := strings.Cut(token, "-")
	if !found {
		return domain.FileStat{}, fmt.Errorf("invalid fingerprint format: %q", token)
	}
	size, err := strconv.ParseInt(sizeText, 10, 64)
	if err != nil {
		return domain.FileStat{}, fmt.Errorf("invalid fingerprint size: %w", err)
	}
	nanos, err := strconv.ParseInt(timeText, 10, 64)
	if err != nil {
		return domain.FileStat{}, fmt.Errorf("invalid fingerprint timestamp: %w", err)
	}
	var modTime time.Time
	if nanos != 0 {
		modTime = time.Unix(0, nanos)
	}
	return domain.FileStat{Path: "", Kind: "", Size: size, ModTime: modTime}, nil
}
