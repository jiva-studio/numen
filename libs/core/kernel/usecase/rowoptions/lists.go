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

// ListFiles keeps the option lists of the connections of each plugin across processes, one set per plugin.
type ListFiles interface {
	ReadOptionLists(pluginID string) ([]connection.OptionList, bool)
	WriteOptionLists(pluginID string, lists []connection.OptionList)
}

// RunningProviders opens the provider of a plugin only while its process runs; a plugin that does not run
// is not started and returns false.
type RunningProviders interface {
	OpenRunningProvider(ctx context.Context, pluginID string) (connectionport.Provider, func(), bool)
}

// Lists keeps the option lists through files, for every Options of the host. A listing counts only for
// the connection as it was read then: a change of its revision, its fields, the version or configuration of
// its plugin, or the connections its plugin reaches through its grants makes it count as none, and a new
// listing of the connection drops it. A listing read for an earlier revision of the connection than one
// kept is not kept.
type Lists struct {
	files      ListFiles
	configs    ConfigRevisions
	mu         sync.Mutex
	refreshing map[string]bool
}

// NewLists returns the lists kept in files, each for the configuration configs names for its plugin.
func NewLists(files ListFiles, configs ConfigRevisions) *Lists {
	return &Lists{files: files, configs: configs, refreshing: map[string]bool{}}
}

func newOptionList(resolved connection.Resolved, field string, options map[string]string, operations []string) connection.OptionList {
	return connection.OptionList{
		ConnectionID: resolved.ID, Field: field, RowOptions: getPrecedingOptions(resolved.Declaration, field, options),
		Operations: slices.Clone(operations),
		Revision:   resolved.Revision, PluginVersion: resolved.PluginVersion, Fields: maps.Clone(resolved.Fields),
	}
}

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

func isSameRequest(k, other connection.OptionList) bool {
	return k.ConnectionID == other.ConnectionID && k.Field == other.Field && maps.Equal(k.RowOptions, other.RowOptions) &&
		slices.Equal(k.Operations, other.Operations)
}

func isSameState(k, other connection.OptionList) bool {
	return k.Revision == other.Revision && k.PluginVersion == other.PluginVersion && k.ConfigRevision == other.ConfigRevision &&
		k.GrantRevision == other.GrantRevision && maps.Equal(k.Fields, other.Fields)
}

func (l *Lists) getConfigRevision(pluginID string) string {
	return l.configs.GetPluginConfigRevision(pluginID)
}

func (l *Lists) get(pluginID string, asked connection.OptionList) ([]modelcall.Option, bool) {
	l.mu.Lock()
	defer l.mu.Unlock()
	kept, _ := l.files.ReadOptionLists(pluginID)
	for _, one := range kept {
		if isSameRequest(one, asked) && isSameState(one, asked) {
			return slices.Clone(one.Options), true
		}
	}
	return nil, false
}

func (l *Lists) put(pluginID string, listed connection.OptionList) {
	l.mu.Lock()
	defer l.mu.Unlock()
	kept, _ := l.files.ReadOptionLists(pluginID)
	if slices.ContainsFunc(kept, func(one connection.OptionList) bool {
		return one.ConnectionID == listed.ConnectionID && one.Revision > listed.Revision
	}) {
		return
	}
	lists := []connection.OptionList{}
	for _, one := range kept {
		if isSameRequest(one, listed) || (one.ConnectionID == listed.ConnectionID && !isSameState(one, listed)) {
			continue
		}
		lists = append(lists, one)
	}
	l.files.WriteOptionLists(pluginID, append(lists, listed))
}

func (l *Lists) drop(pluginID, connectionID string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	kept, isKept := l.files.ReadOptionLists(pluginID)
	if !isKept {
		return
	}
	lists := slices.DeleteFunc(kept, func(one connection.OptionList) bool { return one.ConnectionID == connectionID })
	l.files.WriteOptionLists(pluginID, lists)
}

func (l *Lists) listRequests(pluginID, connectionID string) []connection.OptionList {
	l.mu.Lock()
	defer l.mu.Unlock()
	kept, _ := l.files.ReadOptionLists(pluginID)
	var requests []connection.OptionList
	for _, one := range kept {
		if one.ConnectionID == connectionID {
			requests = append(requests, one)
		}
	}
	return requests
}

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
