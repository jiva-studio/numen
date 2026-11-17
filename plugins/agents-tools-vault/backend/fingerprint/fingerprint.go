// Package fingerprint writes the content of a file as the token the tools hand to an agent, and reads it back.
package fingerprint

import (
	"errors"
	"fmt"
	"regexp"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/domain"
)

var hashPattern = regexp.MustCompile(`^[0-9a-f]{64}$`)

// Format writes the hash of the content of the file: the SHA-256 in lowercase hex.
func Format(ref domain.FileStat) string {
	return ref.Hash
}

// Parse reads a token Format wrote; the stat carries the hash only, as a baseline for a write.
func Parse(token string) (domain.FileStat, error) {
	if token == "" {
		return domain.FileStat{}, errors.New("fingerprint is required")
	}
	if !hashPattern.MatchString(token) {
		return domain.FileStat{}, fmt.Errorf("invalid fingerprint format: %q", token)
	}
	return domain.FileStat{Hash: token}, nil
}
