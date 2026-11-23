package settings

import (
	"encoding/json"
	"fmt"
	"slices"
	"strings"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/connection"
	settingsdomain "github.com/jiva-studio/numen/libs/core/kernel/domain/settings"
)

// appName is the name a use of the rows of the shell is listed under.
const appName = "Numen"

func (r *ConnectionRows) readAppRowSetting(operation string) (connection.RowSetting, error) {
	if !slices.Contains(findAppSetting(keyReranking).Operations, operation) {
		return connection.RowSetting{}, nil
	}
	stored, err := r.readStoredFile()
	if err != nil {
		return connection.RowSetting{}, err
	}
	rows, err := r.readExistingRows(encodeAppRows(stored))
	return connection.RowSetting{Rows: rows, IsDetectedOnly: true}, err
}

func encodeAppRows(stored File) string {
	if len(stored.Search.Reranking) == 0 {
		return ""
	}
	encoded, err := json.Marshal(stored.Search.Reranking)
	if err != nil {
		return ""
	}
	return string(encoded)
}

func (r *ConnectionRows) listAppUses(connectionID string) ([]connection.Use, error) {
	stored, err := r.readStoredFile()
	if err != nil {
		return nil, err
	}
	if !hasConnection(readRowsJSON(encodeAppRows(stored)), connectionID) {
		return nil, nil
	}
	return []connection.Use{{
		PluginID: settingsdomain.AppOwner, PluginName: appName, Operations: findAppSetting(keyReranking).Operations,
	}}, nil
}

func removeAppRows(stored File, connectionID string) (settingsdomain.Setting, bool, error) {
	list := stored.Search.Reranking
	remainder := make([]any, 0, len(list))
	for _, row := range list {
		if object, isObject := row.(map[string]any); isObject && object[settingsdomain.RowConnectionKey] == connectionID {
			continue
		}
		remainder = append(remainder, row)
	}
	if len(remainder) == len(list) {
		return settingsdomain.Setting{}, false, nil
	}
	encoded, err := json.Marshal(remainder)
	if err != nil {
		return settingsdomain.Setting{}, false, fmt.Errorf("encode the rows of %s.%s: %w", settingsdomain.AppOwner, keyReranking, err)
	}
	path := append([]string{settingsdomain.AppOwner}, strings.Split(keyReranking, ".")...)
	return settingsdomain.Setting{Path: path, JSON: string(encoded)}, true, nil
}

func (r *ConnectionRows) readStoredFile() (File, error) {
	if r.source.ReadStored == nil {
		return File{}, nil
	}
	return r.source.ReadStored()
}

func listAppEntriesWithout(stored File, connectionID string) ([]settingsdomain.Setting, error) {
	entry, isChanged, err := removeAppRows(stored, connectionID)
	if err != nil || !isChanged {
		return nil, err
	}
	return []settingsdomain.Setting{entry}, nil
}
