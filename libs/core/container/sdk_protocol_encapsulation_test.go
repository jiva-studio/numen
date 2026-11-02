package container

import (
	"go/types"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"golang.org/x/tools/go/packages"
)

// protocolLeak is a protocol type mentioned by the type of an exported identifier of a public package.
type protocolLeak struct {
	// subject is the qualified name of the exported identifier or the exported type that mentions the protocol type.
	subject string
	// protocolType is the qualified name of the protocol type.
	protocolType string
}

// leakWalk collects the protocol types that the public API of one set of packages mentions.
type leakWalk struct {
	// protocolRoot is the import path of the protocol module; every package at or below it is a
	// protocol package.
	protocolRoot string
	// familyPrefix is the import path prefix of the packages whose declarations are part of the API.
	familyPrefix string
	subject      string
	visited      map[*types.TypeName]bool
	found        map[protocolLeak]bool
}

// isInternalPackage reports whether path names a package below an internal directory.
func isInternalPackage(path string) bool {
	return slices.Contains(strings.Split(path, "/"), "internal")
}

// isProtocolPackage reports whether the package path is a protocol package.
func (w *leakWalk) isProtocolPackage(path string) bool {
	return isWithin(path, w.protocolRoot)
}

// walkNamed records a protocol type, and descends into the declaration of a type of the family.
func (w *leakWalk) walkNamed(named *types.Named) {
	origin := named.Origin().Obj()
	if origin.Pkg() == nil {
		return
	}
	if w.isProtocolPackage(origin.Pkg().Path()) {
		w.found[protocolLeak{subject: w.subject, protocolType: origin.Pkg().Path() + "." + origin.Name()}] = true
		return
	}
	for i := range named.TypeArgs().Len() {
		w.walk(named.TypeArgs().At(i))
	}
	if !strings.HasPrefix(origin.Pkg().Path(), w.familyPrefix) || w.visited[origin] {
		return
	}
	w.visited[origin] = true
	outer := w.subject
	w.subject = origin.Pkg().Path() + "." + origin.Name()
	defer func() { w.subject = outer }()
	w.walk(named.Origin().Underlying())
	methods := types.NewMethodSet(types.NewPointer(named.Origin()))
	for i := range methods.Len() {
		if method := methods.At(i).Obj(); method.Exported() {
			w.walk(method.Type())
		}
	}
}

// walk records the protocol types that t mentions: in a field or method that a user of t can name, in a
// signature, an element type, a type argument or a constraint.
func (w *leakWalk) walk(t types.Type) {
	switch current := types.Unalias(t).(type) {
	case *types.Named:
		w.walkNamed(current)
	case *types.Pointer:
		w.walk(current.Elem())
	case *types.Slice:
		w.walk(current.Elem())
	case *types.Array:
		w.walk(current.Elem())
	case *types.Chan:
		w.walk(current.Elem())
	case *types.Map:
		w.walk(current.Key())
		w.walk(current.Elem())
	case *types.Struct:
		for i := range current.NumFields() {
			if field := current.Field(i); field.Exported() {
				w.walk(field.Type())
			}
		}
	case *types.Tuple:
		for i := range current.Len() {
			w.walk(current.At(i).Type())
		}
	case *types.Signature:
		w.walk(current.Params())
		w.walk(current.Results())
		for i := range current.TypeParams().Len() {
			w.walk(current.TypeParams().At(i).Constraint())
		}
	case *types.Interface:
		for i := range current.NumExplicitMethods() {
			if method := current.ExplicitMethod(i); method.Exported() {
				w.walk(method.Type())
			}
		}
		for i := range current.NumEmbeddeds() {
			w.walk(current.EmbeddedType(i))
		}
	case *types.Union:
		for i := range current.Len() {
			w.walk(current.Term(i).Type())
		}
	case *types.TypeParam:
		w.walk(current.Constraint())
	}
}

// getProtocolLeaks returns the protocol types that the exported identifiers of the public (non-internal)
// packages of pkgs mention in their types, along with the number of exported identifiers examined.
// A protocol type is a type declared in a package whose path is protocolRoot or one below it; the API
// includes every type declared under familyPrefix that an exported identifier reaches.
func getProtocolLeaks(pkgs []*packages.Package, protocolRoot, familyPrefix string) ([]protocolLeak, int) {
	walk := &leakWalk{
		protocolRoot: protocolRoot,
		familyPrefix: familyPrefix,
		visited:      map[*types.TypeName]bool{},
		found:        map[protocolLeak]bool{},
	}
	examined := 0
	for _, pkg := range pkgs {
		if pkg.Types == nil || isInternalPackage(pkg.PkgPath) || walk.isProtocolPackage(pkg.PkgPath) {
			continue
		}
		scope := pkg.Types.Scope()
		for _, name := range scope.Names() {
			object := scope.Lookup(name)
			if !object.Exported() {
				continue
			}
			examined++
			walk.subject = pkg.PkgPath + "." + name
			walk.walk(object.Type())
		}
	}
	leaks := make([]protocolLeak, 0, len(walk.found))
	for one := range walk.found {
		leaks = append(leaks, one)
	}
	slices.SortFunc(leaks, func(a, b protocolLeak) int {
		if byName := strings.Compare(a.subject, b.subject); byName != 0 {
			return byName
		}
		return strings.Compare(a.protocolType, b.protocolType)
	})
	return leaks, examined
}

