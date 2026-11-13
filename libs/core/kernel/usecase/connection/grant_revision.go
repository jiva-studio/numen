package connection

import (
	"slices"
	"strconv"
	"strings"
)

// GetGrantRevision names the connections a plugin reaches through its grants as they are now. A list the
// plugin answered from them is stored under this revision and is read again once the revision changes.
type GetGrantRevision struct{ operation }

// NewGetGrantRevision returns the use case that names the connections a plugin reaches through its grants.
func NewGetGrantRevision(deps Deps) *GetGrantRevision { return &GetGrantRevision{operation: deps.base} }

// Execute returns the ids and revisions of the connections whose type is available and serves an operation
// the plugin is granted, in the order of the list, and false where the plugin is granted none. Equal
// revisions are the same connections unchanged.
func (u *GetGrantRevision) Execute(pluginID string) (string, bool, error) {
	var operations []string
	for _, one := range u.uses.ListGrants() {
		if one.PluginID == pluginID {
			operations = append(operations, one.Operations...)
		}
	}
	if len(operations) == 0 {
		return "", false, nil
	}
	list, err := u.listConnections()
	if err != nil {
		return "", true, err
	}
	var parts []string
	for _, one := range list {
		found, isFound := u.types.GetType(one.TypeID)
		if isFound && found.IsAvailable && slices.ContainsFunc(operations, found.Declaration.HasOperation) {
			parts = append(parts, one.ID+"@"+strconv.Itoa(one.Revision))
		}
	}
	return strings.Join(parts, ","), true, nil
}
