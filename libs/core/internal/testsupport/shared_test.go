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

// A service is a file, and the shared file is not a service file: it declares
// no service and imports nothing, so it is the base of the import graph and
// every other file depends on it. Both properties underlie the rule for adding
// a type to it: a file that declared a service or imported another file would
// own a subject and would be a service file.
//
// buf's STANDARD rule set and its other categories do not check either
// property, so the assertion is here, next to the other walk of the schema.
func TestTheSharedFileDeclaresNoServiceAndImportsNothing(t *testing.T) {
	var target protoreflect.FileDescriptor
	files := 0
	protoregistry.GlobalFiles.RangeFiles(func(file protoreflect.FileDescriptor) bool {
		if file.Package() != "numen.v1" {
			return true
		}
		files++
		if file.Path() == shared {
			target = file
		}
		return true
	})

	// A walk that read none of the schema, or one that did not find the file
	// the rule is about, is a rule checked against nothing. The floor is well
	// under the files the schema holds.
	if files < 10 {
		t.Fatalf("%d files of the schema read: the walk is not reading it", files)
	}
	if target == nil {
		t.Fatalf("%s is not in the schema: the walk is reading something else", shared)
	}

	for _, why := range describeOwnership(target) {
		t.Errorf("%s %s, and the file three services share owns no subject", shared, why)
	}
}

// Applied to the schema's own files, the rule must report the two violations
// separately: a file that declares a service, and a file that imports another
// file.
func TestSharedFileRule_ReportsViolations(t *testing.T) {
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
			"declares SearchService",
			"imports numen/v1/note.proto", "imports numen/v1/shared.proto",
		}},
	} {
		file, err := protoregistry.GlobalFiles.FindFileByPath(one.path)
		if err != nil {
			t.Fatalf("%s: %v", one.path, err)
		}
		if got := describeOwnership(file); !slices.Equal(got, one.why) {
			t.Errorf("%s owns %v, and the rule reads %v", one.path, one.why, got)
		}
	}
}

// describeOwnership lists every way a schema file owns a subject: the services
// it declares, and the files it imports. A file with none can be imported by
// every other file.
func describeOwnership(file protoreflect.FileDescriptor) []string {
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
