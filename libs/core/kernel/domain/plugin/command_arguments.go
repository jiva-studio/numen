package plugin

import (
	"errors"
	"fmt"
	"maps"
	"slices"
	"strconv"

	"github.com/jiva-studio/numen/libs/plugin/catalog/manifest"
)

var (
	// ErrUnknownCommand is a command no enabled plugin declares.
	ErrUnknownCommand = errors.New("no enabled plugin declares that command")
	// ErrBadArguments is arguments or a file that do not meet the declaration.
	ErrBadArguments = errors.New("the arguments do not meet the command")
)

// CheckCommandArguments checks the arguments of the call against the declaration of the command: each
// is declared and of its type, and each required one is given. An argument that names a version is
// accepted where the command has items. A failure is ErrBadArguments.
func CheckCommandArguments(declared manifest.Command, call CommandCall, hasItems bool) error {
	for _, name := range slices.Sorted(maps.Keys(call.Arguments)) {
		if name == manifest.CommandArgumentVersion && hasItems {
			continue
		}
		argument, isDeclared := declared.GetArgument(name)
		if !isDeclared {
			return fmt.Errorf("%w: %s takes no argument %q", ErrBadArguments, declared.ID, name)
		}
		if !isOfType(argument.Type, call.Arguments[name]) {
			return fmt.Errorf("%w: %s is a %s, and %q is none", ErrBadArguments, argument.Name, argument.Type, call.Arguments[name])
		}
	}
	for _, argument := range declared.Arguments {
		if _, isGiven := call.Arguments[argument.Name]; argument.IsRequired && !isGiven {
			return fmt.Errorf("%w: %s needs %s", ErrBadArguments, declared.ID, argument.Name)
		}
	}
	return nil
}

// NormalizeCommandArguments returns the arguments with each boolean one written as true or false.
func NormalizeCommandArguments(declared manifest.Command, arguments map[string]string) map[string]string {
	normalized := make(map[string]string, len(arguments))
	for name, value := range arguments {
		if argument, _ := declared.GetArgument(name); argument.Type == manifest.CommandArgumentBoolean {
			parsed, err := strconv.ParseBool(value)
			value = strconv.FormatBool(err == nil && parsed)
		}
		normalized[name] = value
	}
	return normalized
}

func isOfType(argumentType, value string) bool {
	switch argumentType {
	case manifest.CommandArgumentNumber:
		_, err := strconv.ParseFloat(value, 64)
		return err == nil
	case manifest.CommandArgumentBoolean:
		_, err := strconv.ParseBool(value)
		return err == nil
	}
	return true
}
