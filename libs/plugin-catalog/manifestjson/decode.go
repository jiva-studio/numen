package manifestjson

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"maps"
	"math"
	"slices"

	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/known/structpb"

	"github.com/jiva-studio/numen/libs/plugin-catalog/manifest"
	v1 "github.com/jiva-studio/numen/libs/protocol/gen/numen/v1"
)

// The wire reader takes the keys the schema names; a key it does not name is removed and reported
// before the reader runs.
var wireReader = protojson.UnmarshalOptions{}

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
	return m, nil
}

// Decode decodes a manifest from JSON without validating the rules between its fields.
func Decode(data []byte) (manifest.Manifest, error) {
	canonical, unknown, err := canonicalize(data)
	if err != nil {
		return manifest.Manifest{}, err
	}
	var wire v1.Manifest
	if err := wireReader.Unmarshal(canonical, &wire); err != nil {
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

// canonicalize rewrites the JSON with each key once, the last occurrence winning, which is how
// encoding/json reads a repeated key; the wire reader rejects a repeated key. Keys the schema does
// not name by their JSON name are dropped and returned as paths, in the order of their location.
func canonicalize(data []byte) (canonical []byte, unknown []string, err error) {
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.UseNumber()
	var value any
	if err := decoder.Decode(&value); err != nil {
		return nil, nil, err
	}
	unknown = pruneUnknown(value, (&v1.Manifest{}).ProtoReflect().Descriptor(), "", nil)
	var rest json.RawMessage
	if err := decoder.Decode(&rest); !errors.Is(err, io.EOF) {
		return nil, nil, errTrailingData
	}
	canonical, err = json.Marshal(value)
	return canonical, unknown, err
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
		Secrets:          newSecrets(wire.GetSecrets()),
		Services:         wire.GetServices(),
		Commands:         newCommands(wire.GetCommands()),
		SettingsViews:    newSettingsViews(wire.GetSettingsViews()),
		Contributes:      contributes,
		AppVersion:       manifest.AppVersionRange{Min: wire.GetAppVersion().GetMin(), Max: wire.GetAppVersion().GetMax()},
	}, nil
}

func newBackend(wire *v1.ManifestBackend) manifest.Backend {
	return manifest.Backend{Type: wire.GetType(), Entrypoint: wire.GetEntrypoint(), ServiceName: wire.GetServiceName()}
}

func newFrontend(wire *v1.ManifestFrontend) manifest.Frontend {
	return manifest.Frontend{Entrypoint: wire.GetEntrypoint(), Styles: wire.GetStyles()}
}

func newContributes(wire *v1.ManifestContributes) (manifest.Contributes, error) {
	var kinds []manifest.TabKind
	for _, one := range wire.GetTabKinds() {
		kinds = append(kinds, manifest.TabKind{Kind: one.GetKind(), Title: one.GetTitle(), Icon: one.GetIcon()})
	}
	var fileTypes []manifest.FileType
	for _, one := range wire.GetFileTypes() {
		fileTypes = append(fileTypes, manifest.FileType{TabKind: one.GetTabKind(), Extensions: one.GetExtensions(), FileNames: one.GetFileNames()})
	}
	var themes []manifest.Theme
	for _, one := range wire.GetThemes() {
		themes = append(themes, manifest.Theme{ID: one.GetId(), Name: one.GetName(), Appearance: one.GetAppearance(), File: one.GetFile()})
	}
	var views []manifest.StatusView
	for _, one := range wire.GetStatusViews() {
		views = append(views, manifest.StatusView{ID: one.GetId()})
	}
	noteTypes, err := newNoteTypes(wire.GetNoteTypes())
	if err != nil {
		return manifest.Contributes{}, err
	}
	return manifest.Contributes{
		TabKinds:     kinds,
		Feature:      wire.GetFeature(),
		Category:     wire.GetCategory(),
		FileTypes:    fileTypes,
		Themes:       themes,
		StatusViews:  views,
		Handles:      newHandles(wire.GetHandles()),
		ContentTypes: newDeclaredContentTypes(wire.GetContentTypes()),
		Readers:      newReaders(wire.GetReaders()),
		NoteTypes:    noteTypes,
		FrameOrigins: wire.GetFrameOrigins(),
	}, nil
}

func newDependencies(wire []*v1.ManifestDependency) []manifest.DependencyRequirement {
	if wire == nil {
		return nil
	}
	out := make([]manifest.DependencyRequirement, 0, len(wire))
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

func newSecrets(wire []*v1.ManifestSecret) []manifest.Secret {
	if wire == nil {
		return nil
	}
	out := make([]manifest.Secret, 0, len(wire))
	for _, one := range wire {
		out = append(out, manifest.Secret{
			Name:        one.GetName(),
			Title:       one.GetTitle(),
			Description: one.GetDescription(),
			When:        newCondition(one.GetWhen()),
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
	if wire == nil {
		return nil, nil
	}
	out := make([]manifest.Setting, 0, len(wire))
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
			Min:         one.GetMin(),
			Max:         one.GetMax(),
			Step:        one.GetStep(),
			Replaces:    one.GetReplaces(),
			OptionsBy:   optionsBy,
			When:        newCondition(one.GetWhen()),
			Accept:      one.GetAccept(),
		})
	}
	return out, nil
}

var errOptionShape = errors.New("options must be a list of values or of objects with a value")

// newSettingOptions reads a list of options, each a bare string or an object of ManifestSettingOption.
func newSettingOptions(list *structpb.ListValue) (manifest.SettingOptions, error) {
	if list == nil {
		return nil, nil
	}
	options := make(manifest.SettingOptions, 0, len(list.GetValues()))
	for _, item := range list.GetValues() {
		switch kind := item.GetKind().(type) {
		case *structpb.Value_StringValue:
			options = append(options, manifest.SettingOption{Value: kind.StringValue, Title: "", Detail: "", Dependencies: nil})
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
	if err != nil {
		return manifest.SettingOption{}, fmt.Errorf("%w: %w", errOptionShape, err)
	}
	var wire v1.ManifestSettingOption
	if err := wireReader.Unmarshal(data, &wire); err != nil {
		return manifest.SettingOption{}, fmt.Errorf("%w: %w", errOptionShape, err)
	}
	return manifest.SettingOption{
		Value:        wire.GetValue(),
		Title:        wire.GetTitle(),
		Detail:       wire.GetDetail(),
		Dependencies: wire.GetDependencies(),
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

var errHeadingShape = errors.New("headings must be a list of whole numbers")

func newNoteTypes(wire []*v1.ManifestNoteType) ([]manifest.NoteType, error) {
	if wire == nil {
		return nil, nil
	}
	out := make([]manifest.NoteType, 0, len(wire))
	for _, one := range wire {
		headings, err := newHeadings(one.GetHeadings())
		if err != nil {
			return nil, fmt.Errorf("note type %q: %w", one.GetType(), err)
		}
		levels := make([]int, 0, len(one.GetIdHeadingLevels()))
		for _, level := range one.GetIdHeadingLevels() {
			levels = append(levels, int(level))
		}
		if len(levels) == 0 {
			levels = nil
		}
		out = append(out, manifest.NoteType{
			Type:            one.GetType(),
			Chunks:          one.Chunks,
			Headings:        headings,
			IDHeadingLevels: levels,
			IDPattern:       one.GetIdPattern(),
		})
	}
	return out, nil
}

func newHeadings(list *structpb.ListValue) (*[]int, error) {
	if list == nil {
		return nil, nil
	}
	levels := make([]int, 0, len(list.GetValues()))
	for _, item := range list.GetValues() {
		number, isNumber := item.GetKind().(*structpb.Value_NumberValue)
		if !isNumber || number.NumberValue != math.Trunc(number.NumberValue) {
			return nil, errHeadingShape
		}
		levels = append(levels, int(number.NumberValue))
	}
	return &levels, nil
}

func newCommands(wire []*v1.ManifestCommand) []manifest.Command {
	if wire == nil {
		return nil
	}
	out := make([]manifest.Command, 0, len(wire))
	for _, one := range wire {
		out = append(out, manifest.Command{
			ID:                one.GetId(),
			Title:             one.GetTitle(),
			Icon:              one.GetIcon(),
			ContentTypes:      newContentTypes(one.GetContentTypes()),
			NoteTypes:         one.GetNoteTypes(),
			When:              newCommandWhen(one.GetWhen()),
			Arguments:         newCommandArguments(one.GetArguments()),
			IsHiddenInPalette: one.GetIsHiddenInPalette(),
		})
	}
	return out
}

func newCommandWhen(wire []*v1.ManifestContextCondition) []manifest.ContextCondition {
	if wire == nil {
		return nil
	}
	out := make([]manifest.ContextCondition, 0, len(wire))
	for _, one := range wire {
		out = append(out, manifest.ContextCondition{Key: one.GetKey(), Is: one.GetIs()})
	}
	return out
}

func newCommandArguments(wire []*v1.ManifestCommandArgument) []manifest.CommandArgument {
	if wire == nil {
		return nil
	}
	out := make([]manifest.CommandArgument, 0, len(wire))
	for _, one := range wire {
		out = append(out, manifest.CommandArgument{
			Name:              one.GetName(),
			Title:             one.GetTitle(),
			Type:              one.GetType(),
			IsRequired:        one.GetIsRequired(),
			IsHiddenInPalette: one.GetIsHiddenInPalette(),
		})
	}
	return out
}

func newHandles(wire []*v1.ManifestHandle) []manifest.Handle {
	if wire == nil {
		return nil
	}
	out := make([]manifest.Handle, 0, len(wire))
	for _, one := range wire {
		out = append(out, manifest.Handle{Purpose: one.GetPurpose(), ContentTypes: newContentTypes(one.GetContentTypes()), NoteTypes: one.GetNoteTypes()})
	}
	return out
}

func newDeclaredContentTypes(wire []*v1.ManifestContentType) []manifest.DeclaredContentType {
	if wire == nil {
		return nil
	}
	out := make([]manifest.DeclaredContentType, 0, len(wire))
	for _, one := range wire {
		out = append(out, manifest.DeclaredContentType{Type: newContentType(one.GetType()), Globs: one.GetGlobs()})
	}
	return out
}

func newReaders(wire []*v1.ManifestReader) []manifest.Reader {
	if wire == nil {
		return nil
	}
	out := make([]manifest.Reader, 0, len(wire))
	for _, one := range wire {
		out = append(out, manifest.Reader{ID: one.GetId(), ContentTypes: newContentTypes(one.GetContentTypes()), Rank: newHandlerRank(one.GetRank())})
	}
	return out
}

func newSettingsViews(wire []*v1.ManifestSettingsView) []manifest.SettingsView {
	if wire == nil {
		return nil
	}
	out := make([]manifest.SettingsView, 0, len(wire))
	for _, one := range wire {
		out = append(out, manifest.SettingsView{ID: one.GetId(), Title: one.GetTitle(), After: one.GetAfter()})
	}
	return out
}

// pruneUnknown removes from the decoded JSON value every key that the message does not name by its
// JSON name, at every depth, and returns the path of each key removed. Values of the well-known
// types are left as written, except the options of a setting.
func pruneUnknown(value any, message protoreflect.MessageDescriptor, path string, unknown []string) []string {
	object, isObject := value.(map[string]any)
	if !isObject {
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
		entries, isObject := value.(map[string]any)
		if !isObject {
			return unknown
		}
		for _, name := range slices.Sorted(maps.Keys(entries)) {
			unknown = pruneEntry(entries[name], field.MapValue(), joinPath(path, name), unknown)
		}
	case field.IsList():
		items, isList := value.([]any)
		if !isList {
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

// pruneEntry prunes one value of a field whose type is a message of the schema, or a list of the
// options of a setting.
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

// pruneOptionList prunes each option written as an object in a list of options.
func pruneOptionList(value any, path string, unknown []string) []string {
	items, isList := value.([]any)
	if !isList {
		return unknown
	}
	descriptor := (&v1.ManifestSettingOption{}).ProtoReflect().Descriptor()
	for index, item := range items {
		unknown = pruneUnknown(item, descriptor, fmt.Sprintf("%s[%d]", path, index), unknown)
	}
	return unknown
}
