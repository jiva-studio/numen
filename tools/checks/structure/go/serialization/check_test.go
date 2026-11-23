// Package serialization is the check `go/serialization`, run on the tree with the planted violations that prove it.
package serialization

import (
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/jiva-studio/numen/tools/checks/structure/go/load"
)

// TestNoInnerLayerImportsSerializationOrUsesFormatTags verifies that port/, usecase/, and domain/
// import no wire/storage format packages and carry no serialization struct tags.
func TestNoInnerLayerImportsSerializationOrUsesFormatTags(t *testing.T) {
	wrong, read, err := findSerialization([]layerRoot{
		{dir: "../domain", isImportAllowed: true},
		{dir: "../port", isImportAllowed: false},
		{dir: "../usecase", isImportAllowed: false},
		{dir: "../../../shared/note/domain", isImportAllowed: false},
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, one := range wrong {
		t.Error(one)
	}
	if read == 0 {
		t.Fatal("no file of an inner layer read: the rule is checked against nothing")
	}
}

// The rule rejects a format import in a port, a protobuf subpackage included, and a format tag in a port and in
// the domain; it passes a format import in the domain, a tag of another kind and a test file.
func TestSerializationRule_RejectsFormatImportsAndTags(t *testing.T) {
	root := load.WriteTreeFixture(t, map[string]string{
		"domain/note/note.go":      "package note\n\nimport _ \"encoding/json\"\n\ntype Note struct {\n\tTitle string `json:\"title\"`\n\tBody  string `validate:\"required\"`\n}\n",
		"port/store/store.go":      "package store\n\nimport (\n\t_ \"encoding/json\"\n\t_ \"google.golang.org/protobuf/proto\"\n)\n\ntype Row struct {\n\tID string `db:\"id\"`\n}\n",
		"port/store/store_test.go": "package store\n\nimport _ \"encoding/xml\"\n\ntype fixture struct {\n\tName string `xml:\"name\"`\n}\n",
	})
	domain := filepath.Join(root, "domain")
	port := filepath.Join(root, "port")
	wrong, read, err := findSerialization([]layerRoot{
		{dir: domain, isImportAllowed: true},
		{dir: port, isImportAllowed: false},
	})
	if err != nil {
		t.Fatal(err)
	}
	notePath := filepath.Join(domain, "note", "note.go")
	storePath := filepath.Join(port, "store", "store.go")
	want := []string{
		notePath + " Note.Title has serialization tag `json:\"title\"`: format tags belong on adapter DTOs",
		storePath + " imports encoding/json: inner layers do not parse wire or storage formats",
		storePath + " imports google.golang.org/protobuf/proto: inner layers do not parse wire or storage formats",
		storePath + " Row.ID has serialization tag `db:\"id\"`: format tags belong on adapter DTOs",
	}
	if !slices.Equal(wrong, want) {
		t.Errorf("the rule rejects %v, want %v", wrong, want)
	}
	if read != 2 {
		t.Errorf("%d files read, want 2", read)
	}
}

func TestMain(m *testing.M) {
	os.Exit(load.RunTests(m))
}
