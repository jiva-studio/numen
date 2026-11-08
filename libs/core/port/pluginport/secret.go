package pluginport

import (
	"context"
	"strings"
	"unicode/utf8"
)

// SecretPermission names one declaration of a plugin: the plugin, the name the plugin reads the
// secret under, and the group it declares, empty for none. A declaration is identified by the name
// and the group together.
type SecretPermission struct {
	Plugin string
	Name   string
	Group  string
}

// Secret is one named secret. Its value is kept outside the settings file and is not part of it.
type Secret struct {
	ID      string
	Name    string
	Group   string
	Allowed []SecretPermission
}

// SecretLedger is the ordered list of secrets kept in the settings file.
type SecretLedger interface {
	// GetSecrets returns the valid secrets in list order.
	GetSecrets() ([]Secret, error)
	// ChangeSecrets reads the list, applies change and writes the result as one step. Nothing is
	// written when change fails or leaves the list as it was.
	ChangeSecrets(change func([]Secret) ([]Secret, error)) error
}

// SecretPlugins is what the use cases on secrets need from the plugin host.
type SecretPlugins interface {
	// ListDeclarations returns every secret declaration of every installed plugin.
	ListDeclarations() []SecretPermission
	// ApplySecretChange runs change while no other secret operation runs, and then configures the
	// plugins whose secrets changed. change returns the ids of the plugins to configure although
	// their list of secrets did not change.
	ApplySecretChange(ctx context.Context, change func() (readers []string, err error)) error
}

// SecretSource is where the host reads the list of secrets from and how it removes the permissions
// of declarations that are gone.
type SecretSource struct {
	List  func() ([]Secret, error)
	Prune func(declared []SecretPermission) error
}

// MaxSecretNameLength is the longest name of a secret or a group, in characters.
const MaxSecretNameLength = 64

// IsValidSecretName reports whether the text is a name of a secret or a group: not empty, without
// leading or trailing space, and at most MaxSecretNameLength characters.
func IsValidSecretName(name string) bool {
	return name != "" && strings.TrimSpace(name) == name && utf8.RuneCountInString(name) <= MaxSecretNameLength
}

// secretIDLength is the length of the id of a secret.
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
