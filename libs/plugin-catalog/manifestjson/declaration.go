package manifestjson

import (
	"encoding/json"
	"fmt"
)

// Declaration is which sections and lifecycle keys a manifest document writes. The decoded manifest
// holds a zero value for a key that is absent and for a key written as zero; a Declaration tells them
// apart.
type Declaration struct {
	hasBackend      bool
	lifecycleFields map[string]bool
}

// ReadDeclaration reads which sections and lifecycle keys the manifest JSON writes.
func ReadDeclaration(data []byte) (Declaration, error) {
	if _, err := Decode(data); err != nil {
		return Declaration{}, fmt.Errorf("read declaration: %w", err)
	}
	var document struct {
		Backend   *json.RawMessage           `json:"backend"`
		Lifecycle map[string]json.RawMessage `json:"lifecycle"`
	}
	if err := json.Unmarshal(data, &document); err != nil {
		return Declaration{}, fmt.Errorf("read declaration: %w", err)
	}
	declaration := Declaration{hasBackend: document.Backend != nil, lifecycleFields: make(map[string]bool, len(document.Lifecycle))}
	for key := range document.Lifecycle {
		declaration.lifecycleFields[key] = true
	}
	return declaration, nil
}

// HasBackend reports whether the manifest writes a backend section.
func (d Declaration) HasBackend() bool {
	return d.hasBackend
}

// HasLifecycleField reports whether the manifest writes the lifecycle key, by its JSON name.
func (d Declaration) HasLifecycleField(name string) bool {
	return d.lifecycleFields[name]
}
