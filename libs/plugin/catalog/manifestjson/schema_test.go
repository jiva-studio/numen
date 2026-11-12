package manifestjson_test

import (
	"encoding/json"
	"errors"
	"reflect"
	"slices"
	"strings"
	"testing"
	"testing/iotest"

	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/reflect/protoreflect"

	v1 "github.com/jiva-studio/numen/libs/core/protocol/gen/numen/v1"
	"github.com/jiva-studio/numen/libs/plugin/catalog/manifestjson"
)

// everyFieldManifest sets every field of the schema at least once, and every kind of option.
const everyFieldManifest = `{
	"id": "org.example.full", "name": "Full", "version": "1.2.3", "protocol": 1, "description": "d", "isOffByDefault": true,
	"capabilities": ["tools"], "targets": ["desktop.editor"], "activationEvents": ["onStartup"],
	"permissions": ["vault:read"],
	"dependencies": [{"id": "weights", "kind": "model", "version": "1", "sourceUrl": "https://example.org/w",
		"checksumSha256": "0000000000000000000000000000000000000000000000000000000000000000", "metadata": {"k": "v"}}],
	"backend": {"entrypoint": "bin/full"},
	"frontend": {},
	"settings": [
		{"key": "mode", "type": "enum", "title": "Mode", "description": "d", "default": "a", "palette": true, "icon": "gauge",
			"options": ["a", {"value": "b", "title": "B", "detail": "d", "dependencies": ["weights"], "when": "app.platform != 'darwin'"}],
			"min": 1, "max": 2, "step": 1, "accept": [".txt"], "multiple": true},
		{"key": "sub", "type": "enum", "title": "Sub", "optionsBy": {"setting": "mode", "options": {"a": ["x"], "b": ["y"]}},
			"when": {"setting": "mode", "is": ["a"]}},
		{"key": "theme", "type": "enum", "title": "Theme", "optionsFrom": "themes", "preview": true},
		{"key": "token", "type": "string", "title": "Token", "generate": "token"},
		{"key": "rows", "type": "connections", "title": "Rows", "operations": ["llm.complete"], "max": 2}
	],
	"services": ["numen.v1.Full"],
	"settingsViews": [{"id": "v", "title": "V"}],
	"contributes": {"commands": [{"id": "task.run", "title": "Run", "icon": "play",
		"when": "target.contentType.startsWith('audio/')",
		"permission": "vault:read",
		"arguments": [{"name": "n", "title": "N", "type": "number", "required": false},
			{"name": "m", "title": "M", "type": "number", "required": true}]}],
		"tabKinds": [{"kind": "full", "title": "Full", "icon": "gauge", "when": "target.noteType == 'full'", "rank": "alternate"}], "feature": "full tabs", "category": "views",
		"themes": [{"id": "full-dark", "name": "Full Dark", "appearance": "dark", "file": "themes/full-dark.css"}],
		"handlers": [{"command": "app:file.createText", "title": "Transcribe", "when": "target.contentType.startsWith('audio/')"}],
		"triggers": [{"event": "file.added", "command": "app:file.createText", "when": "config.org.example.full.mode == 'a'"}],
		"textKinds": [{"kind": "recognition", "title": "Recognized text"}],
		"readers": [{"id": "r", "contentTypes": ["audio/*"], "rank": "default"}],
		"noteTypes": [{"type": "link", "chunks": false, "headings": [1, 2], "idHeadingLevels": [3], "idPattern": "[a-z]+"}],
		"frameOrigins": ["https://player.example"],
		"slots": [{"slot": "titleBar.commands", "command": "task.run", "group": "run", "order": 1, "values": {"n": "3"}},
			{"slot": "titleBar", "view": "bar"},
			{"slot": "fileTree.context", "submenu": "full.new", "title": "New", "icon": "plus", "when": "target.isFolder"}],
		"listeners": [{"name": "mcp", "port": 3333, "protocol": "tcp"}, {"name": "console", "portSetting": "mode", "protocol": "tcp"}],
		"keybindings": [{"key": "mod+shift+r", "command": "task.run", "values": {"n": "3"}, "when": "window.isVaultOpen"}],
		"viewCapabilities": ["conversation"],
		"dropHandlers": [{"types": ["text/uri-list"]}],
		"connectionTypes": [{"id": "svc", "title": "Service", "description": "d", "operations": ["llm.complete"], "fields": [
			{"key": "baseUrl", "type": "string", "title": "Address", "description": "d", "isAddress": true, "fixedValue": "https://svc.example/v1"},
			{"key": "mode", "type": "enum", "title": "Mode", "options": ["a", "b"], "default": "a"},
			{"key": "apiKey", "type": "secret", "title": "Key", "required": true},
			{"key": "parallelRequests", "type": "number", "title": "Parallel", "default": 4, "min": 1, "max": 32, "step": 1}]}],
		"operations": [{"name": "llm.complete", "title": "complete text"}]}
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
	if len(m.Settings) != 5 || len(m.Settings[1].OptionsBy.Options["a"]) != 1 || m.Settings[1].OptionsBy.Options["a"][0].Value != "x" {
		t.Errorf("optionsBy = %+v", m.Settings)
	}
	if len(m.Contributes.NoteTypes) != 1 || m.Contributes.NoteTypes[0].Headings == nil || !slices.Equal(*m.Contributes.NoteTypes[0].Headings, []int{1, 2}) ||
		!slices.Equal(m.Contributes.NoteTypes[0].IDHeadingLevels, []int{3}) {
		t.Errorf("noteTypes = %+v", m.Contributes.NoteTypes)
	}
}

func listModelFields(kind reflect.Type, visited map[reflect.Type]bool) []string {
	for kind.Kind() == reflect.Slice || kind.Kind() == reflect.Pointer {
		kind = kind.Elem()
	}
	if kind.Kind() != reflect.Struct || visited[kind] || kind.PkgPath() == "" {
		return nil
	}
	visited[kind] = true
	var names []string
	for field := range kind.Fields() {
		names = append(names, kind.Name()+"."+field.Name)
		names = append(names, listModelFields(field.Type, visited)...)
	}
	return names
}

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
	case reflect.Pointer:
		if !value.IsNil() {
			collectSetValues(value.Elem(), set)
		}
	}
}

func isUnmappedByDesign(name string) bool {
	return name == "DependencyRequirement.PluginDir" || name == "Discovered.Dir" || name == "Manifest.Warnings" || name == "NoteType.Chunks"
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
	if _, err := manifestjson.Decode([]byte(`{"id":"a.b"} {"id":"c.d"}`)); err == nil || !strings.Contains(err.Error(), "unexpected data after the manifest") {
		t.Errorf("data after the manifest error = %v", err)
	}
	var syntaxErr *json.SyntaxError
	if _, err := manifestjson.Decode([]byte(`{invalid}`)); !errors.As(err, &syntaxErr) {
		t.Errorf("expected json.SyntaxError from Decode, got %v", err)
	}
	if _, err := manifestjson.Read(strings.NewReader(`{invalid}`)); !errors.As(err, &syntaxErr) || !strings.Contains(err.Error(), "decode manifest:") {
		t.Errorf("expected wrapped json.SyntaxError from Read, got %v", err)
	}
	readErr := errors.New("read failed")
	if _, err := manifestjson.Read(iotest.ErrReader(readErr)); !errors.Is(err, readErr) {
		t.Errorf("expected wrapped reader error from Read, got %v", err)
	}
}

func TestAnOptionOfNoValueShapeIsRefused(t *testing.T) {
	if _, err := manifestjson.Decode(
		[]byte(`{"settings":[{"key":"k","options":[1]}]}`),
	); err == nil || !strings.Contains(err.Error(), `setting "k"`) ||
		!strings.Contains(err.Error(), "options must be a list") {
		t.Errorf("a number was accepted as an option: %v", err)
	}
	if _, err := manifestjson.Decode(
		[]byte(`{"settings":[{"key":"k","options":[{"value":1}]}]}`),
	); err == nil ||
		!strings.Contains(err.Error(), "options must be a list") {
		t.Errorf("an option with numeric value was accepted: %v", err)
	}
	if _, err := manifestjson.Decode(
		[]byte(`{"settings":[{"key":"k","optionsBy":{"setting":"p","options":{"a":[1]}}}]}`),
	); err == nil || !strings.Contains(err.Error(), `setting "k"`) ||
		!strings.Contains(err.Error(), `options under "a"`) {
		t.Errorf("an invalid optionsBy option was accepted: %v", err)
	}
	if _, err := manifestjson.Decode(
		[]byte(`{"contributes":{"noteTypes":[{"type":"link","headings":["bad"]}]}}`),
	); err == nil ||
		!strings.Contains(err.Error(), `note type "link"`) {
		t.Errorf("a string heading was accepted: %v", err)
	}
	if _, err := manifestjson.Decode(
		[]byte(`{"contributes":{"noteTypes":[{"type":"link","headings":[1.5]}]}}`),
	); err == nil ||
		!strings.Contains(err.Error(), `note type "link"`) {
		t.Errorf("a fractional heading was accepted: %v", err)
	}
	m, err := manifestjson.Decode([]byte(`{"contributes":{"noteTypes":[{"type":"link","headings":[]}]}}`))
	if err != nil || len(m.Contributes.NoteTypes) != 1 || m.Contributes.NoteTypes[0].Headings == nil || len(*m.Contributes.NoteTypes[0].Headings) != 0 {
		t.Errorf("empty headings = %+v, %v", m.Contributes.NoteTypes, err)
	}
}

func TestANumberOutsideTheRangeOfAFieldIsRefused(t *testing.T) {
	_, err := manifestjson.Decode([]byte(`{"protocol":4294967296}`))
	if err == nil {
		t.Error("a limit that overflows was accepted")
	}
}

func TestAKeyOfTheWrongTypeIsRefused(t *testing.T) {
	if _, err := manifestjson.Decode([]byte(`{"id":"a.b","contributes":{"readers":"markdown"}}`)); err == nil {
		t.Error("a string was accepted as a list of readers")
	}
}

func TestKeysTheSchemaDoesNotNameAreReportedAsWarnings(t *testing.T) {
	m, err := manifestjson.Decode([]byte(`{"id":"a.b","unrelated":{},"contributes":{"tabKinds":[{"kind":"k","title":"K","shape":1}],
		"extra":true},"settings":[{"key":"s","type":"enum","title":"S","default":{"nested":1},"options":["a",{"value":"b","hint":"h"}],
		"optionsBy":{"setting":"s","options":{"a":[{"value":"x","hint":"h"}]}}}]}`))
	if err != nil {
		t.Fatal(err)
	}
	want := []string{
		`unknown key "contributes.extra"`,
		`unknown key "contributes.tabKinds[0].shape"`,
		`unknown key "settings[0].options[1].hint"`,
		`unknown key "settings[0].optionsBy.options.a[0].hint"`,
		`unknown key "unrelated"`,
	}
	if !slices.Equal(m.Warnings, want) {
		t.Errorf("warnings = %q, want %q", m.Warnings, want)
	}
}

func TestAManifestWithOnlyKnownKeysHasNoWarnings(t *testing.T) {
	m, err := manifestjson.Decode([]byte(everyFieldManifest))
	if err != nil {
		t.Fatal(err)
	}
	if len(m.Warnings) != 0 {
		t.Errorf("warnings = %q, want none", m.Warnings)
	}
}
