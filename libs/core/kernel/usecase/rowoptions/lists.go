package rowoptions

import (
	"context"
	"maps"
	"slices"
	"sync"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/connection"
	"github.com/jiva-studio/numen/libs/core/kernel/domain/modelcall"
	"github.com/jiva-studio/numen/libs/core/kernel/port/connectionport"
	"github.com/jiva-studio/numen/libs/plugin/catalog/manifest"
)

// KeptList is the options a provider listed for one row field of a connection, with the values of the row
// fields declared before it and the operations it was asked with, and what of the connection the listing
// depends on.
type KeptList struct {
	ConnectionID  string
	Field         string
	RowOptions    map[string]string
	Operations    []string
	Revision      int
	PluginVersion string
	Fields        map[string]string
	Options       []modelcall.Option
}

// ListFiles keeps the option lists of the connections of each plugin across processes, one set per plugin.
type ListFiles interface {
	ReadOptionLists(pluginID string) ([]KeptList, bool)
	WriteOptionLists(pluginID string, lists []KeptList)
}

// RunningProviders opens the provider of a plugin only while its process runs; a plugin that does not run
// is not started and returns false.
type RunningProviders interface {
	OpenRunningProvider(ctx context.Context, pluginID string) (connectionport.Provider, func(), bool)
}

// Lists keeps the option lists through files, for every Options of the host. A listing counts only for
// the connection as it was read then: a change of its revision, its fields or the version of its plugin
// makes it count as none.
type Lists struct {
	files      ListFiles
	mu         sync.Mutex
	refreshing map[string]bool
}

// NewLists returns the lists kept in files.
func NewLists(files ListFiles) *Lists {
	return &Lists{files: files, refreshing: map[string]bool{}}
}

func newKeptList(resolved connection.Resolved, field string, options map[string]string, operations []string) KeptList {
	return KeptList{
		ConnectionID: resolved.ID, Field: field, RowOptions: getPrecedingOptions(resolved.Declaration, field, options),
		Operations: slices.Clone(operations),
		Revision:   resolved.Revision, PluginVersion: resolved.PluginVersion, Fields: maps.Clone(resolved.Fields),
	}
}

// getPrecedingOptions returns the values of the row fields the type declares before the field: what the
// field can take depends on them, not on its own value or on the fields after it.
func getPrecedingOptions(declaration manifest.ConnectionType, field string, options map[string]string) map[string]string {
	preceding := map[string]string{}
	for _, one := range declaration.RowFields {
		if one.Key == field {
			break
		}
		if value, isSet := options[one.Key]; isSet {
			preceding[one.Key] = value
		}
	}
	return preceding
}

func (k KeptList) isSameRequest(other KeptList) bool {
	return k.ConnectionID == other.ConnectionID && k.Field == other.Field && maps.Equal(k.RowOptions, other.RowOptions) &&
		slices.Equal(k.Operations, other.Operations)
}

func (k KeptList) isSameState(other KeptList) bool {
	return k.Revision == other.Revision && k.PluginVersion == other.PluginVersion && maps.Equal(k.Fields, other.Fields)
}

func (l *Lists) get(pluginID string, asked KeptList) ([]modelcall.Option, bool) {
	l.mu.Lock()
	defer l.mu.Unlock()
	kept, _ := l.files.ReadOptionLists(pluginID)
	for _, one := range kept {
		if one.isSameRequest(asked) && one.isSameState(asked) {
			return slices.Clone(one.Options), true
		}
	}
	return nil, false
}

// put keeps the listing in place of the one of the same request, and drops the listings of the connection
// made for another state of it.
func (l *Lists) put(pluginID string, listed KeptList) {
	l.mu.Lock()
	defer l.mu.Unlock()
	kept, _ := l.files.ReadOptionLists(pluginID)
	lists := []KeptList{}
	for _, one := range kept {
		if one.isSameRequest(listed) || (one.ConnectionID == listed.ConnectionID && !one.isSameState(listed)) {
			continue
		}
		lists = append(lists, one)
	}
	l.files.WriteOptionLists(pluginID, append(lists, listed))
}

// listRequests returns the listings kept for the connection, in any state of it.
func (l *Lists) listRequests(pluginID, connectionID string) []KeptList {
	l.mu.Lock()
	defer l.mu.Unlock()
	kept, _ := l.files.ReadOptionLists(pluginID)
	var requests []KeptList
	for _, one := range kept {
		if one.ConnectionID == connectionID {
			requests = append(requests, one)
		}
	}
	return requests
}

// begin marks a refresh of the connection as running, and returns false where one runs already.
func (l *Lists) begin(connectionID string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.refreshing[connectionID] {
		return false
	}
	l.refreshing[connectionID] = true
	return true
}

func (l *Lists) end(connectionID string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	delete(l.refreshing, connectionID)
}
