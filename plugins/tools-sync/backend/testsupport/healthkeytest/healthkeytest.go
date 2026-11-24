// Package healthkeytest tells whether the host takes the key of a health item.
package healthkeytest

import (
	"regexp"

	"github.com/jiva-studio/numen/plugins/tools-sync/backend/domain/healthkey"
)

var pattern = regexp.MustCompile(`^[a-z][a-z0-9-]*$`)

// IsValid reports whether the host takes the key: lowercase letters, digits and hyphens starting with a
// letter, at most healthkey.MaxLength characters.
func IsValid(key string) bool {
	return len(key) <= healthkey.MaxLength && pattern.MatchString(key)
}
