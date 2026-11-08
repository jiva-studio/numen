package settings

import (
	"context"
	"encoding/json"
	"fmt"
	"maps"
	"slices"

	"github.com/jiva-studio/numen/libs/core/port/connectionport"
	"github.com/jiva-studio/numen/libs/core/port/pluginport"
	"github.com/jiva-studio/numen/libs/core/port/settingsport"
)

// ConnectionRowsSource is what the rows of the plugins are read from and written to. Plugins lists
// the plugins with the value in force of each setting. ReadStored reads the settings file as it is
// stored. Write is the path every write of a setting takes. Report receives the failures of a
// plugin that cannot be configured.
type ConnectionRowsSource struct {
	Connections connectionport.Repository
	Plugins     func() []pluginport.PluginInfo
	Declared    Declarations
	ReadStored  func() (File, error)
	Write       func(entries []settingsport.Setting) error
	Refresh     func(ctx context.Context, forced ...string) error
	Report      func(err error)
}

// ConnectionRows reads and changes the settings of type connections of the plugins.
type ConnectionRows struct{ source ConnectionRowsSource }

// NewConnectionRows returns the rows of the plugins in the source.
func NewConnectionRows(source ConnectionRowsSource) *ConnectionRows {
	return &ConnectionRows{source: source}
}

// ListRows returns the rows in force of the setting of the plugin that needs the operation, in the
// order the person set. A setting that is not shown has none, and a row that names a connection
// that does not exist is left out.
func (r *ConnectionRows) ListRows(pluginID, operation string) ([]connectionport.Row, error) {
	for _, plugin := range r.source.Plugins() {
		if plugin.ID != pluginID {
			continue
		}
		for _, one := range plugin.Settings {
			if one.Type == "connections" && one.Operation == operation && one.IsShown {
				return r.readExistingRows(one.ValueJSON)
			}
		}
	}
	return nil, nil
}

func (r *ConnectionRows) readExistingRows(valueJSON string) ([]connectionport.Row, error) {
	rows := readRowsJSON(valueJSON)
	if len(rows) == 0 {
		return nil, nil
	}
	connections, err := r.source.Connections.ListConnections()
	if err != nil {
		return nil, err
	}
	exists := make(map[string]bool, len(connections))
	for _, one := range connections {
		exists[one.ID] = true
	}
	return slices.DeleteFunc(rows, func(row connectionport.Row) bool { return !exists[row.ConnectionID] }), nil
}

func readRowsJSON(valueJSON string) []connectionport.Row {
	var stored []struct {
		Connection string `json:"connection"`
		Model      string `json:"model"`
		Reasoning  string `json:"reasoning"`
	}
	if valueJSON == "" || json.Unmarshal([]byte(valueJSON), &stored) != nil {
		return nil
	}
	rows := make([]connectionport.Row, 0, len(stored))
	for _, one := range stored {
		rows = append(rows, connectionport.Row{ConnectionID: one.Connection, Model: one.Model, Reasoning: one.Reasoning})
	}
	return rows
}

// ListUses returns the settings of rows in force that name the connection, once for each plugin and
// operation, in the order of the plugin ids.
func (r *ConnectionRows) ListUses(connectionID string) ([]connectionport.Use, error) {
	var uses []connectionport.Use
	for _, plugin := range r.source.Plugins() {
		for _, one := range plugin.Settings {
			if one.Type != "connections" || !one.IsShown || !namesConnection(readRowsJSON(one.ValueJSON), connectionID) {
				continue
			}
			uses = append(uses, connectionport.Use{PluginID: plugin.ID, PluginName: plugin.Name, Operation: one.Operation})
		}
	}
	return uses, nil
}

func namesConnection(rows []connectionport.Row, connectionID string) bool {
	return slices.ContainsFunc(rows, func(row connectionport.Row) bool { return row.ConnectionID == connectionID })
}

// RemoveRows writes each setting of rows of the stored file without the rows that name the
// connection, through the path every write of a setting takes. A setting that is not shown, and a
// setting whose rows are left out of force, is reached through the stored file.
func (r *ConnectionRows) RemoveRows(connectionID string) error {
	stored, err := r.source.ReadStored()
	if err != nil {
		return err
	}
	var entries []settingsport.Setting
	for _, owner := range slices.Sorted(maps.Keys(stored.Plugins)) {
		declared, isFound := r.source.Declared(owner)
		if !isFound {
			continue
		}
		for _, one := range declared {
			if one.Type != "connections" {
				continue
			}
			entry, isChanged, err := removeRowsOf(stored, owner, one.Key, connectionID)
			if err != nil {
				return err
			}
			if isChanged {
				entries = append(entries, entry)
			}
		}
	}
	if len(entries) == 0 {
		return nil
	}
	return r.source.Write(entries)
}

func removeRowsOf(stored File, owner, key, connectionID string) (settingsport.Setting, bool, error) {
	list, isList := stored.Plugins[owner][key].([]any)
	if !isList {
		return settingsport.Setting{}, false, nil
	}
	kept := make([]any, 0, len(list))
	for _, row := range list {
		if object, isObject := row.(map[string]any); isObject && object[rowConnection] == connectionID {
			continue
		}
		kept = append(kept, row)
	}
	if len(kept) == len(list) {
		return settingsport.Setting{}, false, nil
	}
	encoded, err := json.Marshal(kept)
	if err != nil {
		return settingsport.Setting{}, false, fmt.Errorf("encode the rows of %s.%s: %w", owner, key, err)
	}
	return settingsport.Setting{Path: []string{owner, key}, JSON: string(encoded)}, true, nil
}

// Refresh sends the configuration again to each running plugin whose rows in force name the
// connection. A plugin that cannot be configured is reported and the others are sent theirs.
func (r *ConnectionRows) Refresh(ctx context.Context, connectionID string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	uses, err := r.ListUses(connectionID)
	if err != nil {
		return err
	}
	var pluginIDs []string
	for _, one := range uses {
		if !slices.Contains(pluginIDs, one.PluginID) {
			pluginIDs = append(pluginIDs, one.PluginID)
		}
	}
	if len(pluginIDs) == 0 {
		return nil
	}
	if err := r.source.Refresh(ctx, pluginIDs...); err != nil {
		if ctxErr := ctx.Err(); ctxErr != nil {
			return ctxErr
		}
		r.source.Report(fmt.Errorf("send the configuration after connection %s changed: %w", connectionID, err))
	}
	return nil
}
