package connection

import (
	"strings"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/plugin"
	"github.com/jiva-studio/numen/libs/plugin/catalog/manifest"
)

// Rejection is a stored connection that is left out of force, and why.
type Rejection struct {
	ID     string
	Reason string
}

// FilterValid returns the connections of the list that are valid, in list order, and a rejection for each
// other one: a bad or repeated id, a bad name or one repeated ignoring case, a type id without a plugin id
// and a local id, or a revision below 1. A type that is not installed and a field the type does not declare
// are not invalid.
func FilterValid(list []Connection) ([]Connection, []Rejection) {
	var kept []Connection
	var rejections []Rejection
	seenIDs := map[string]bool{}
	seenNames := map[string]bool{}
	for _, one := range list {
		if reason := findIssue(one, seenIDs, seenNames); reason != "" {
			rejections = append(rejections, Rejection{ID: one.ID, Reason: reason})
			continue
		}
		seenIDs[one.ID] = true
		seenNames[strings.ToLower(one.Name)] = true
		kept = append(kept, one)
	}
	return kept, rejections
}

func findIssue(one Connection, seenIDs, seenNames map[string]bool) string {
	switch {
	case !IsValidID(one.ID):
		return "the id has the wrong shape"
	case seenIDs[one.ID]:
		return "the id is used twice"
	case !IsValidName(one.Name):
		return "the name has the wrong shape"
	case seenNames[strings.ToLower(one.Name)]:
		return "the name is used twice"
	case !isValidTypeID(one.TypeID):
		return "the type has the wrong shape"
	case one.Revision < 1:
		return "the revision is below 1"
	}
	return ""
}

func isValidTypeID(typeID string) bool {
	_, _, isValid := manifest.SplitConnectionTypeID(typeID)
	return isValid
}

// ListGrants returns the operations each plugin names in its permissions connections:use:<operation>, in
// the order of the plugins. A plugin that names none is left out.
func ListGrants(plugins []plugin.Info) []Grant {
	var grants []Grant
	for _, one := range plugins {
		var operations []string
		for _, permission := range one.Permissions {
			if operation, isGrant := manifest.ParseConnectionsUse(permission); isGrant {
				operations = append(operations, operation)
			}
		}
		if len(operations) > 0 {
			grants = append(grants, Grant{PluginID: one.ID, PluginName: one.Name, Operations: operations})
		}
	}
	return grants
}
