// Package semver reads and orders the versions of plugins, which follow semantic versioning.
package semver

import (
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// ErrInvalid is a text that is not a semantic version.
var ErrInvalid = errors.New("plugin version is not valid semver (e.g. 1.0.0)")

var pattern = regexp.MustCompile(
	`^v?(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)(-((0|[1-9][0-9]*)|([0-9A-Za-z-]*[A-Za-z-][0-9A-Za-z-]*))(\.((0|[1-9][0-9]*)|([0-9A-Za-z-]*[A-Za-z-][0-9A-Za-z-]*)))*)?(\+([0-9A-Za-z-]+)(\.[0-9A-Za-z-]+)*)?$`,
)

// IsValid reports whether text is a semantic version, with or without a leading v.
func IsValid(text string) bool { return pattern.MatchString(text) }

// Compare orders two semantic versions by precedence: -1 when a precedes b, 1 when b precedes a
// and 0 when they are equal. A leading v and build metadata are ignored.
func Compare(a, b string) (int, error) {
	left, err := parseVersion(a)
	if err != nil {
		return 0, err
	}
	right, err := parseVersion(b)
	if err != nil {
		return 0, err
	}
	for i := range left.numbers {
		if left.numbers[i] != right.numbers[i] {
			return compareNumbers(left.numbers[i], right.numbers[i]), nil
		}
	}
	return comparePrerelease(left.prerelease, right.prerelease), nil
}

type version struct {
	numbers    [3]uint64
	prerelease []string
}

func parseVersion(text string) (version, error) {
	if !IsValid(text) {
		return version{}, fmt.Errorf("%w: %q", ErrInvalid, text)
	}
	core := strings.TrimPrefix(text, "v")
	core, _, _ = strings.Cut(core, "+")
	core, prerelease, hasPrerelease := strings.Cut(core, "-")
	var parsed version
	for i, part := range strings.Split(core, ".") {
		number, err := strconv.ParseUint(part, 10, 64)
		if err != nil {
			return version{}, fmt.Errorf("%w: %q: %w", ErrInvalid, text, err)
		}
		parsed.numbers[i] = number
	}
	if hasPrerelease {
		parsed.prerelease = strings.Split(prerelease, ".")
	}
	return parsed, nil
}

func compareNumbers(a, b uint64) int {
	switch {
	case a < b:
		return -1
	case a > b:
		return 1
	}
	return 0
}

func comparePrerelease(a, b []string) int {
	if len(a) == 0 || len(b) == 0 {
		return compareNumbers(uint64(len(b)), uint64(len(a)))
	}
	for i := 0; i < len(a) && i < len(b); i++ {
		if order := compareIdentifier(a[i], b[i]); order != 0 {
			return order
		}
	}
	return compareNumbers(uint64(len(a)), uint64(len(b)))
}

func compareIdentifier(a, b string) int {
	left, errLeft := strconv.ParseUint(a, 10, 64)
	right, errRight := strconv.ParseUint(b, 10, 64)
	switch {
	case errLeft == nil && errRight == nil:
		return compareNumbers(left, right)
	case errLeft == nil:
		return -1
	case errRight == nil:
		return 1
	}
	return strings.Compare(a, b)
}
