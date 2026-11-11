package plugin

import (
	"strings"
	"unicode/utf8"
)

// DeclarationKey names one declaration of a plugin: the plugin, the name the plugin reads the
// secret under, and the group it declares, empty for none. A declaration is identified by the name
// and the group together.
type DeclarationKey struct {
	Plugin string
	Name   string
	Group  string
}

// Secret is one named secret. Its value is kept outside the settings file and is not part of it.
type Secret struct {
	ID      string
	Name    string
	Group   string
	Allowed []DeclarationKey
}

// SecretSource is where the host reads the list of secrets from and how it removes the permissions
// of declarations that are gone.
type SecretSource struct {
	List  func() ([]Secret, error)
	Prune func(declared []DeclarationKey) error
}

// MaxSecretNameLength is the longest name of a secret or a group, in characters.
const MaxSecretNameLength = 64

// IsValidSecretName reports whether the text is a name of a secret or a group: not empty, without
// leading or trailing space, and at most MaxSecretNameLength characters.
func IsValidSecretName(name string) bool {
	return name != "" && strings.TrimSpace(name) == name && utf8.RuneCountInString(name) <= MaxSecretNameLength
}

const secretIDLength = 9

// IsValidSecretID reports whether the text has the shape of the id of a secret: "s" and 8
// hexadecimal digits.
func IsValidSecretID(id string) bool {
	if len(id) != secretIDLength || id[0] != 's' {
		return false
	}
	for _, digit := range id[1:] {
		if !strings.ContainsRune("0123456789abcdef", digit) {
			return false
		}
	}
	return true
}

// GetSecretEntryName returns the name the value of a secret is stored under in the key store, which the
// host reads it by when it sends the value to a plugin.
func GetSecretEntryName(id string) string { return "secret/" + id }
