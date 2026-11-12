package manifest

import (
	"fmt"
	"math"
	"regexp"
	"slices"
	"strings"
)

const (
	// ConnectionFieldSecret is the type of a connection field whose value is kept in the key store.
	ConnectionFieldSecret = "secret"

	// ConnectionParallelRequests is the key of the field the core reads as the limit of calls that
	// run at once on a connection.
	ConnectionParallelRequests = "parallelRequests"

	connectionTypeIDSeparator = "/"
)

var (
	connectionTypeIDPattern    = regexp.MustCompile(`^[a-z][a-z0-9]*(-[a-z0-9]+)*$`)
	connectionOperationPattern = regexp.MustCompile(`^[a-z][a-z0-9]*(\.[a-z][a-z0-9]*)*$`)
	connectionFieldKeyPattern  = regexp.MustCompile(`^[A-Za-z][A-Za-z0-9]*$`)
)

// ConnectionType is a kind of connection a plugin provides. ID is the local part of the type id, and
// Operations are the names of the operations the type serves. RowFields are the values a row that names
// a connection of the type holds besides the connection. IsBuiltIn makes the core list one connection of the
// type while the plugin is enabled, with no field set; only a type with no required field that is not fixed
// declares it.
type ConnectionType struct {
	ID          string
	Title       string
	Description string
	Operations  []string
	Fields      []ConnectionField
	RowFields   []ConnectionField
	IsBuiltIn   bool
}

// ConnectionField is one value a connection of a type holds. FixedValue is held by every connection
// of the type and is neither stored nor edited. IsAddress marks the field whose change clears the
// secret fields of the connection. OptionsFrom names the operation of the type that lists the values
// of a row field.
type ConnectionField struct {
	Key         string
	Type        string
	Title       string
	Description string
	Default     any
	FixedValue  *string
	IsRequired  bool
	IsAddress   bool
	Options     SettingOptions
	Min         *float64
	Max         *float64
	Step        *float64
	OptionsFrom string
}

func (t ConnectionType) GetField(key string) (ConnectionField, bool) {
	for _, one := range t.Fields {
		if one.Key == key {
			return one, true
		}
	}
	return ConnectionField{}, false
}

func (t ConnectionType) GetRowField(key string) (ConnectionField, bool) {
	for _, one := range t.RowFields {
		if one.Key == key {
			return one, true
		}
	}
	return ConnectionField{}, false
}

// HasOperation reports whether the type serves the operation.
func (t ConnectionType) HasOperation(operation string) bool {
	return slices.Contains(t.Operations, operation)
}

func (c Contributes) GetConnectionType(id string) (ConnectionType, bool) {
	for _, one := range c.ConnectionTypes {
		if one.ID == id {
			return one, true
		}
	}
	return ConnectionType{}, false
}

// GetConnectionTypeID returns the id of a type everywhere outside the manifest: the plugin id, a
// slash and the local id.
func GetConnectionTypeID(pluginID, typeID string) string {
	return pluginID + connectionTypeIDSeparator + typeID
}

// SplitConnectionTypeID splits the id of a type at its first slash. A plugin id has no slash.
func SplitConnectionTypeID(typeID string) (pluginID, id string, isValid bool) {
	pluginID, id, found := strings.Cut(typeID, connectionTypeIDSeparator)
	if !found || pluginID == "" || id == "" || strings.Contains(id, connectionTypeIDSeparator) {
		return "", "", false
	}
	return pluginID, id, true
}

func reservedConnectionFieldKeys() []string { return []string{"type", "name", "id"} }

func connectionFieldTypes() []string {
	return []string{SettingTypeString, settingTypeNumber, settingTypeBoolean, SettingTypeEnum, SettingTypeFile, ConnectionFieldSecret}
}

func (m Manifest) listConnectionTypeIssues() []error {
	var issues []error
	observed := map[string]bool{}
	for _, one := range m.Contributes.ConnectionTypes {
		switch {
		case !connectionTypeIDPattern.MatchString(one.ID):
			issues = append(issues, newConnectionIssue("type id %q is not lowercase words joined by hyphens", one.ID))
		case observed[one.ID]:
			issues = append(issues, newConnectionIssue("type %q is declared twice", one.ID))
		}
		observed[one.ID] = true
		if strings.TrimSpace(one.Title) == "" {
			issues = append(issues, newConnectionIssue("type %q has no title", one.ID))
		}
		issues = append(issues, listConnectionOperationIssues(one)...)
		issues = append(issues, listConnectionFieldIssues(one)...)
		issues = append(issues, listRowFieldIssues(one)...)
		issues = append(issues, listBuiltInIssues(one)...)
	}
	return issues
}

func listBuiltInIssues(t ConnectionType) []error {
	if !t.IsBuiltIn {
		return nil
	}
	var issues []error
	for _, field := range t.Fields {
		if field.IsRequired && field.FixedValue == nil {
			issues = append(issues, newFieldIssue(t.ID, field, "is required and the type is built in, so no one sets it"))
		}
	}
	return issues
}

func newConnectionIssue(format string, args ...any) error {
	return fmt.Errorf("%w: connection %s", ErrInvalidContribution, fmt.Sprintf(format, args...))
}

func listConnectionOperationIssues(t ConnectionType) []error {
	if len(t.Operations) == 0 {
		return []error{newConnectionIssue("type %q declares no operation", t.ID)}
	}
	var issues []error
	for index, operation := range t.Operations {
		switch {
		case !connectionOperationPattern.MatchString(operation):
			issues = append(issues, newConnectionIssue("type %q: operation %q is not lowercase words joined by dots", t.ID, operation))
		case slices.Index(t.Operations, operation) != index:
			issues = append(issues, newConnectionIssue("type %q: operation %q is listed twice", t.ID, operation))
		}
	}
	return issues
}

