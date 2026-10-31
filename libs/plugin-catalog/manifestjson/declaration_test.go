package manifestjson

import "testing"

func TestReadDeclaration_TellsAZeroFromAnAbsentKey(t *testing.T) {
	declaration, err := ReadDeclaration([]byte(`{"backend":{"entrypoint":"x"},"lifecycle":{"idleTimeoutSeconds":0}}`))
	if err != nil {
		t.Fatal(err)
	}
	if !declaration.HasBackend() {
		t.Error("the backend section is written")
	}
	if !declaration.HasLifecycleField("idleTimeoutSeconds") {
		t.Error("idleTimeoutSeconds is written as zero")
	}
	if declaration.HasLifecycleField("memoryLimitBytes") {
		t.Error("memoryLimitBytes is not written")
	}
}

func TestReadDeclaration_ReportsNoBackendSection(t *testing.T) {
	declaration, err := ReadDeclaration([]byte(`{"frontend":{"entrypoint":"x"}}`))
	if err != nil {
		t.Fatal(err)
	}
	if declaration.HasBackend() {
		t.Error("no backend section is written")
	}
}

func TestReadDeclaration_RejectsADocumentTheSchemaRejects(t *testing.T) {
	if _, err := ReadDeclaration([]byte(`{"id": 5}`)); err == nil {
		t.Error("a number for the id fits no schema")
	}
}
