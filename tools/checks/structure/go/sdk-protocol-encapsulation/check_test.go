package sdkprotocolencapsulation

import (
	"slices"
	"strings"
	"testing"

	"github.com/jiva-studio/numen/tools/checks/structure/go/load"
)

// The rule reports a protocol type in every position of an exported identifier and accepts the
// positions a user cannot name.
func TestSDKProtocolRule_RejectsEveryExportedPositionOfAProtocolType(t *testing.T) {
	dir := load.WriteFiles(t, map[string]string{
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
	dir := load.WriteFiles(t, map[string]string{
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
	dir := load.WriteFiles(t, map[string]string{
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

func TestTheSDKNamesNoProtocolType(t *testing.T) {
	if testing.Short() {
		t.Skip("loads and type-checks the modules of the workspace")
	}
	checkSDKNamesNoProtocolType(t, load.Workspace(t))
}
