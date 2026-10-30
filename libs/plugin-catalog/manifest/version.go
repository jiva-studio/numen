package manifest

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
)

// ErrInvalidAppVersion is a bound of the application versions a plugin runs in that is not a version.
var ErrInvalidAppVersion = errors.New("invalid application version")

// ErrInvertedAppVersion is a range of application versions whose minimum exceeds its maximum.
var ErrInvertedAppVersion = errors.New("minimum application version exceeds the maximum")

// AppVersionRange bounds the application versions a plugin runs in. Both bounds are inclusive and an
// empty bound is open.
type AppVersionRange struct {
	Min string
	Max string
}

// Locate returns -1 for a version below the range, 1 for one above it and 0 for one inside it. A
// version that is no version is -1.
func (r AppVersionRange) Locate(version string) int {
	if r.Min != "" {
		if order, err := CompareVersions(version, r.Min); err != nil || order < 0 {
			return -1
		}
	}
	if r.Max != "" {
		if order, err := CompareVersions(version, r.Max); err != nil || order > 0 {
			return 1
		}
	}
	return 0
}

// listAppVersionIssues checks that each bound is a version and that the minimum does not exceed the maximum.
func (m Manifest) listAppVersionIssues() []error {
	var issues []error
	limits := []struct{ name, value string }{{"min", m.AppVersion.Min}, {"max", m.AppVersion.Max}}
	for _, limit := range limits {
		if limit.value != "" && !manifestVersionPattern.MatchString(limit.value) {
			issues = append(issues, fmt.Errorf("%w: %s %q", ErrInvalidAppVersion, limit.name, limit.value))
		}
	}
	if len(issues) == 0 && m.AppVersion.Min != "" && m.AppVersion.Max != "" {
		if order, err := CompareVersions(m.AppVersion.Min, m.AppVersion.Max); err == nil && order > 0 {
			issues = append(issues, fmt.Errorf("%w: %s > %s", ErrInvertedAppVersion, m.AppVersion.Min, m.AppVersion.Max))
		}
	}
	return issues
}

// CompareVersions orders two semantic versions by precedence: -1 when a precedes b, 1 when b precedes a
// and 0 when they are equal. A leading v and build metadata are ignored.
func CompareVersions(a, b string) (int, error) {
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
	if !manifestVersionPattern.MatchString(text) {
		return version{numbers: [3]uint64{}, prerelease: nil}, fmt.Errorf("%w: %q", ErrInvalidVersion, text)
	}
	core := strings.TrimPrefix(text, "v")
	core, _, _ = strings.Cut(core, "+")
	core, prerelease, hasPrerelease := strings.Cut(core, "-")
	parsed := version{numbers: [3]uint64{}, prerelease: nil}
	for i, part := range strings.Split(core, ".") {
		number, err := strconv.ParseUint(part, 10, 64)
		if err != nil {
			return version{numbers: [3]uint64{}, prerelease: nil}, fmt.Errorf("%w: %q: %w", ErrInvalidVersion, text, err)
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

// comparePrerelease orders two prerelease lists: a version without one follows any with one, and a
// shorter list precedes a longer one it begins.
func comparePrerelease(a, b []string) int {
	switch {
	case len(a) == 0 && len(b) == 0:
		return 0
	case len(a) == 0:
		return 1
	case len(b) == 0:
		return -1
	}
	for i := 0; i < len(a) && i < len(b); i++ {
		if order := compareIdentifier(a[i], b[i]); order != 0 {
			return order
		}
	}
	return compareNumbers(uint64(len(a)), uint64(len(b)))
}

// compareIdentifier orders two prerelease identifiers: numeric ones by value and before alphanumeric
// ones, which are ordered by their characters.
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
