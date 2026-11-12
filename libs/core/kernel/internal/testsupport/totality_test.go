package testsupport

import (
	"testing"

	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"

	_ "github.com/jiva-studio/numen/libs/core/protocol/gen/numen/v1"
)

func getWalkedMessages() map[protoreflect.FullName]string {
	return map[protoreflect.FullName]string{
		"numen.v1.CatalogErrorCode":            "adapter/window/catalog: written by newWireErrorCode",
		"numen.v1.CatalogIncompatibility":      "adapter/window/catalog: written by newWireIncompatibility",
		"numen.v1.CatalogState":                "adapter/window/catalog: written by newWireState",
		"numen.v1.CommandErrorCode":            "internal/adapter/plugin/commandbridge: read by RunCommand",
		"numen.v1.ErrorCode":                   "adapter/window/editor: worded by note errors",
		"numen.v1.FlushResult":                 "internal/adapterlib/wire: read by left",
		"numen.v1.HostIndexState":              "plugin/sdk-go/host: read by EnsureIndexed",
		"numen.v1.HostSearchErrorCode":         "plugin/sdk-go/host: read by readSearchError",
		"numen.v1.HostStoreErrorCode":          "plugin/sdk-go/host: read by readStoreError",
		"numen.v1.HostVaultErrorCode":          "plugin/sdk-go/host: read by fail",
		"numen.v1.HostWindowErrorCode":         "plugin/sdk-go/host: read by readWindowError",
		"numen.v1.DependencyKind":              "internal/adapter/plugin: walked by TestDependencyKindsRoundTripTheWire",
		"numen.v1.DependencyState":             "internal/adapter/plugin: walked by TestDependencyStatesAreWrittenToTheWire",
		"numen.v1.Mode":                        "internal/adapterlib/wire: round trip through ConvertModeFromWire and ConvertModeToWire",
		"numen.v1.NameSource":                  "adapter/window/editor: written by newWireNameSource",
		"numen.v1.ProviderConnectionErrorCode": "adapter/window/connection: written by getErrorCode",
		"numen.v1.HostModelErrorCode":          "adapter/plugin/hostendpoint: written by getModelErrorCode",
		"numen.v1.HostModelChatFinishReason":   "adapter/plugin/hostendpoint: written by writeChatFinishReason",
		"numen.v1.ProviderChatFinishReason":    "internal/adapter/plugin/providerbridge: read by readFinishReason",
		"numen.v1.ModelErrorCode":              "adapter/window/model: written by getErrorCode",
		"numen.v1.Priority":                    "adapter/plugin/hostendpoint: read by readPriority, plugin/sdk-go/connection: written by writePriority",
		"numen.v1.ProviderFailureKind":         "internal/adapter/plugin/providerbridge: read by readFailure",
		"numen.v1.ProviderFailureReason":       "internal/adapter/plugin/providerbridge: read by readReason",
		"numen.v1.PluginEventErrorCode":        "internal/adapter/plugin: read by readEventRefusal",
		"numen.v1.Presence":                    "adapter/window/editor: written by the presences table",
		"numen.v1.Role":                        "internal/adapterlib/wire: round trip through ConvertRoleFromWire and ConvertRoleToWire",
		"numen.v1.EmbedRole":                   "plugin/sdk-go/embedding: read by readRole and written by writeRole",
		"numen.v1.Marker":                      "internal/adapterlib/wire: round trip through ConvertMarkerFromWire and ConvertMarkerToWire",
		"numen.v1.SearchMode":                  "adapter/window/editor: read by modeOf",
		"numen.v1.SearchFloorSource":           "plugin/sdk-go/host: read by readFloorSource",
		"numen.v1.Relation":                    "adapter/window/editor: written by convertRelationToWire",
		"numen.v1.Origin":                      "adapter/window/theme: written by encodeOrigin",
		"numen.v1.TextErrorCode":               "internal/adapter/plugin/textbridge: read by readError",
		"numen.v1.TabMoveErrorCode":            "adapter/window/editor: written by convertMoveErrorToWire",
		"numen.v1.TaskKind":                    "internal/adapterlib/wire: written by convertKind",
		"numen.v1.ToolEffect":                  "plugin/sdk-go/tools: written by writeEffect",
		"numen.v1.ToolErrorCode":               "plugin/sdk-go/tools: written by CallTool",
		"numen.v1.ToolServer":                  "plugin/sdk-go/tools: written by writeServers",
		"numen.v1.ToolStep":                    "plugin/sdk-go/tools: written by writeStep",
		"numen.v1.Unit":                        "internal/adapterlib/wire: written by unitOf",
		"numen.v1.VaultsErrorCode":             "adapter/window/editor: written by vaultsErrorCodeBy",
		"numen.v1.WorkState":                   "plugin/sdk-go/host: read by EnqueueWork",
	}
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
			if _, ok := getWalkedMessages()[name]; !ok {
				t.Errorf("%s is in the schema and nothing walks its values", name)
			}
		}
		return true
	})
	for name := range getWalkedMessages() {
		if !target[name] {
			t.Errorf("%s is walked here and is not in the schema", name)
		}
	}
}

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
