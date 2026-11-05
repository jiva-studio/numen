package settings

import (
	"errors"

	"github.com/jiva-studio/numen/libs/core/port/pluginport"
)

// OwnerApp is the owner of the settings of the shell. Every other first-level key of the file
// other than the version is the id of a plugin.
const OwnerApp = "app"

// The reasons CheckWrite rejects a write.
var (
	// ErrUndeclared is a key that its owner does not declare, or an owner that has no declaration.
	ErrUndeclared = errors.New("the setting is not declared")
	// ErrWrongType is a value of another type than the declaration states.
	ErrWrongType = errors.New("the value is of the wrong type")
	// ErrNotAnOption is a value that is none of the options of an enum setting.
	ErrNotAnOption = errors.New("the value is none of the options")
	// ErrOutOfRange is a number outside the bounds of a range or a number setting.
	ErrOutOfRange = errors.New("the number is outside the bounds")
)

// Declarations returns the settings an owner declares, and false for an owner that has no declaration.
type Declarations func(owner string) ([]pluginport.PluginManifestSetting, bool)

// Registries returns the values a registry lists, such as the installed themes, and false for a
// registry that has no provider.
type Registries func(name string) ([]string, bool)

// CheckWrite takes the declarations of the owners, the registries, the path of the setting and the
// value. It returns why the value is not one the setting at the path takes. The first element of
// the path is the owner and the rest is the key path under it, as the file lays them out. One
// function checks a write for every owner: the declared key, the type of the value, the
// membership of an enum and the bounds of a range.
func CheckWrite(_ Declarations, _ Registries, _ []string, _ any) error {
	return nil
}

// GetAppDeclaration returns the settings of the owner app, declared in the format of a manifest
// setting. A key is the path under the owner, joined by dots.
func GetAppDeclaration() []pluginport.PluginManifestSetting {
	return nil
}

// GetOwner returns the values stored under an owner, the plugin id or OwnerApp.
func (File) GetOwner(_ string) map[string]any {
	return nil
}

// GetValue returns the value in force at a path: the stored value, else the default its owner
// declares. The first element of the path is the owner.
func (File) GetValue(_ ...string) (any, bool) {
	return nil, false
}
