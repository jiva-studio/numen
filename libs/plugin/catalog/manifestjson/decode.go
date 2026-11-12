package manifestjson

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"maps"
	"slices"

	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/known/structpb"

	v1 "github.com/jiva-studio/numen/libs/core/protocol/gen/numen/v1"
	"github.com/jiva-studio/numen/libs/plugin/catalog/manifest"
)

// Read decodes a manifest from JSON input and validates it.
func Read(r io.Reader) (manifest.Manifest, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return manifest.Manifest{}, fmt.Errorf("decode manifest: %w", err)
	}
	m, err := Decode(data)
	if err != nil {
		return manifest.Manifest{}, fmt.Errorf("decode manifest: %w", err)
	}
	if err := m.Validate(); err != nil {
		return manifest.Manifest{}, err
	}
	if issues := listServiceNameIssues(m); len(issues) > 0 {
		return manifest.Manifest{}, errors.Join(issues...)
	}
	return m, nil
}

// Decode decodes a manifest from JSON without validating the rules between its fields.
func Decode(data []byte) (manifest.Manifest, error) {
	canonical, unknown, err := canonicalize(data)
	if err != nil {
		return manifest.Manifest{}, err
	}
	var wire v1.Manifest
	if err := protojson.Unmarshal(canonical, &wire); err != nil {
		return manifest.Manifest{}, err
	}
	m, err := newManifest(&wire)
	if err != nil {
		return manifest.Manifest{}, err
	}
	for _, path := range unknown {
		m.Warnings = append(m.Warnings, fmt.Sprintf("unknown key %q", path))
	}
	return m, nil
}

var errTrailingData = errors.New("unexpected data after the manifest")

func canonicalize(data []byte) (canonical []byte, unknown []string, err error) {
	decoder := json.NewDecoder(bytes.NewReader(data))
	// gomutants:disable-next-line STATEMENT_REMOVE reason="UseNumber preserves exact numeric tokens before protojson unmarshaling"
	decoder.UseNumber()
	var value any
	if err := decoder.Decode(&value); err != nil {
		return nil, nil, err
	}
	if err := checkSlotOrders(value); err != nil {
		return nil, nil, err
	}
	unknown = pruneUnknown(value, (&v1.Manifest{}).ProtoReflect().Descriptor(), "", nil)
	var rest json.RawMessage
	if err := decoder.Decode(&rest); !errors.Is(err, io.EOF) {
		return nil, nil, errTrailingData
	}
	canonical, err = json.Marshal(value)
	// gomutants:disable-next-line RETURN_ERROR_NIL reason="value decoded from JSON always marshals back to JSON"
	return canonical, unknown, err
}

var errSlotOrderShape = fmt.Errorf("%w: the order of an entry must be a number, not a string", manifest.ErrInvalidSlotEntry)

func checkSlotOrders(root any) error {
	object, isObject := root.(map[string]any)
	if !isObject {
		return nil
	}
	contributes, isObject := object["contributes"].(map[string]any)
	if !isObject {
		return nil
	}
	entries, isList := contributes["slots"].([]any)
	if !isList {
		return nil
	}
	for _, entry := range entries {
		fields, isObject := entry.(map[string]any)
		if !isObject {
			continue
		}
		if text, isText := fields["order"].(string); isText {
			return fmt.Errorf("%w: %q", errSlotOrderShape, text)
		}
	}
	return nil
}

