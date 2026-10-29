package container

import (
	"reflect"
	"testing"
)

// A section is placed under the key the document spells on the field that holds
// it, and what a model row writes is placed under that same key. A key that
// appears twice is a row writing a setting nothing reads.
func TestEverySectionKeyIsTheOneTheDocumentSpells(t *testing.T) {
	for _, one := range []struct {
		in         any
		field      string
		targetPath string
	}{
		{Settings{}, "Indexing", indexingAt},
		{Settings{}, "Agent", agentAt},
		{Indexing{}, "Embedding", embeddingAt},
	} {
		field, there := reflect.TypeOf(one.in).FieldByName(one.field)
		if !there {
			t.Errorf("%T holds no %s", one.in, one.field)
			continue
		}
		if key := field.Tag.Get("json"); key != one.targetPath {
			t.Errorf("%T.%s is at %q and is read under %q", one.in, one.field, key, one.targetPath)
		}
	}
}
