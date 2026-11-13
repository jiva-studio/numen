package connection

import (
	"github.com/jiva-studio/numen/libs/core/kernel/domain/connection"
	"github.com/jiva-studio/numen/libs/core/kernel/port/connectionport"
)

// ListRows reads the rows a plugin calls an operation through.
type ListRows struct {
	operation
	settings connectionport.RowSettings
}

// Execute returns the rows the person chose for the operation, in their order. Where the person chose none
// and the setting discovers, or the plugin is granted the operation, it returns a row with no options for
// each connection whose type is available and serves the operation, in the order of the list of
// connections, the detected ones last. A type whose provider has not reported what it detects yet is asked
// first; one that answered, in this start or an earlier one, is not asked.
func (u *ListRows) Execute(pluginID, operationName string) ([]connection.Row, error) {
	setting, err := u.settings.ReadRowSetting(pluginID, operationName)
	if err != nil {
		return nil, err
	}
	isDiscovering := setting.IsDiscovering || connection.IsGranted(u.uses.ListGrants(), pluginID, operationName)
	if len(setting.Rows) > 0 || !isDiscovering {
		return setting.Rows, nil
	}
	list, err := u.listConnectionsAfterDetection(u.listServingTypes(operationName))
	if err != nil {
		return nil, err
	}
	var rows []connection.Row
	for _, one := range list {
		found, isFound := u.types.GetType(one.TypeID)
		if isFound && found.IsAvailable && found.Declaration.HasOperation(operationName) {
			rows = append(rows, connection.Row{ConnectionID: one.ID, Options: map[string]string{}, IsDiscovered: true})
		}
	}
	return rows, nil
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