func newManifest(wire *v1.Manifest) (manifest.Manifest, error) {
	settings, err := newSettings(wire.GetSettings())
	if err != nil {
		return manifest.Manifest{}, err
	}
	contributes, err := newContributes(wire.GetContributes())
	if err != nil {
		return manifest.Manifest{}, err
	}
	return manifest.Manifest{
		ID:               wire.GetId(),
		Name:             wire.GetName(),
		Version:          wire.GetVersion(),
		Protocol:         int(wire.GetProtocol()),
		Description:      wire.GetDescription(),
		Capabilities:     wire.GetCapabilities(),
		Targets:          wire.GetTargets(),
		ActivationEvents: wire.GetActivationEvents(),
		Permissions:      wire.GetPermissions(),
		Dependencies:     newDependencies(wire.GetDependencies()),
		Backend:          newBackend(wire.GetBackend()),
		Frontend:         newFrontend(wire.GetFrontend()),
		Settings:         settings,
		Services:         wire.GetServices(),
		SettingsViews:    newSettingsViews(wire.GetSettingsViews()),
		Contributes:      contributes,
		IsOffByDefault:   wire.GetIsOffByDefault(),
	}, nil
}

func newBackend(wire *v1.ManifestBackend) manifest.Backend {
	return manifest.Backend{Entrypoint: wire.GetEntrypoint()}
}

func newFrontend(wire *v1.ManifestFrontend) *manifest.Frontend {
	if wire == nil {
		return nil
	}
	return &manifest.Frontend{}
}

func newDependencies(wire []*v1.ManifestDependency) []manifest.DependencyRequirement {
	var out []manifest.DependencyRequirement
	for _, one := range wire {
		out = append(out, manifest.DependencyRequirement{
			ID:             one.GetId(),
			Kind:           manifest.DependencyKind(one.GetKind()),
			Version:        one.GetVersion(),
			SourceURL:      one.GetSourceUrl(),
			ChecksumSHA256: one.GetChecksumSha256(),
			Metadata:       one.GetMetadata(),
			PluginDir:      "",
		})
	}
	return out
}

func newCondition(wire *v1.ManifestCondition) *manifest.Condition {
	if wire == nil {
		return nil
	}
	return &manifest.Condition{Setting: wire.GetSetting(), Is: wire.GetIs()}
}

func newSettings(wire []*v1.ManifestSetting) ([]manifest.Setting, error) {
	var out []manifest.Setting
	for _, one := range wire {
		options, err := newSettingOptions(one.GetOptions())
		if err != nil {
			return nil, fmt.Errorf("setting %q: %w", one.GetKey(), err)
		}
		optionsBy, err := newOptionsBy(one.GetOptionsBy())
		if err != nil {
			return nil, fmt.Errorf("setting %q: %w", one.GetKey(), err)
		}
		out = append(out, manifest.Setting{
			Key:         one.GetKey(),
			Type:        one.GetType(),
			Title:       one.GetTitle(),
			Description: one.GetDescription(),
			Default:     one.GetDefault().AsInterface(),
			Palette:     one.GetPalette(),
			Icon:        one.GetIcon(),
			Options:     options,
			Min:         one.Min,
			Max:         one.Max,
			Step:        one.Step,
			OptionsBy:   optionsBy,
			When:        newCondition(one.GetWhen()),
			Accept:      one.GetAccept(),
			Multiple:    one.GetMultiple(),
			Preview:     one.GetPreview(),
			OptionsFrom: one.GetOptionsFrom(),
			Generate:    one.GetGenerate(),
			Operations:  one.GetOperations(),
			Discover:    one.GetDiscover(),
		})
	}
	return out, nil
}

var errOptionShape = errors.New("options must be a list of values or of objects with a value")

func newSettingOptions(list *structpb.ListValue) (manifest.SettingOptions, error) {
	var options manifest.SettingOptions
	for _, item := range list.GetValues() {
		switch kind := item.GetKind().(type) {
		case *structpb.Value_StringValue:
			options = append(options, manifest.SettingOption{Value: kind.StringValue, Title: "", Detail: "", Dependencies: nil, When: ""})
		case *structpb.Value_StructValue:
			option, err := newSettingOption(kind.StructValue)
			if err != nil {
				return nil, err
			}
			options = append(options, option)
		default:
			return nil, errOptionShape
		}
	}
	return options, nil
}

