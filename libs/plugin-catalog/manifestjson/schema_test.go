package manifestjson_test

import (
	"reflect"
	"testing"

	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/reflect/protoreflect"

	"github.com/jiva-studio/numen/libs/plugin-catalog/manifestjson"
	v1 "github.com/jiva-studio/numen/libs/protocol/gen/numen/v1"
)

// everyFieldManifest sets every field of the schema at least once, and every kind of option.
const everyFieldManifest = `{
	"id": "org.example.full", "name": "Full", "version": "1.2.3", "protocol": 1, "description": "d",
	"capabilities": ["tools"], "targets": ["desktop.editor"], "activationEvents": ["*"],
	"permissions": ["vault:read"], "runtimes": ["onnxruntime"],
	"dependencies": [{"id": "weights", "kind": "model", "version": "1", "sourceUrl": "https://example.org/w",
		"checksumSha256": "0000000000000000000000000000000000000000000000000000000000000000", "metadata": {"k": "v"}}],
	"lifecycle": {"idleTimeoutSeconds": 5, "keepAliveWhileVisible": true, "memoryLimitBytes": 7, "memoryLimitMB": 3},
	"backend": {"type": "process", "entrypoint": "bin/full", "serviceName": "numen.v1.Full"},
	"frontend": {"entrypoint": "dist/index.js", "styles": "dist/styles.css"},
	"settings": [
		{"key": "mode", "type": "enum", "title": "Mode", "description": "d", "default": "a", "palette": true, "icon": "gauge",
			"options": ["a", {"value": "b", "title": "B", "detail": "d", "dependencies": ["weights"]}],
			"min": 1, "max": 2, "step": 1, "modelsOf": "x", "replaces": ["a.b"], "accept": [".txt"]},
		{"key": "sub", "type": "enum", "title": "Sub", "optionsBy": {"setting": "mode", "options": {"a": ["x"], "b": ["y"]}},
			"when": {"setting": "mode", "is": ["a"]}}
	],
	"secrets": [{"name": "apiKey", "title": "Key", "description": "d", "when": {"setting": "mode", "is": ["a"]}}],
	"services": ["numen.v1.Full"],
	"noteTypes": [{"type": "link", "chunks": false, "headings": [1, 2], "idHeadingLevels": [3], "idPattern": "[a-z]+"}],
	"commands": [{"id": "run", "title": "Run", "icon": "play", "contentTypes": ["audio/*"], "noteTypes": ["link"],
		"when": {"liveTextKinds": ["k"], "hasOwnText": true},
		"arguments": [{"name": "n", "title": "N", "type": "number", "required": false, "hiddenInPalette": true},
			{"name": "m", "title": "M", "type": "number", "required": true}],
		"hiddenInPalette": true}],
	"texts": {"contentTypes": ["audio/*"]},
	"playback": {"contentTypes": ["video/*"], "noteTypes": ["link"]},
	"contentTypes": [{"type": "application/x-example", "globs": ["*.example"]}],
	"readers": [{"id": "r", "contentTypes": ["audio/*"], "rank": "default"}],
	"settingsViews": [{"id": "v", "title": "V", "after": "mode"}],
	"frames": ["https://player.example"],
	"contributes": {"tabKinds": [{"kind": "full", "title": "Full", "icon": "gauge"}], "feature": "full tabs", "category": "views"},
	"appVersion": {"min": "2026.1.0", "max": "2027.1.0"}
}`

func TestTheSampleManifestSetsEveryFieldOfTheSchema(t *testing.T) {
	var wire v1.Manifest
	if err := protojson.Unmarshal([]byte(everyFieldManifest), &wire); err != nil {
		t.Fatal(err)
	}

	present := map[string]bool{}
	collectSetFields(wire.ProtoReflect(), present)

	var missing []string
	for _, name := range listSchemaFields(wire.ProtoReflect().Descriptor(), map[string]bool{}) {
		if !present[name] {
			missing = append(missing, name)
		}
	}
	if len(missing) > 0 {
		t.Errorf("the sample manifest leaves these fields of the schema unset, so a mapping for them is untested: %v", missing)
	}
}

// listSchemaFields lists every field of the message and of the messages it holds, as Message.field.
func listSchemaFields(message protoreflect.MessageDescriptor, visited map[string]bool) []string {
	if visited[string(message.FullName())] {
		return nil
	}
	visited[string(message.FullName())] = true
	var names []string
	fields := message.Fields()
	for i := range fields.Len() {
		field := fields.Get(i)
		names = append(names, string(message.FullName())+"."+string(field.Name()))
		child := field.Message()
		if field.IsMap() {
			child = field.MapValue().Message()
		}
		if child != nil && child.ParentFile().Package() == "numen.v1" {
			names = append(names, listSchemaFields(child, visited)...)
		}
	}
	return names
}

