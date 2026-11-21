package connection

import (
	"github.com/jiva-studio/numen/libs/core/kernel/domain/connection"
)

// ListRows reads the rows a plugin calls an operation through.
type ListRows struct {
	operation
	settings RowSettings
}

// Execute returns the rows the person chose for the operation, in their order. Where the person chose none
// and the setting discovers, or the plugin is granted the operation, it returns a row with no options for
// each connection whose type is available and serves the operation, in the order of the list of
// connections, the detected ones last; a setting that is detected only gets the detected ones. A type whose provider has not reported what it detects yet is asked
// first; one that answered, in this start or an earlier one, is not asked.
func (u *ListRows) Execute(pluginID, operationName string) ([]connection.Row, error) {
	setting, err := u.settings.ReadRowSetting(pluginID, operationName)
	if err != nil {
		return nil, err
	}
	grants := connection.ListGrants(u.uses.ListPlugins())
	isDiscovering := setting.IsDiscovering || setting.IsDetectedOnly || connection.IsGranted(grants, pluginID, operationName)
	if len(setting.Rows) > 0 || !isDiscovering {
		return setting.Rows, nil
	}
	list, err := u.listConnectionsAfterDetection(u.listServingTypes(operationName))
	if err != nil {
		return nil, err
	}
	var rows []connection.Row
	for _, one := range list {
		if u.isDiscoverable(one, operationName, setting.IsDetectedOnly) {
			rows = append(rows, connection.Row{ConnectionID: one.ID, Options: map[string]string{}, IsDiscovered: true})
		}
	}
	return rows, nil
}

func (u *ListRows) isDiscoverable(one connection.Connection, operationName string, isDetectedOnly bool) bool {
	if isDetectedOnly && !connection.IsDetected(one.ID) {
		return false
	}
	found, isFound := u.types.GetType(one.TypeID)
	return isFound && found.IsAvailable && found.Declaration.HasOperation(operationName)
}

func (u *ListRows) listServingTypes(operationName string) []string {
	var typeIDs []string
	for _, one := range u.types.ListTypes() {
		if one.IsAvailable && one.Declaration.HasOperation(operationName) {
			typeIDs = append(typeIDs, one.GetID())
		}
	}
	return typeIDs
}
