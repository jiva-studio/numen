package testsupport

import (
	"slices"
	"testing"

	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"
)

// shared is the file of the schema holding the types three or more services
// use. A type enters it only when three services already use it, and everything
// else stays in the file of the service that answers with it.
const shared = "numen/v1/shared.proto"

// A service is a file, and the shared file is not a service's: it declares none
// and it imports nothing, so it is the base of the import graph and everything
// else is above it. Both are what the rule for entering it rests on — a file
// that answered with something, or reached for something, would own a subject
// and would be a service's file after all.
//
// buf's STANDARD set has no rule for either, and neither has any other category
// it carries, so the assertion stands here beside the other walk of the schema.
func TestTheSharedFileDeclaresNoServiceAndImportsNothing(t *testing.T) {
	var held protoreflect.FileDescriptor
	files := 0
	protoregistry.GlobalFiles.RangeFiles(func(file protoreflect.FileDescriptor) bool {
		if file.Package() != "numen.v1" {
			return true
		}
		files++
		if file.Path() == shared {
			held = file
		}
		return true
	})

	// A walk that read none of the schema, or one that did not find the file
	// the rule is about, is a rule checked against nothing. The floor is well
	// under the files the schema holds.
	if files < 10 {
		t.Fatalf("%d files of the schema read: the walk is not reading it", files)
	}
	if held == nil {
		t.Fatalf("%s is not in the schema: the walk is reading something else", shared)
	}

	for _, why := range owning(held) {
		t.Errorf("%s %s, and the file three services share owns no subject", shared, why)
	}
}

// Put over the schema's own files, the rule has to refuse the two halves apart:
// a file that answers with something, and a file that reaches for something.
func TestWhatTheSharedFileRuleRefuses(t *testing.T) {
	for _, one := range []struct {
		path string
		why  []string
	}{
		{path: shared},
		{path: "numen/v1/window.proto", why: []string{"declares WindowService"}},
		{path: "numen/v1/note.proto", why: []string{
			"declares NoteService", "imports numen/v1/file.proto", "imports numen/v1/shared.proto",
		}},
		{path: "numen/v1/search.proto", why: []string{
			"declares SearchService", "imports numen/v1/file.proto",
			"imports numen/v1/note.proto", "imports numen/v1/shared.proto",
		}},
	} {
		file, err := protoregistry.GlobalFiles.FindFileByPath(one.path)
		if err != nil {
			t.Fatalf("%s: %v", one.path, err)
		}
		if got := owning(file); !slices.Equal(got, one.why) {
			t.Errorf("%s owns %v, and the rule reads %v", one.path, one.why, got)
		}
	}
}

// owning is every way a file of the schema owns a subject of its own: the
// services it answers with, and the files it reaches for. A file owning none is
// one every other may stand on.
func owning(file protoreflect.FileDescriptor) []string {
	var out []string
	for i := range file.Services().Len() {
		out = append(out, "declares "+string(file.Services().Get(i).Name()))
	}
	for i := range file.Imports().Len() {
		out = append(out, "imports "+file.Imports().Get(i).Path())
	}
	slices.Sort(out)
	return out
}