func newSettingOption(object *structpb.Struct) (manifest.SettingOption, error) {
	data, err := json.Marshal(object.AsMap())
	// gomutants:disable-next-line BRANCH_IF reason="structpb.Struct map always marshals to JSON"
	if err != nil {
		return manifest.SettingOption{}, fmt.Errorf("%w: %w", errOptionShape, err)
	}
	var wire v1.ManifestSettingOption
	if err := protojson.Unmarshal(data, &wire); err != nil {
		return manifest.SettingOption{}, fmt.Errorf("%w: %w", errOptionShape, err)
	}
	return manifest.SettingOption{
		Value:        wire.GetValue(),
		Title:        wire.GetTitle(),
		Detail:       wire.GetDetail(),
		Dependencies: wire.GetDependencies(),
		When:         wire.GetWhen(),
	}, nil
}

func newOptionsBy(wire *v1.ManifestOptionsBy) (*manifest.OptionsBy, error) {
	if wire == nil {
		return nil, nil
	}
	options := make(map[string]manifest.SettingOptions, len(wire.GetOptions()))
	for value, list := range wire.GetOptions() {
		parsed, err := newSettingOptions(list)
		if err != nil {
			return nil, fmt.Errorf("options under %q: %w", value, err)
		}
		options[value] = parsed
	}
	return &manifest.OptionsBy{Setting: wire.GetSetting(), Options: options}, nil
}

func newSettingsViews(wire []*v1.ManifestSettingsView) []manifest.SettingsView {
	var out []manifest.SettingsView
	for _, one := range wire {
		out = append(out, manifest.SettingsView{ID: one.GetId(), Title: one.GetTitle()})
	}
	return out
}

func pruneUnknown(value any, message protoreflect.MessageDescriptor, path string, unknown []string) []string {
	object, ok := value.(map[string]any)
	if !ok {
		return unknown
	}
	for _, key := range slices.Sorted(maps.Keys(object)) {
		field := message.Fields().ByJSONName(key)
		if field == nil {
			delete(object, key)
			unknown = append(unknown, joinPath(path, key))
			continue
		}
		unknown = pruneField(object[key], field, joinPath(path, key), unknown)
	}
	return unknown
}

func joinPath(path, key string) string {
	if path == "" {
		return key
	}
	return path + "." + key
}

func pruneField(value any, field protoreflect.FieldDescriptor, path string, unknown []string) []string {
	switch {
	case field.IsMap():
		entries, ok := value.(map[string]any)
		if !ok {
			return unknown
		}
		for _, name := range slices.Sorted(maps.Keys(entries)) {
			unknown = pruneEntry(entries[name], field.MapValue(), joinPath(path, name), unknown)
		}
	case field.IsList():
		items, ok := value.([]any)
		if !ok {
			return unknown
		}
		for index, item := range items {
			unknown = pruneEntry(item, field, fmt.Sprintf("%s[%d]", path, index), unknown)
		}
	default:
		return pruneEntry(value, field, path, unknown)
	}
	return unknown
}

func pruneEntry(value any, field protoreflect.FieldDescriptor, path string, unknown []string) []string {
	message := field.Message()
	if message == nil {
		return unknown
	}
	if message.FullName() == listValueName {
		return pruneOptionList(value, path, unknown)
	}
	if message.ParentFile().Package() != "numen.v1" {
		return unknown
	}
	return pruneUnknown(value, message, path, unknown)
}

const listValueName protoreflect.FullName = "google.protobuf.ListValue"

func pruneOptionList(value any, path string, unknown []string) []string {
	items, ok := value.([]any)
	if !ok {
		return unknown
	}
	descriptor := (&v1.ManifestSettingOption{}).ProtoReflect().Descriptor()
	for index, item := range items {
		unknown = pruneUnknown(item, descriptor, fmt.Sprintf("%s[%d]", path, index), unknown)
	}
	return unknown
}
