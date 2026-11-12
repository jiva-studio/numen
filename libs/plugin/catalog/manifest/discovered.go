package manifest

import (
	"fmt"
	"strings"
)

// FileName is the name of the manifest file in the folder of a plugin.
const FileName = "manifest.json"

// Discovered is a validated plugin located on disk.
type Discovered struct {
	Dir      string
	Manifest Manifest
}

// FilterByTarget returns only plugins that can run in the target workspace.
func FilterByTarget(plugins []Discovered, target string) []Discovered {
	var matched []Discovered
	for _, p := range plugins {
		if p.Manifest.Supports(target) {
			matched = append(matched, p)
		}
	}
	return matched
}

// DuplicateContributionError is an id that a plugin declares and an earlier plugin already declares: a kind of
// tab or a reader. Only one plugin owns each, and the earlier one keeps it.
type DuplicateContributionError struct {
	PluginID string
	OwnerID  string
	What     string
	ID       string
}

// Error names the plugin that is hasConflict, the plugin that owns the id, and the id.
func (c DuplicateContributionError) Error() string {
	return fmt.Sprintf("%v: plugin %q declares the %s %q, which plugin %q already declares", ErrInvalidContribution, c.PluginID, c.What, c.ID, c.OwnerID)
}

// Unwrap makes the conflict an ErrInvalidContribution.
func (c DuplicateContributionError) Unwrap() error { return ErrInvalidContribution }

// readerTabKindPrefix is how the window names the tab kind of a reader.
const readerTabKindPrefix = "reader:"

// ListContributionConflicts lists the declarations of tab kinds and readers that a
// plugin makes after an earlier plugin of the list already made them. The reader of an id and the
// tab kind of the name the window gives that reader are one claim, which one plugin may make as both. A plugin with a conflict owns
// nothing, because the host refuses it.
func ListContributionConflicts(plugins []Discovered) []DuplicateContributionError {
	owners := map[string]string{}
	var conflicts []DuplicateContributionError
	for _, p := range plugins {
		pending := map[string]bool{}
		hasConflict := false
		claim := func(what, space, id string) {
			key := space + "\x00" + id
			if owner, isOwned := owners[key]; isOwned {
				conflicts = append(conflicts, DuplicateContributionError{PluginID: p.Manifest.ID, OwnerID: owner, What: what, ID: id})
				hasConflict = true
				return
			}
			pending[key] = true
		}
		c := p.Manifest.Contributes
		for _, kind := range c.TabKinds {
			claim("tab kind", "tab", kind.Kind)
		}
		for _, reader := range c.Readers {
			claim("reader", "tab", getReaderTabKind(reader.ID))
		}
		if hasConflict {
			continue
		}
		for key := range pending {
			owners[key] = p.Manifest.ID
		}
	}
	return conflicts
}

func getReaderTabKind(id string) string {
	if strings.HasPrefix(id, readerTabKindPrefix) {
		return id
	}
	return readerTabKindPrefix + id
}
