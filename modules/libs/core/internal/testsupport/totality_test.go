package testsupport

import (
	"testing"

	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"

	_ "github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1"
)

// walked is every enum of the schema and where its values are walked one by
// one. The tests themselves are in the packages that hold the mappings, because
// a mapping is unexported and belongs to its own package.
//
// This list is the one thing here that is written by hand, and it is a list of
// enums and not of values: an enum added to the schema and forgotten fails
// below, and a value added to an enum already here fails in the test named
// beside it.
var walked = map[protoreflect.FullName]string{
	"numen.v1.Counts":        "internal/wire: round trip through CountsIn and CountsOf",
	"numen.v1.Fault":         "adapter/webui: written by faultOf",
	"numen.v1.FlushResult":   "internal/wire: read by left",
	"numen.v1.Goal":          "internal/wire: round trip through GoalIn and GoalOf",
	"numen.v1.Mode":          "internal/wire: round trip through ModeIn and ModeOf",
	"numen.v1.NamedBy":       "adapter/webui: written by namedByOf",
	"numen.v1.NoteType":      "adapter/webui: written by typeOf",
	"numen.v1.Presence":      "adapter/webui: written by the standing table",
	"numen.v1.Rating":        "adapter/flashcardsui: read by rating",
	"numen.v1.Refusal":       "adapter/mcp: worded by said",
	"numen.v1.Role":          "adapter/webui: read by roleOf",
	"numen.v1.Rule":          "internal/wire: round trip through RuleIn and RuleOf",
	"numen.v1.Seat":          "adapter/webui: written by seatOf",
	"numen.v1.Shelf":         "internal/adapter/theme: written by shelved",
	"numen.v1.SourceKind":    "adapter/webui: written by kindOf",
	"numen.v1.State":         "adapter/webui: written by stood and by run",
	"numen.v1.StopReason":    "internal/wire: written by StopReasonOf",
	"numen.v1.Unit":          "internal/wire: written by unitOf",
	"numen.v1.VaultsRefusal": "adapter/webui: written by vaultRefusedBy",
	"numen.v1.Way":           "adapter/webui: read by wayOf",
}

// TestEveryEnumOfTheSchemaIsWalked. An enum nothing walks is an enum whose next
// value can be added, generated, carried and read without anything anywhere
// acting on it.
func TestEveryEnumOfTheSchemaIsWalked(t *testing.T) {
	held := map[protoreflect.FullName]bool{}
	protoregistry.GlobalFiles.RangeFiles(func(file protoreflect.FileDescriptor) bool {
		if file.Package() != "numen.v1" {
			return true
		}
		for _, name := range enumsOf(file) {
			held[name] = true
			if _, ok := walked[name]; !ok {
				t.Errorf("%s is in the schema and nothing walks its values", name)
			}
		}
		return true
	})
	for name := range walked {
		if !held[name] {
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