// loadAPIPackages loads the packages of the module in dir with their types, test files excluded.
func loadAPIPackages(t *testing.T, dir string, env []string) []*packages.Package {
	t.Helper()
	config := &packages.Config{
		Dir:  dir,
		Env:  append(os.Environ(), env...),
		Mode: packages.NeedName | packages.NeedFiles | packages.NeedCompiledGoFiles | packages.NeedImports | packages.NeedDeps | packages.NeedTypes | packages.NeedSyntax | packages.NeedTypesInfo,
	}
	loaded, err := packages.Load(config, "./...")
	if err != nil {
		t.Fatal(err)
	}
	for _, pkg := range loaded {
		for _, one := range pkg.Errors {
			t.Errorf("%s does not load: %v", pkg.ID, one)
		}
	}
	return loaded
}

// The exported identifiers of the public packages of the plugin SDK and its feature modules mention no
// type declared in the protocol module: a plugin that compiles against the SDK names no protocol type.
func checkSDKNamesNoProtocolType(t *testing.T, workspace workspacePackages) {
	t.Helper()
	var pkgs []*packages.Package
	modules := 0
	root, err := filepath.Abs(filepath.Join(numenModulesDir, "libs", "plugin-sdk-go"))
	if err != nil {
		t.Fatal(err)
	}
	for _, dir := range listWorkspaceModules(t) {
		relative, err := filepath.Rel(root, dir)
		if err != nil {
			t.Fatal(err)
		}
		if strings.HasPrefix(relative, "..") {
			continue
		}
		modules++
		for _, pkg := range workspace.listModule(t, dir) {
			if isTestVariant(pkg) {
				continue
			}
			for _, one := range pkg.Errors {
				t.Errorf("%s does not load: %v", pkg.ID, one)
			}
			pkgs = append(pkgs, pkg)
		}
	}
	leaks, examined := getProtocolLeaks(pkgs, wire, sdkModule)
	for _, one := range leaks {
		t.Errorf("%s mentions the protocol type %s", one.subject, one.protocolType)
	}

	// A load that finds few modules, packages or identifiers, or an SDK that never imports the
	// protocol, checks nothing.
	importsProtocol := 0
	for _, pkg := range pkgs {
		for path := range pkg.Imports {
			if isWithin(path, wire) {
				importsProtocol++
				break
			}
		}
	}
	if modules < 3 || len(pkgs) < 20 || examined < 100 || importsProtocol < 5 {
		t.Fatalf("%d modules, %d packages, %d exported identifiers and %d packages importing the protocol: the rule is checked against nothing", modules, len(pkgs), examined, importsProtocol)
	}
}

