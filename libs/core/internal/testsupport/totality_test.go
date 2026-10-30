package testsupport

import (
	"testing"

	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"

	_ "github.com/jiva-studio/numen/libs/protocol/gen/numen/v1"
)

// walked is every enum of the schema and where its values are walked one by
// one. The tests themselves are in the packages that hold the mappings, because
// a mapping is unexported and belongs to its own package.
//
// This list is maintained manually, and it is a list of enums and not of
// values: an enum added to the schema and missing here fails below, and a
// value added to an enum already listed fails in the test named for that enum.
var walked = map[protoreflect.FullName]string{
	"numen.v1.CommandErrorCode":     "internal/adapter/plugin/commandbridge: read by RunCommand",
	"numen.v1.CompletionConsent":    "adapter/window/editor: written by the consents table",
	"numen.v1.CompletionErrorCode":  "plugin-sdk-go/ai: read by readRefusal",
	"numen.v1.ErrorCode":            "adapter/window/editor: worded by note errors",
	"numen.v1.CalibrationOutcome":   "adapter/window/editor/search: walked by TestEveryCalibrationOutcomeHasItsOwnWireValue",
	"numen.v1.FloorSource":          "adapter/window/editor/search: walked by TestEveryFloorSourceHasItsOwnWireValue",
	"numen.v1.FlushResult":          "internal/adapterlib/wire: read by left",
	"numen.v1.HostIndexState":       "plugin-sdk-go/host: read by EnsureIndexed",
	"numen.v1.HostStoreErrorCode":   "plugin-sdk-go/host: read by readStoreError",
	"numen.v1.HostVaultErrorCode":   "plugin-sdk-go/host: read by fail",
	"numen.v1.DependencyKind":       "internal/adapter/plugin: walked by TestDependencyKindsRoundTripTheWire",
	"numen.v1.DependencyState":      "internal/adapter/plugin: walked by TestDependencyStatesAreWrittenToTheWire",
	"numen.v1.Mode":                 "internal/adapterlib/wire: round trip through ConvertModeFromWire and ConvertModeToWire",
	"numen.v1.NameSource":           "adapter/window/editor: written by newWireNameSource",
	"numen.v1.PluginEventErrorCode": "internal/adapter/plugin: read by readEventRefusal",
	"numen.v1.Presence":             "adapter/window/editor: written by the presences table",
	"numen.v1.Role":                 "internal/adapterlib/wire: round trip through ConvertRoleFromWire and ConvertRoleToWire",
	"numen.v1.Marker":               "internal/adapterlib/wire: round trip through ConvertMarkerFromWire and ConvertMarkerToWire",
	"numen.v1.SearchMode":           "adapter/window/editor: read by modeOf",
	"numen.v1.Relation":             "adapter/window/editor: written by convertRelationToWire",
	"numen.v1.Origin":               "internal/adapter/theme: written by encodeOrigin",
	"numen.v1.TextErrorCode":        "internal/adapter/plugin/textbridge: read by readError",
	"numen.v1.ToolEffect":           "plugin-sdk-go/tools: written by writeEffect",
	"numen.v1.ToolErrorCode":        "plugin-sdk-go/tools: written by CallTool",
	"numen.v1.ToolServer":           "plugin-sdk-go/tools: written by writeServers",
	"numen.v1.ToolStep":             "plugin-sdk-go/tools: written by writeStep",
	"numen.v1.Unit":                 "internal/adapterlib/wire: written by unitOf",
	"numen.v1.VaultsErrorCode":      "adapter/window/editor: written by vaultsErrorCodeBy",
	"numen.v1.WorkState":            "plugin-sdk-go/host: read by EnqueueWork",
}

// TestEveryEnumOfTheSchemaIsWalked. An enum nothing walks is an enum whose next
// value can be added, generated, carried and read without anything anywhere
// acting on it.
func TestEveryEnumOfTheSchemaIsWalked(t *testing.T) {
	target := map[protoreflect.FullName]bool{}
	protoregistry.GlobalFiles.RangeFiles(func(file protoreflect.FileDescriptor) bool {
		if file.Package() != "numen.v1" {
			return true
		}
		for _, name := range enumsOf(file) {
			target[name] = true
			if _, ok := walked[name]; !ok {
				t.Errorf("%s is in the schema and nothing walks its values", name)
			}
		}
		return true
	})
	for name := range walked {
		if !target[name] {
			t.Errorf("%s is walked here and is not in the schema", name)
		}
	}
}

// enumsOf is every enum a file declares, inside its messages as well as beside
// them.
func enumsOf(file protoreflect.FileDescriptor) []protoreflect.FullName {
	var out []protoreflect.FullName
	for i := range file.Enums().Len() {
		out = append(out, file.Enums().Get(i).FullName())
	}
	var walk func(protoreflect.MessageDescriptors)
	walk = func(messages protoreflect.MessageDescriptors) {
		for i := range messages.Len() {
			message := messages.Get(i)
			for j := range message.Enums().Len() {
				out = append(out, message.Enums().Get(j).FullName())
			}
			walk(message.Messages())
		}
	}
	walk(file.Messages())
	return out
}