func listConnectionFieldIssues(t ConnectionType) []error {
	var issues []error
	observed := map[string]bool{}
	addresses := 0
	for _, field := range t.Fields {
		switch {
		case !connectionFieldKeyPattern.MatchString(field.Key):
			issues = append(issues, newConnectionIssue("type %q: field key %q is not letters and digits starting with a letter", t.ID, field.Key))
		case slices.Contains(reservedConnectionFieldKeys(), field.Key):
			issues = append(issues, newConnectionIssue("type %q: field key %q is reserved", t.ID, field.Key))
		case observed[field.Key]:
			issues = append(issues, newConnectionIssue("type %q: field %q is declared twice", t.ID, field.Key))
		}
		observed[field.Key] = true
		if field.IsAddress {
			addresses++
		}
		if field.OptionsFrom != "" {
			issues = append(issues, newFieldIssue(t.ID, field, "names optionsFrom and is no row field"))
		}
		issues = append(issues, listConnectionFieldShapeIssues(t.ID, field)...)
	}
	if addresses > 1 {
		issues = append(issues, newConnectionIssue("type %q marks %d fields as the address, at most one", t.ID, addresses))
	}
	return issues
}

func listConnectionFieldShapeIssues(typeID string, field ConnectionField) []error {
	if !slices.Contains(connectionFieldTypes(), field.Type) {
		return []error{newFieldIssue(typeID, field, "has type %q, which is not a field type", field.Type)}
	}
	issues := listConnectionFieldKindIssues(typeID, field)
	issues = append(issues, listConnectionFieldValueIssues(typeID, field)...)
	if field.Key == ConnectionParallelRequests {
		issues = append(issues, listParallelRequestsIssues(typeID, field)...)
	}
	return issues
}

func newFieldIssue(typeID string, field ConnectionField, format string, args ...any) error {
	return newConnectionIssue("type %q: field %q %s", typeID, field.Key, fmt.Sprintf(format, args...))
}

func listConnectionFieldKindIssues(typeID string, field ConnectionField) []error {
	var issues []error
	if field.Type == SettingTypeEnum && len(field.Options) == 0 {
		issues = append(issues, newFieldIssue(typeID, field, "is an enum with no options"))
	}
	if field.Type == ConnectionFieldSecret && (field.Default != nil || field.FixedValue != nil || field.IsAddress || len(field.Options) > 0) {
		issues = append(issues, newFieldIssue(typeID, field, "is a secret and has a default, a fixed value, options or the address mark"))
	}
	return issues
}

func listConnectionFieldValueIssues(typeID string, field ConnectionField) []error {
	var issues []error
	if field.FixedValue != nil && (field.Type != SettingTypeString || field.IsRequired || field.Default != nil) {
		issues = append(issues, newFieldIssue(typeID, field, "has a fixed value and is not an optional string with no default"))
	}
	if field.IsAddress && field.Type != SettingTypeString {
		issues = append(issues, newFieldIssue(typeID, field, "is the address and is not a string"))
	}
	return issues
}

func listParallelRequestsIssues(typeID string, field ConnectionField) []error {
	if field.Type != settingTypeNumber || field.Min == nil || *field.Min < 1 || field.Step == nil || *field.Step != 1 {
		return []error{newConnectionIssue("type %q: %s is a number with a min of at least 1 and a step of 1", typeID, ConnectionParallelRequests)}
	}
	if field.Default == nil {
		return nil
	}
	value, isNumber := field.Default.(float64)
	if !isNumber || value < *field.Min || value != math.Trunc(value) {
		return []error{newConnectionIssue("type %q: the default of %s is a whole number of at least min", typeID, ConnectionParallelRequests)}
	}
	return nil
}

// rowFieldConnection is the key a row stores its connection under.
const rowFieldConnection = "connection"

func listRowFieldKeyIssues(typeID, key string, observed map[string]bool) []error {
	switch {
	case !connectionFieldKeyPattern.MatchString(key):
		return []error{newConnectionIssue("type %q: row field key %q is not letters and digits starting with a letter", typeID, key)}
	case key == rowFieldConnection:
		return []error{newConnectionIssue("type %q: row field key %q is reserved", typeID, key)}
	case observed[key]:
		return []error{newConnectionIssue("type %q: row field %q is declared twice", typeID, key)}
	}
	return nil
}

func listRowFieldIssues(t ConnectionType) []error {
	var issues []error
	observed := map[string]bool{}
	for _, field := range t.RowFields {
		issues = append(issues, listRowFieldKeyIssues(t.ID, field.Key, observed)...)
		observed[field.Key] = true
		if field.Type != SettingTypeString && field.Type != SettingTypeEnum {
			issues = append(issues, newFieldIssue(t.ID, field, "is a row field and is not a string or an enum"))
		}
		if field.Type == SettingTypeEnum && len(field.Options) == 0 && field.OptionsFrom == "" {
			issues = append(issues, newFieldIssue(t.ID, field, "is an enum with neither options nor optionsFrom"))
		}
		if field.FixedValue != nil || field.IsAddress {
			issues = append(issues, newFieldIssue(t.ID, field, "is a row field and has a fixed value or the address mark"))
		}
		if field.OptionsFrom != "" && !t.HasOperation(field.OptionsFrom) {
			issues = append(issues, newFieldIssue(t.ID, field, "lists its options from %q, which the type does not serve", field.OptionsFrom))
		}
	}
	return issues
}
