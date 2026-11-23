package manifest

import (
	"fmt"
	"slices"
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
		if p.Manifest.CanTarget(target) {
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

// CoreOwnerID is the owner a conflict names for a contribution the core makes itself.
const CoreOwnerID = "numen"

// searchFunctionSpace is the space of the names of query functions.
const searchFunctionSpace = "searchFunction"

// viewCapabilitySpace is the space of the application view kinds: one plugin draws each.
const viewCapabilitySpace = "viewCapability"

// slotPointSpace and submenuSpace are the spaces of the names a slot goes by: a slot point and a submenu
// share one, so no plugin gives a submenu the name of a slot point another declares.
const (
	slotPointSpace = "slotPoint"
	submenuSpace   = "submenu"
)

// CoreClaims are the contributions the core makes itself: the names of its built-in query functions.
type CoreClaims struct {
	SearchFunctions []string
}

// ListContributionConflicts lists the declarations of tab kinds, readers, application view kinds, search functions and
// license providers that a plugin makes after the core or an earlier plugin of the list already made
// them. The reader of an id and the tab kind of the name the window gives that reader are one claim,
// which one plugin may make as both. Two plugins may contribute to one license id unless they give
// the same provider different productIds or urls. A plugin with a conflict owns nothing, because the
// host refuses it.
func ListContributionConflicts(plugins []Discovered, core CoreClaims) []DuplicateContributionError {
	owners := map[string]string{}
	for _, name := range core.SearchFunctions {
		owners[searchFunctionSpace+"\x00"+name] = CoreOwnerID
	}
	licenseProviders := map[string]licenseProviderClaim{}
	var conflicts []DuplicateContributionError
	for _, p := range plugins {
		c := p.Manifest.Contributes
		pending, pluginConflicts := collectPluginContributionClaims(p.Manifest.ID, c, owners)
		pluginConflicts = append(pluginConflicts, listLicenseProviderConflicts(p.Manifest.ID, c.Licenses, licenseProviders)...)
		if len(pluginConflicts) > 0 {
			conflicts = append(conflicts, pluginConflicts...)
			continue
		}
		for key := range pending {
			owners[key] = p.Manifest.ID
		}
		recordLicenseProviders(p.Manifest.ID, c.Licenses, licenseProviders)
	}
	return conflicts
}

func collectPluginContributionClaims(
	pluginID string,
	c Contributes,
	owners map[string]string,
) (map[string]bool, []DuplicateContributionError) {
	pending := map[string]bool{}
	var conflicts []DuplicateContributionError
	claim := func(what, space, id string) {
		key := space + "\x00" + id
		if owner, isOwned := owners[key]; isOwned {
			conflicts = append(conflicts, DuplicateContributionError{PluginID: pluginID, OwnerID: owner, What: what, ID: id})
			return
		}
		pending[key] = true
	}
	for _, kind := range c.TabKinds {
		claim("tab kind", "tab", kind.Kind)
	}
	for _, reader := range c.Readers {
		claim("reader", "tab", getReaderTabKind(reader.ID))
	}
	for _, capability := range listApplicationViewCapabilities(c.ViewCapabilities) {
		claim("view capability", viewCapabilitySpace, capability)
	}
	for _, f := range c.SearchFunctions.Functions {
		claim("search function", searchFunctionSpace, f.Name)
	}
	claimSlotName := func(what, space, id string, against ...string) {
		for _, other := range against {
			if owner, isOwned := owners[other+"\x00"+id]; isOwned {
				conflicts = append(conflicts, DuplicateContributionError{PluginID: pluginID, OwnerID: owner, What: what, ID: id})
				return
			}
		}
		pending[space+"\x00"+id] = true
	}
	for _, point := range c.SlotPoints {
		claimSlotName("slot point", slotPointSpace, point.Slot, slotPointSpace, submenuSpace)
	}
	for _, entry := range c.Slots {
		if entry.Submenu != "" {
			claimSlotName("submenu", submenuSpace, entry.Submenu, slotPointSpace)
		}
	}
	return pending, conflicts
}

type licenseProviderClaim struct {
	productID      string
	productIDOwner string
	url            string
	urlOwner       string
}

func listLicenseProviderConflicts(
	pluginID string,
	licenses []License,
	claimed map[string]licenseProviderClaim,
) []DuplicateContributionError {
	var conflicts []DuplicateContributionError
	for _, lic := range licenses {
		for _, prov := range lic.Providers {
			id := lic.ID + "/" + prov.Provider
			existing := claimed[id]
			if owner := findLicenseProviderConflictOwner(prov, existing); owner != "" {
				conflicts = append(conflicts, DuplicateContributionError{
					PluginID: pluginID,
					OwnerID:  owner,
					What:     "license provider",
					ID:       id,
				})
			}
		}
	}
	return conflicts
}

func findLicenseProviderConflictOwner(prov LicenseProvider, existing licenseProviderClaim) string {
	if prov.ProductID != "" && existing.productID != "" && prov.ProductID != existing.productID {
		return existing.productIDOwner
	}
	if prov.URL != "" && existing.url != "" && prov.URL != existing.url {
		return existing.urlOwner
	}
	return ""
}

func recordLicenseProviders(pluginID string, licenses []License, claimed map[string]licenseProviderClaim) {
	for _, lic := range licenses {
		for _, prov := range lic.Providers {
			id := lic.ID + "/" + prov.Provider
			entry := claimed[id]
			if entry.productID == "" && prov.ProductID != "" {
				entry.productID = prov.ProductID
				entry.productIDOwner = pluginID
			}
			if entry.url == "" && prov.URL != "" {
				entry.url = prov.URL
				entry.urlOwner = pluginID
			}
			claimed[id] = entry
		}
	}
}

func getReaderTabKind(id string) string {
	if strings.HasPrefix(id, readerTabKindPrefix) {
		return id
	}
	return readerTabKindPrefix + id
}

func listApplicationViewCapabilities(capabilities []string) []string {
	return slices.DeleteFunc(slices.Clone(capabilities), func(capability string) bool {
		return !slices.Contains(GetApplicationViewCapabilities(), capability)
	})
}
