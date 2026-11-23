// Package portsreferenced is the check `go/ports-referenced`, run on the tree with the planted violations that prove it.
package portsreferenced

import (
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/jiva-studio/numen/tools/checks/structure/go/load"
)

// A port is a dependency the core has on something outside it, and a port with no
// caller is dead. An interface left in port/ after its last caller was removed is
// unused indirection, and the composition root keeps binding an adapter to it.
//
// Test files are read: a port called only from a test counts as used.
func TestPorts_EveryPortTypeIsReferenced(t *testing.T) {
	unasked, err := getUnnamedTypes("..", filepath.Join("..", "port"), true)
	if err != nil {
		t.Fatal(err)
	}
	for _, one := range unasked {
		t.Errorf("port.%s is declared and nothing asks for it", one)
	}
}

// TestPortTypesAreReferencedOutsidePort checks that every type a port
// declares is used outside port/; a word nothing outside uses belongs to no
// port and outlives the port that used it.
// The rule does not decide which words belong here: domain versus boundary is
// judged by reading.
func TestPortTypesAreReferencedOutsidePort(t *testing.T) {
	unnamed, err := getUnnamedTypes("..", filepath.Join("..", "port"), false)
	if err != nil {
		t.Fatal(err)
	}
	for _, one := range unnamed {
		t.Errorf("port.%s is declared and nothing outside port/ names it", one)
	}
}

// With port/ read as the whole tree, nothing outside it names anything, and the
// rule must return the names below. This confirms the passes above pass on naming
// and not on an empty walk.
func TestPortRule_ReportsUnreferencedPortTypes(t *testing.T) {
	pos := filepath.Join("..", "port")
	unnamed, err := getUnnamedTypes(pos, pos, false)
	if err != nil {
		t.Fatal(err)
	}
	for _, one := range []string{"NoteRefs", "VaultReaders"} {
		if !slices.Contains(unnamed, one) {
			t.Errorf("nothing outside names port.%s and the rule does not refuse it", one)
		}
	}
}

// TestPortMethodsAreCalledOutsidePort checks that every port method has a
// caller that calls it through the port. A method with no caller is never exercised and a defect in it goes unnoticed.
// Test files are not read here, and that asymmetry is the rule: a method called
// only by a test exists to be tested. The two rules above read tests.
func TestPortMethodsAreCalledOutsidePort(t *testing.T) {
	consumers := portConsumers{}
	uncalled, err := getUncalledMethods("..", filepath.Join("..", "port"), consumers)
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range consumers.listSingleConsumerContracts() {
		t.Errorf("%s belongs to its consumer: %v", name, consumers[name])
	}
	for _, one := range uncalled {
		t.Errorf("%s is declared and nothing outside a test calls it", one)
	}
}

// Read against port/ as the whole tree, nothing outside it calls anything, and
// the rule has to be restored with the methods below.
func TestPortMethodRule_ReportsUncalledMethods(t *testing.T) {
	pos := filepath.Join("..", "port")
	uncalled, err := getUncalledMethods(pos, pos, nil)
	if err != nil {
		t.Fatal(err)
	}
	for _, one := range []string{"NoteRefs.GetNotes"} {
		if !slices.Contains(uncalled, one) {
			t.Errorf("nothing outside calls %s and the rule does not refuse it", one)
		}
	}
}

// TestPortWalk_FindsThePlantedPort verifies on a planted tree that the walks find the method nothing
// calls through the port and the type nothing names, and fail on a port folder with no declaration.
// The use case calls Close on a type of its own, which leaves Reader.Close uncalled.
func TestPortWalk_FindsThePlantedPort(t *testing.T) {
	root := load.WriteTreeFixture(t, map[string]string{
		"port/noteport/noteport.go": "package noteport\n\ntype Reader interface {\n\tRead() string\n\tClose()\n}\n\ntype Orphan interface{ Drop() }\n",
		"usecase/note/note.go": "package note\n\nimport \"fixture/port/noteport\"\n\ntype file struct{}\n\nfunc (file) Close() {}\n\n" +
			"func read(r noteport.Reader) string {\n\tfile{}.Close()\n\treturn r.Read()\n}\n",
	})
	port := filepath.Join(root, "port", "noteport")
	consumers := portConsumers{}
	uncalled, err := getUncalledMethods(root, port, consumers, "GOWORK=off")
	if err != nil {
		t.Fatal(err)
	}
	if want := []string{"noteport.Orphan", "noteport.Reader"}; !slices.Equal(consumers.listSingleConsumerContracts(), want) {
		t.Errorf("unshared contracts = %v, want %v", consumers, want)
	}
	if want := []string{"Orphan.Drop", "Reader.Close"}; !slices.Equal(uncalled, want) {
		t.Errorf("the walk reports uncalled methods %v, want %v", uncalled, want)
	}
	unnamed, err := getUnnamedTypes(root, port, false)
	if err != nil {
		t.Fatal(err)
	}
	if want := []string{"Orphan"}; !slices.Equal(unnamed, want) {
		t.Errorf("the walk reports unnamed types %v, want %v", unnamed, want)
	}
	empty := t.TempDir()
	if _, err := getUncalledMethods(empty, empty, nil); err == nil {
		t.Error("a port folder without a declaration passed")
	}
}

// A factory result is consumed even when the use case never spells the port name.
func TestPortOwnershipCountsFactoryResultsAndExcludesFixturesAndWiring(t *testing.T) {
	root := load.WriteTreeFixture(t, map[string]string{
		"port/fileport/file.go":            "package fileport\n type Reader interface{ Read() string }; type Files interface{ Open() (Reader, error) }",
		"usecase/file/file.go":             "package file\n import f \"fixture/port/fileport\"\n func Read(files f.Files) string { r, err := files.Open(); if err != nil { return \"\" }; return r.Read() }",
		"usecase/search/search.go":         "package search\n import f \"fixture/port/fileport\"\n func Read(r f.Reader) string { return r.Read() }",
		"usecase/file/filetest/fixture.go": "package filetest\n import f \"fixture/port/fileport\"\n type Fixture struct { Files f.Files }",
		"adapter/disk/claim.go":            "package disk\n import f \"fixture/port/fileport\"\n type Disk struct{}; func (Disk) Open() (f.Reader,error){ return nil,nil }; var _ f.Files = Disk{}",
		"container/wire.go":                "package container\n import f \"fixture/port/fileport\"\n type Wiring struct { Files f.Files }",
	})
	consumers := portConsumers{}
	_, err := getUncalledMethods(root, filepath.Join(root, "port"), consumers, "GOWORK=off")
	if err != nil {
		t.Fatal(err)
	}
	if want := []string{"fileport.Files"}; !slices.Equal(consumers.listSingleConsumerContracts(), want) {
		t.Fatalf("unshared contracts = %v, want %v", consumers, want)
	}
	if len(consumers["fileport.Files"]) != 1 || !consumers["fileport.Files"]["usecase/file"] {
		t.Fatalf("file factory consumers = %v", consumers["fileport.Files"])
	}
}

func TestMain(m *testing.M) {
	os.Exit(load.RunTests(m))
}
