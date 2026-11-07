package service_test

import (
	"slices"
	"strings"
	"testing"
)

func TestFileCreateSchemaRequiresText(t *testing.T) {
	_, svc := setupTestVault()
	registry, err := svc.Registry()
	if err != nil {
		t.Fatal(err)
	}

	schema := getToolSchema(t, registry, "file_create", false)

	required := requireList(t, "the required list of file_create", schema["required"])
	if !slices.Contains(required, any("text")) {
		t.Errorf("the input schema of file_create requires %v, want text among them", required)
	}
}

func TestFileCreateWithoutTextIsRejectedAsMissing(t *testing.T) {
	v, svc := setupTestVault()
	registry, err := svc.Registry()
	if err != nil {
		t.Fatal(err)
	}

	text := getRejectionText(t, registry, testVault, "file_create", `{"path":"orbits.txt"}`)

	if !strings.Contains(text, "the argument text is missing") {
		t.Errorf("file_create without text gave the error %q, want the argument text named as missing", text)
	}
	requireAbsent(t, v, "orbits.txt")
}

func TestFileCreateWithEmptyTextCreatesAnEmptyFile(t *testing.T) {
	v, svc := setupTestVault()
	registry, err := svc.Registry()
	if err != nil {
		t.Fatal(err)
	}

	callTool(t, registry, testVault, "file_create", `{"path":"orbits.txt","text":""}`)

	requireContent(t, v, "orbits.txt", "")
}
