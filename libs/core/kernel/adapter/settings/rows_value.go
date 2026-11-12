package settings

import (
	"fmt"
	"maps"
	"slices"
	"strings"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/connection"
	"github.com/jiva-studio/numen/libs/plugin/catalog/manifest"
)

// The keys of a row in the value of a setting of rows.
const (
	rowConnection = "connection"
	rowModel      = "model"
	rowReasoning  = "reasoning"
)

type rowIdentity struct{ connection, model string }

func checkRows(declared manifest.Setting, registries Registries, value any) error {
	list, isList := value.([]any)
	if !isList {
		return fmt.Errorf("%w: expected a list of rows", ErrWrongType)
	}
	if declared.Max != nil && float64(len(list)) > *declared.Max {
		return fmt.Errorf("%d rows are %w 0..%v", len(list), ErrOutOfRange, *declared.Max)
	}
	connectionIDs, isKnown := listConnectionIDs(registries)
	pairs := map[rowIdentity]bool{}
	for index, row := range list {
		identity, err := readRow(row, connectionIDs, isKnown)
		if err != nil {
			return fmt.Errorf("row %d: %w", index+1, err)
		}
		if pairs[identity] {
			return fmt.Errorf("row %d: %w: the connection and the model are in a row before", index+1, ErrWrongType)
		}
		pairs[identity] = true
	}
	return nil
}

func listConnectionIDs(registries Registries) (map[string]bool, bool) {
	if registries == nil {
		return nil, false
	}
	ids, isFound := registries(RegistryConnections)
	if !isFound {
		return nil, false
	}
	connectionIDs := make(map[string]bool, len(ids))
	for _, id := range ids {
		connectionIDs[id] = true
	}
	return connectionIDs, true
}

func readRow(row any, connectionIDs map[string]bool, isKnown bool) (rowIdentity, error) {
	object, isObject := row.(map[string]any)
	if !isObject {
		return rowIdentity{}, fmt.Errorf("%w: expected an object", ErrWrongType)
	}
	for _, key := range slices.Sorted(maps.Keys(object)) {
		if key != rowConnection && key != rowModel && key != rowReasoning {
			return rowIdentity{}, fmt.Errorf("%w: %q is no key of a row", ErrWrongType, key)
		}
	}
	connection, err := readRowText(object, rowConnection, true)
	if err != nil {
		return rowIdentity{}, err
	}
	model, err := readRowText(object, rowModel, true)
	if err != nil {
		return rowIdentity{}, err
	}
	if _, err := readRowText(object, rowReasoning, false); err != nil {
		return rowIdentity{}, err
	}
	if isKnown && !connectionIDs[connection] {
		return rowIdentity{}, fmt.Errorf("%q is %w the connections", connection, ErrNotAnOption)
	}
	return rowIdentity{connection: connection, model: model}, nil
}

func readRowText(object map[string]any, key string, isRequired bool) (string, error) {
	value, isSet := object[key]
	if !isSet {
		if isRequired {
			return "", fmt.Errorf("%w: %s is missing", ErrWrongType, key)
		}
		return "", nil
	}
	text, isText := value.(string)
	if !isText {
		return "", fmt.Errorf("%w: %s is no string", ErrWrongType, key)
	}
	if isRequired && text == "" {
		return "", fmt.Errorf("%w: %s is empty", ErrWrongType, key)
	}
	return text, nil
}

func (f *File) dropInvalidRows(owner string, declared manifest.Setting) {
	list, isList := f.Plugins[owner][declared.Key].([]any)
	if !isList {
		return
	}
	connectionIDs := make(map[string]bool, len(f.Connections))
	for _, one := range f.Connections {
		connectionIDs[one.ID] = true
	}
	remainder := make([]any, 0, len(list))
	pairs := map[rowIdentity]bool{}
	for index, row := range list {
		identity, err := readRow(row, connectionIDs, true)
		reason := ""
		switch {
		case err != nil:
			reason = err.Error()
		case pairs[identity]:
			reason = "the connection and the model are in a row before"
		case declared.Max != nil && float64(len(remainder)) >= *declared.Max:
			reason = fmt.Sprintf("more than %v rows", *declared.Max)
		}
		if reason != "" {
			f.AddNotice(fmt.Sprintf("numen.json: %s.%s row %d: %s; the row is ignored",
				owner, declared.Key, index+1, strings.TrimSpace(reason)))
			continue
		}
		pairs[identity] = true
		remainder = append(remainder, row)
	}
	f.Plugins[owner][declared.Key] = remainder
}

func withConnectionIDs(connections []connection.Connection, registries Registries) Registries {
	ids := make([]string, 0, len(connections))
	for _, one := range connections {
		ids = append(ids, one.ID)
	}
	return func(name string) ([]string, bool) {
		if name == RegistryConnections {
			return ids, true
		}
		if registries == nil {
			return nil, false
		}
		return registries(name)
	}
}

func dropInvalidRowsOf(f *File, owner string, settings []manifest.Setting) {
	for _, one := range settings {
		if one.Type == manifest.SettingTypeConnections {
			f.dropInvalidRows(owner, one)
		}
	}
}
