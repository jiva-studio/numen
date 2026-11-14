package settings

import (
	"fmt"
	"maps"
	"slices"
	"strings"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/connection"
	"github.com/jiva-studio/numen/libs/plugin/catalog/manifest"
)

// RowConnectionKey is the key of the connection in a row of the value of a setting of rows. Every other key
// of a row is a row field of the type of the connection, held as a string.
const RowConnectionKey = "connection"

// rowIdentity is the connection and the options of a row, written in the order of their keys.
type rowIdentity struct{ connection, options string }

// RowIssue is a row of a setting of rows that is left out: its place in the list, from 1, and why.
type RowIssue struct {
	Index  int
	Reason string
}

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
			return fmt.Errorf("row %d: %w: the connection and its options are in a row before", index+1, ErrWrongType)
		}
		pairs[identity] = true
	}
	return nil
}

// FilterRows returns the rows of the list that the setting of rows takes, in list order, and an issue for
// each other row: a row CheckValue refuses, a row whose connection and options are in a row before, and
// each row beyond max.
func FilterRows(declared manifest.Setting, registries Registries, list []any) ([]any, []RowIssue) {
	connectionIDs, isKnown := listConnectionIDs(registries)
	remainder := make([]any, 0, len(list))
	var issues []RowIssue
	pairs := map[rowIdentity]bool{}
	for index, row := range list {
		identity, err := readRow(row, connectionIDs, isKnown)
		reason := ""
		switch {
		case err != nil:
			reason = err.Error()
		case pairs[identity]:
			reason = "the connection and its options are in a row before"
		case declared.Max != nil && float64(len(remainder)) >= *declared.Max:
			reason = fmt.Sprintf("more than %v rows", *declared.Max)
		}
		if reason != "" {
			issues = append(issues, RowIssue{Index: index + 1, Reason: strings.TrimSpace(reason)})
			continue
		}
		pairs[identity] = true
		remainder = append(remainder, row)
	}
	return remainder, issues
}

func listConnectionIDs(registries Registries) (map[string]bool, bool) {
	if registries == nil {
		return nil, false
	}
	ids, isFound := registries(RegistryConnections)
	if !isFound {
		return nil, false
	}
	detected, _ := registries(RegistryDetectedConnections)
	pending, _ := registries(RegistryPendingDetections)
	connectionIDs := make(map[string]bool, len(ids)+len(detected))
	for _, id := range slices.Concat(ids, detected) {
		connectionIDs[id] = true
	}
	for _, typeID := range pending {
		connectionIDs[typeID+pendingSuffix] = true
	}
	return connectionIDs, true
}

func readRow(row any, connectionIDs map[string]bool, isKnown bool) (rowIdentity, error) {
	object, isObject := row.(map[string]any)
	if !isObject {
		return rowIdentity{}, fmt.Errorf("%w: expected an object", ErrWrongType)
	}
	connectionID, err := readRowText(object, RowConnectionKey, true)
	if err != nil {
		return rowIdentity{}, err
	}
	var options strings.Builder
	for _, key := range slices.Sorted(maps.Keys(object)) {
		if key == RowConnectionKey {
			continue
		}
		text, err := readRowText(object, key, false)
		if err != nil {
			return rowIdentity{}, err
		}
		fmt.Fprintf(&options, "%q=%q;", key, text)
	}
	if isKnown && !isKnownConnection(connectionIDs, connectionID) {
		return rowIdentity{}, fmt.Errorf("%q is %w the connections", connectionID, ErrNotAnOption)
	}
	return rowIdentity{connection: connectionID, options: options.String()}, nil
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

func checkDiscover(settings []manifest.Setting, value any) error {
	if value == nil {
		return nil
	}
	choices, isObject := value.(map[string]any)
	if !isObject {
		return fmt.Errorf("%w: expected an object of true or false by the key of a setting of rows", ErrWrongType)
	}
	for _, key := range slices.Sorted(maps.Keys(choices)) {
		isDiscoverable := slices.ContainsFunc(settings, func(one manifest.Setting) bool {
			return one.Key == key && one.Type == manifest.SettingTypeConnections && one.Discover
		})
		if !isDiscoverable {
			return fmt.Errorf("%s: %w that discovers connections", key, ErrUndeclared)
		}
		if _, isBool := choices[key].(bool); !isBool {
			return fmt.Errorf("%s: %w: expected true or false", key, ErrWrongType)
		}
	}
	return nil
}

// pendingSuffix marks, in the set of known connections, a type whose detected connections are not known yet.
const pendingSuffix = "\x00pending"

func isKnownConnection(connectionIDs map[string]bool, id string) bool {
	if connectionIDs[id] {
		return true
	}
	typeID, _, isDetected := connection.SplitDetectedID(id)
	return isDetected && connectionIDs[typeID+pendingSuffix]
}
