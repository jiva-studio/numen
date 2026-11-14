package providerbridge

import (
	"encoding/json"
	"fmt"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/connection"
)

// DetectionFiles keeps the answers of the providers of each plugin across processes.
type DetectionFiles interface {
	ReadDetectedConnections(pluginID string) ([]byte, bool)
	WriteDetectedConnections(pluginID string, content []byte)
}

// storedFile is the content of the file of one plugin: the answers by type id.
type storedFile struct {
	Answers map[string]storedAnswer `json:"answers"`
}

type storedAnswer struct {
	PluginVersion string             `json:"pluginVersion"`
	Connections   []storedConnection `json:"connections"`
}

type storedConnection struct {
	Name   string         `json:"name"`
	Fields map[string]any `json:"fields"`
	Path   string         `json:"path,omitempty"`
}

func (d *Detection) load(pluginID string) {
	if d.files == nil || d.loaded[pluginID] {
		return
	}
	d.loaded[pluginID] = true
	raw, isKept := d.files.ReadDetectedConnections(pluginID)
	if !isKept {
		return
	}
	var stored storedFile
	if err := json.Unmarshal(raw, &stored); err != nil {
		d.report(fmt.Errorf("read the detected connections of %s: %w", pluginID, err))
		return
	}
	for typeID, one := range stored.Answers {
		list := make([]connection.Connection, 0, len(one.Connections))
		for _, kept := range one.Connections {
			fields := kept.Fields
			if fields == nil {
				fields = map[string]any{}
			}
			list = append(list, connection.Connection{
				ID: connection.GetDetectedID(typeID, kept.Name), TypeID: typeID, Name: kept.Name, Fields: fields,
				Path: kept.Path,
			})
		}
		d.answers[typeID] = answer{pluginID: pluginID, pluginVersion: one.PluginVersion, connections: list}
	}
}

func (d *Detection) writeStoreFile(pluginID string) error {
	d.writeMu.Lock()
	defer d.writeMu.Unlock()
	files, stored := d.readStoredFile(pluginID)
	if files == nil {
		return nil
	}
	raw, err := json.Marshal(stored)
	if err != nil {
		return err
	}
	files.WriteDetectedConnections(pluginID, raw)
	return nil
}

func (d *Detection) readStoredFile(pluginID string) (DetectionFiles, storedFile) {
	d.mu.Lock()
	defer d.mu.Unlock()
	stored := storedFile{Answers: map[string]storedAnswer{}}
	for typeID, one := range d.answers {
		if one.pluginID != pluginID {
			continue
		}
		kept := storedAnswer{PluginVersion: one.pluginVersion, Connections: []storedConnection{}}
		for _, found := range one.connections {
			kept.Connections = append(kept.Connections, storedConnection{Name: found.Name, Fields: found.Fields, Path: found.Path})
		}
		stored.Answers[typeID] = kept
	}
	return d.files, stored
}