// writeFiles writes the files, keyed by slash-separated path, into a new module named fixture.
func writeFiles(t *testing.T, files map[string]string) string {
	t.Helper()
	dir := t.TempDir()
	files["go.mod"] = "module fixture\n\ngo 1.26\n"
	for name, text := range files {
		target := filepath.Join(dir, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(target), 0o750); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(target, []byte(text), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	return dir
}

// The rule reports a protocol type in every position of an exported identifier and accepts the
// positions a user cannot name.
func TestSDKProtocolRule_RejectsEveryExportedPositionOfAProtocolType(t *testing.T) {
	dir := writeFiles(t, map[string]string{
		"gen/gen.go": `package gen

type Message struct{ Text string }

type Level int
`,
		"api/api.go": `package api

import "fixture/gen"

func Param(m *gen.Message) {}

func Result() gen.Level { return 0 }

func Multiple() (int, []gen.Message, error) { return 0, nil, nil }

var Shared gen.Message

const Strong gen.Level = 1

type Alias = gen.Message

type Field struct {
	Body gen.Message
	internalBody gen.Message
}

type Embedded struct{ *gen.Message }

type Lookup map[string]chan gen.Level

type Contract interface {
	Send(gen.Message) error
	receive(gen.Message)
}

type Holder struct{}

func (Holder) Method() gen.Message { return gen.Message{} }

func (Holder) private(m gen.Message) {}

type Box[T any] struct{ Value T }

func Boxed() Box[gen.Message] { return Box[gen.Message]{} }

func Generic[T gen.Message | gen.Level](v T) {}

func ReachImpl() *unexportedImpl { return nil }

type unexportedImpl struct{ Body gen.Level }

func (*unexportedImpl) Read() gen.Message { return gen.Message{} }
`,
		"api/clean.go": `package api

import "fixture/gen"

func unexported(m gen.Message) {}

var unexportedVar gen.Message

type Clean struct {
	Name string
	body gen.Message
}

func (Clean) Name2() string { return "" }

func (Clean) hide(m gen.Message) {}

func New() *Clean { return &Clean{} }
`,
		"api/clean_test.go": `package api

import "fixture/gen"

func TestOnly(m gen.Message) {}

func Exported(m gen.Message) {}
`,
		"internal/box/box.go": `package box

import "fixture/gen"

func Convert(m gen.Message) gen.Message { return m }
`,
		"cmd/tool/main.go": `package main

func main() {}
`,
	})
	pkgs := loadAPIPackages(t, dir, []string{"GOWORK=off"})
	leaks, examined := getProtocolLeaks(pkgs, "fixture/gen", "fixture/")
	var got []string
	for _, one := range leaks {
		got = append(got, strings.TrimPrefix(one.subject, "fixture/")+" "+strings.TrimPrefix(one.protocolType, "fixture/"))
	}
	want := []string{
		"api.Alias gen.Message",
		"api.Boxed gen.Message",
		"api.Contract gen.Message",
		"api.Embedded gen.Message",
		"api.Field gen.Message",
		"api.Generic gen.Level",
		"api.Generic gen.Message",
		"api.Holder gen.Message",
		"api.Lookup gen.Level",
		"api.Multiple gen.Message",
		"api.Param gen.Message",
		"api.Result gen.Level",
		"api.Shared gen.Message",
		"api.Strong gen.Level",
		"api.unexportedImpl gen.Level",
		"api.unexportedImpl gen.Message",
	}
	if !slices.Equal(got, want) {
		t.Errorf("leaks %v, want %v", got, want)
	}
	if examined < 15 {
		t.Errorf("%d exported identifiers examined, want at least 15", examined)
	}
}

// The rule reads nothing from a package below an internal directory or from a test file.
func TestSDKProtocolRule_AcceptsAnAPIWithoutProtocolTypes(t *testing.T) {
	dir := writeFiles(t, map[string]string{
		"gen/gen.go": `package gen

type Message struct{ Text string }
`,
		"api/api.go": `package api

import "fixture/gen"

type Event struct {
	Name string
	text gen.Message
}

func (Event) Describe() string { return "" }

func convert(m gen.Message) Event { return Event{Name: m.Text} }

func New(name string) Event { return convert(gen.Message{Text: name}) }
`,
		"internal/box/box.go": `package box

import "fixture/gen"

func Convert(m gen.Message) gen.Message { return m }
`,
	})
	leaks, examined := getProtocolLeaks(loadAPIPackages(t, dir, []string{"GOWORK=off"}), "fixture/gen", "fixture/")
	if len(leaks) != 0 {
		t.Errorf("leaks %v, want none", leaks)
	}
	if examined < 2 {
		t.Errorf("%d exported identifiers examined, want at least 2", examined)
	}
}

// A type of any package of the protocol module is a protocol type, the generated ones and the ones
// written by hand beside them; a package whose path only starts like the module is not.
func TestSDKProtocolRule_RejectsTypesOfEveryPackageOfTheProtocolModule(t *testing.T) {
	dir := writeFiles(t, map[string]string{
		"protocol/gen/gen.go": `package gen

type Message struct{ Text string }
`,
		"protocol/schema/schema.go": `package schema

type Enum interface{ String() string }
`,
		"protocolx/other.go": `package protocolx

type Other struct{}
`,
		"api/api.go": `package api

import (
	"fixture/protocol/gen"
	"fixture/protocol/schema"
	"fixture/protocolx"
)

func Send(m gen.Message) {}

func Check(e schema.Enum) {}

func Keep(o protocolx.Other) {}
`,
	})
	leaks, _ := getProtocolLeaks(loadAPIPackages(t, dir, []string{"GOWORK=off"}), "fixture/protocol", "fixture/api")
	var got []string
	for _, one := range leaks {
		got = append(got, strings.TrimPrefix(one.subject, "fixture/")+" "+strings.TrimPrefix(one.protocolType, "fixture/"))
	}
	want := []string{"api.Check protocol/schema.Enum", "api.Send protocol/gen.Message"}
	if !slices.Equal(got, want) {
		t.Errorf("leaks %v, want %v", got, want)
	}
}