// collectSetFields records every field that is set in the message or in a message it holds.
func collectSetFields(message protoreflect.Message, present map[string]bool) {
	message.Range(func(field protoreflect.FieldDescriptor, value protoreflect.Value) bool {
		present[string(message.Descriptor().FullName())+"."+string(field.Name())] = true
		switch {
		case field.IsMap():
			if field.MapValue().Message() != nil {
				value.Map().Range(func(_ protoreflect.MapKey, entry protoreflect.Value) bool {
					collectSetInMessage(entry.Message(), present)
					return true
				})
			}
		case field.IsList():
			if field.Message() != nil {
				for i := range value.List().Len() {
					collectSetInMessage(value.List().Get(i).Message(), present)
				}
			}
		case field.Message() != nil:
			collectSetInMessage(value.Message(), present)
		}
		return true
	})
}

func collectSetInMessage(message protoreflect.Message, present map[string]bool) {
	if message.Descriptor().ParentFile().Package() == "numen.v1" {
		collectSetFields(message, present)
	}
}

func TestTheSampleManifestIsReadWithEveryFieldMapped(t *testing.T) {
	m, err := manifestjson.Decode([]byte(everyFieldManifest))
	if err != nil {
		t.Fatal(err)
	}

	set := map[string]bool{}
	collectSetValues(reflect.ValueOf(m), set)

	for _, name := range listModelFields(reflect.TypeOf(m), map[reflect.Type]bool{}) {
		if !set[name] && !isUnmappedByDesign(name) {
			t.Errorf("%s holds its zero value in every instance of the sample: a mapping for it is untested", name)
		}
	}
}

func listModelFields(kind reflect.Type, visited map[reflect.Type]bool) []string {
	for kind.Kind() == reflect.Slice || kind.Kind() == reflect.Ptr {
		kind = kind.Elem()
	}
	if kind.Kind() != reflect.Struct || visited[kind] || kind.PkgPath() == "" {
		return nil
	}
	visited[kind] = true
	var names []string
	for i := range kind.NumField() {
		field := kind.Field(i)
		names = append(names, kind.Name()+"."+field.Name)
		names = append(names, listModelFields(field.Type, visited)...)
	}
	return names
}

// collectSetValues records, as Type.Field, every field that holds a non-zero value in the struct or
// in a struct it holds.
func collectSetValues(value reflect.Value, set map[string]bool) {
	switch value.Kind() {
	case reflect.Struct:
		for i := range value.NumField() {
			field := value.Field(i)
			if !field.IsZero() {
				set[value.Type().Name()+"."+value.Type().Field(i).Name] = true
			}
			collectSetValues(field, set)
		}
	case reflect.Slice:
		for i := range value.Len() {
			collectSetValues(value.Index(i), set)
		}
	case reflect.Ptr:
		if !value.IsNil() {
			collectSetValues(value.Elem(), set)
		}
	}
}

// isUnmappedByDesign names the fields the reader does not read from the file: the folder of the plugin
// is known to whoever discovers it, and a note type that keeps the default chunking is false by choice.
func isUnmappedByDesign(name string) bool {
	return name == "DependencyRequirement.PluginDir" || name == "Discovered.Dir" || name == "NoteType.Chunks"
}

func TestOnlyTheJSONNameOfAFieldIsRead(t *testing.T) {
	m, err := manifestjson.Decode([]byte(`{"id":"a.b","activation_events":["on-edit"],"settings_views":[{"id":"v"}]}`))
	if err != nil {
		t.Fatal(err)
	}

	if len(m.ActivationEvents) != 0 || len(m.SettingsViews) != 0 {
		t.Errorf("the protobuf names of the fields were read: %+v", m)
	}
}

func TestARepeatedKeyResolvesToItsLastValue(t *testing.T) {
	m, err := manifestjson.Decode([]byte(`{"id":"a.b","description":"first","description":"last"}`))
	if err != nil {
		t.Fatal(err)
	}

	if m.Description != "last" {
		t.Errorf("description = %q, want last", m.Description)
	}
}

func TestDataAfterTheManifestIsRefused(t *testing.T) {
	if _, err := manifestjson.Decode([]byte(`{"id":"a.b"} {"id":"c.d"}`)); err == nil {
		t.Error("data after the manifest was accepted")
	}
}

func TestAnOptionOfNoValueShapeIsRefused(t *testing.T) {
	_, err := manifestjson.Decode([]byte(`{"settings":[{"key":"k","options":[1]}]}`))
	if err == nil {
		t.Error("a number was accepted as an option")
	}
}

func TestANumberOutsideTheRangeOfAFieldIsRefused(t *testing.T) {
	_, err := manifestjson.Decode([]byte(`{"lifecycle":{"memoryLimitMB":18446744073709551615}}`))
	if err == nil {
		t.Error("a limit that overflows was accepted")
	}
}
