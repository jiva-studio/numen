package settings

import (
	"context"
	"encoding/json"
	"fmt"
	"maps"
	"slices"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/connection"
	"github.com/jiva-studio/numen/libs/core/kernel/domain/plugin"
	settingsdomain "github.com/jiva-studio/numen/libs/core/kernel/domain/settings"
	"github.com/jiva-studio/numen/libs/core/kernel/port/connectionport"
	"github.com/jiva-studio/numen/libs/plugin/catalog/manifest"
)

// ConnectionRowsSource is what the rows of the plugins are read from and written to. Plugins lists
// the plugins with the value in force of each setting. ReadStored reads the settings file as it is
// stored. Write is the path every write of a setting takes. Report receives the failures of a
// plugin that cannot be configured.
type ConnectionRowsSource struct {
	Connections connectionport.Repository
	Plugins     func() []plugin.Info
	Declared    Declarations
	ReadStored  func() (File, error)
	Write       func(entries []settingsdomain.Setting) error
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
func (r *ConnectionRows) ListRows(pluginID, operation string) ([]connection.Row, error) {
	for _, plugin := range r.source.Plugins() {
		if plugin.ID != pluginID {
			continue
		}
		for _, one := range plugin.Settings {
			if one.Type == manifest.SettingTypeConnections && one.Operation == operation && one.IsShown {
				return r.readExistingRows(one.ValueJSON)
			}
		}
	}
	return nil, nil
}

func (r *ConnectionRows) readExistingRows(valueJSON string) ([]connection.Row, error) {
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
	return slices.DeleteFunc(rows, func(row connection.Row) bool { return !exists[row.ConnectionID] }), nil
}

func readRowsJSON(valueJSON string) []connection.Row {
	var stored []struct {
		Connection string `json:"connection"`
		Model      string `json:"model"`
		Reasoning  string `json:"reasoning"`
	}
	if valueJSON == "" || json.Unmarshal([]byte(valueJSON), &stored) != nil {
		return nil
	}
	rows := make([]connection.Row, 0, len(stored))
	for _, one := range stored {
		rows = append(rows, connection.Row{ConnectionID: one.Connection, Model: one.Model, Reasoning: one.Reasoning})
	}
	return rows
}

// ListUses returns the settings of rows in force that name the connection, once for each plugin and
// operation, in the order of the plugin ids.
func (r *ConnectionRows) ListUses(connectionID string) ([]connection.Use, error) {
	var uses []connection.Use
	for _, plugin := range r.source.Plugins() {
		for _, one := range plugin.Settings {
			if one.Type != manifest.SettingTypeConnections || !one.IsShown || !namesConnection(readRowsJSON(one.ValueJSON), connectionID) {
				continue
			}
			uses = append(uses, connection.Use{PluginID: plugin.ID, PluginName: plugin.Name, Operation: one.Operation})
		}
	}
	return uses, nil
}

func namesConnection(rows []connection.Row, connectionID string) bool {
	return slices.ContainsFunc(rows, func(row connection.Row) bool { return row.ConnectionID == connectionID })
}

// RemoveRows writes each setting of rows of the stored file without the rows that name the
// connection, through the path every write of a setting takes. A setting that is not shown, and a
// setting whose rows are left out of force, is reached through the stored file.
func (r *ConnectionRows) RemoveRows(connectionID string) error {
	stored, err := r.source.ReadStored()
	if err != nil {
		return err
	}
	entries, err := r.listEntriesWithout(stored, connectionID)
	if err != nil {
		return err
	}
	if len(entries) == 0 {
		return nil
	}
	return r.source.Write(entries)
}

func removeRowsOf(stored File, owner, key, connectionID string) (settingsdomain.Setting, bool, error) {
	list, isList := stored.Plugins[owner][key].([]any)
	if !isList {
		return settingsdomain.Setting{}, false, nil
	}
	remainder := make([]any, 0, len(list))
	for _, row := range list {
		if object, isObject := row.(map[string]any); isObject && object[rowConnection] == connectionID {
			continue
		}
		remainder = append(remainder, row)
	}
	if len(remainder) == len(list) {
		return settingsdomain.Setting{}, false, nil
	}
	encoded, err := json.Marshal(remainder)
	if err != nil {
		return settingsdomain.Setting{}, false, fmt.Errorf("encode the rows of %s.%s: %w", owner, key, err)
	}
	return settingsdomain.Setting{Path: []string{owner, key}, JSON: string(encoded)}, true, nil
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

func (r *ConnectionRows) listEntriesWithout(stored File, connectionID string) ([]settingsdomain.Setting, error) {
	var entries []settingsdomain.Setting
	for _, owner := range slices.Sorted(maps.Keys(stored.Plugins)) {
		declared, isFound := r.source.Declared(owner)
		if !isFound {
			continue
		}
		for _, one := range declared {
			if one.Type != manifest.SettingTypeConnections {
				continue
			}
			entry, isChanged, err := removeRowsOf(stored, owner, one.Key, connectionID)
			if err != nil {
				return nil, err
			}
			if isChanged {
				entries = append(entries, entry)
			}
		}
	}
	return entries, nil
}
