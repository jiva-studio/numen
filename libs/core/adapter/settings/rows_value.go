package settings

import (
	"fmt"
	"maps"
	"slices"
	"strings"

	"github.com/jiva-studio/numen/libs/core/port/pluginport"
)

// The keys of a row in the value of a setting of rows.
const (
	rowConnection = "connection"
	rowModel      = "model"
	rowReasoning  = "reasoning"
)

type rowIdentity struct{ connection, model string }

// checkRows returns why a value is not the rows a setting of type connections takes: a list of at
// most max objects that name an existing connection and a model, and no pair of connection and
// model twice. A registry that lists no connections leaves the connection unchecked.
func checkRows(declared pluginport.PluginManifestSetting, registries Registries, value any) error {
	list, isList := value.([]any)
	if !isList {
		return fmt.Errorf("%w: expected a list of rows", ErrWrongType)
	}
	if declared.Max != nil && float64(len(list)) > *declared.Max {
		return fmt.Errorf("%d rows are %w 0..%v", len(list), ErrOutOfRange, *declared.Max)
	}
	known, isKnown := listConnectionIDs(registries)
	seen := map[rowIdentity]bool{}
	for index, row := range list {
		identity, err := readRow(row, known, isKnown)
		if err != nil {
			return fmt.Errorf("row %d: %w", index+1, err)
		}
		if seen[identity] {
			return fmt.Errorf("row %d: %w: the connection and the model are in a row before", index+1, ErrWrongType)
		}
		seen[identity] = true
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
	known := make(map[string]bool, len(ids))
	for _, id := range ids {
		known[id] = true
	}
	return known, true
}

func readRow(row any, known map[string]bool, isKnown bool) (rowIdentity, error) {
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
	if isKnown && !known[connection] {
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

// dropInvalidRows leaves out of force each row of a setting of rows that checkRows would refuse,
// and each row beyond max, with a notice each. A value that is no list is left to the check of the
// whole value.
func (f *File) dropInvalidRows(owner string, declared pluginport.PluginManifestSetting) {
	list, isList := f.Plugins[owner][declared.Key].([]any)
	if !isList {
		return
	}
	known := make(map[string]bool, len(f.Connections))
	for _, one := range f.Connections {
		known[one.ID] = true
	}
	kept := make([]any, 0, len(list))
	seen := map[rowIdentity]bool{}
	for index, row := range list {
		identity, err := readRow(row, known, true)
		reason := ""
		switch {
		case err != nil:
			reason = err.Error()
		case seen[identity]:
			reason = "the connection and the model are in a row before"
		case declared.Max != nil && float64(len(kept)) >= *declared.Max:
			reason = fmt.Sprintf("more than %v rows", *declared.Max)
		}
		if reason != "" {
			f.AddNotice(fmt.Sprintf("numen.json: %s.%s row %d: %s; the row is ignored",
				owner, declared.Key, index+1, strings.TrimSpace(reason)))
			continue
		}
		seen[identity] = true
		kept = append(kept, row)
	}
	f.Plugins[owner][declared.Key] = kept
}

// withConnectionIDs answers the registry of connections from the connections of the file and every
// other registry from registries.
func (f *File) withConnectionIDs(registries Registries) Registries {
	ids := make([]string, 0, len(f.Connections))
	for _, one := range f.Connections {
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
