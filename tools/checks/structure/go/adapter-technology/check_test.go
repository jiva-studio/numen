// Package adaptertechnology is the check `go/adapter-technology`, run on the tree with the planted violations that prove it.
package adaptertechnology

import (
	"go/token"
	"go/types"
	"os"
	"slices"
	"testing"

	"github.com/jiva-studio/numen/tools/checks/structure/go/load"
)

// A technology package gives access to something outside the process: the file system, the network, a child
// process, a database, the operating system, a device or a vendor SDK. The standard time package is not one,
// because reading the clock is a technology only for the clock adapter, which the exceptions list names.
// Third-party packages count, except the ones that only compute.
func TestTechnologyImportRule_NamesWhatReachesOutsideTheProcess(t *testing.T) {
	for path, want := range map[string]bool{
		"os": true, "os/exec": true, "io/fs": true, "net/http": true, "net": true, "database/sql": true,
		"golang.org/x/sys/unix": true, "github.com/rjeczalik/notify": true, "modernc.org/sqlite": true,
		"time": false, "io": false, "context": false, "sync": false, "encoding/json": false, "path/filepath": false,
		"golang.org/x/sync/errgroup": false, "github.com/jiva-studio/numen/libs/note/domain": false,
	} {
		if got := load.IsTechnologyImport(path); got != want {
			t.Errorf("%s is a technology: %v, want %v", path, got, want)
		}
	}
}

// A package of internal/adapter/, adapter/index, adapter/settings or one that implements a port, and calls no use
// case, is a driven adapter and calls a technology, a function or a method of a package that reaches outside
// the process; one that holds
// callbacks, or imports a technology package only for its types, is a seam written as an adapter. A
// package of internal/adapter/ that calls a use case is a driving adapter in the wrong folder.
func TestTechnologyRule_RejectsAnAdapterThatCallsNoTechnology(t *testing.T) {
	dir := load.WriteTreeFixture(t, map[string]string{
		"usecase/find/find.go":               "package find\n\nfunc Execute() {}\n",
		"port/storeport/store.go":            "package storeport\n\ntype Store interface{ Save() }\n",
		"adapter/blob/blob.go":               "package blob\n\ntype Blob struct{}\n\nfunc (Blob) Save() {}\n",
		"adapter/window/route/route.go":      "package route\n\ntype Route struct{ OnCall func() }\n",
		"adapter/window/window.go":           "package window\n\nimport \"fixture/usecase/find\"\n\nfunc Show() { find.Execute() }\n",
		"adapter/index/disk/disk.go":         "package disk\n\nimport \"os\"\n\nfunc Read() ([]byte, error) { return os.ReadFile(\"x\") }\n",
		"adapter/index/file/file.go":         "package file\n\nimport \"os\"\n\nfunc Close(f *os.File) error { return f.Close() }\n",
		"adapter/index/hook/hook.go":         "package hook\n\ntype Hook struct{ OnChange func() }\n",
		"adapter/index/typeonly/typeonly.go": "package typeonly\n\nimport \"net/http\"\n\nfunc New() http.Handler { return nil }\n\nvar _ = http.HandlerFunc(nil)\n",
		"adapter/index/convert/convert.go":   "package convert\n\nimport \"os\"\n\ntype Mode = os.FileMode\n\nfunc Wrap(m uint32) Mode { return os.FileMode(m) }\n",
		"internal/adapter/host/host.go":      "package host\n\nimport \"fixture/usecase/find\"\n\nfunc Serve() { find.Execute() }\n",
	})
	found, driven := getAdapterTechnologyFindings(load.LoadFixture(t, dir), "fixture", map[string]string{})
	want := []string{
		"adapter/blob is a driven adapter that calls no technology: it holds callbacks or names types, it integrates nothing",
		"adapter/index/convert is a driven adapter that calls no technology: it holds callbacks or names types, it integrates nothing",
		"adapter/index/hook is a driven adapter that calls no technology: it holds callbacks or names types, it integrates nothing",
		"adapter/index/typeonly is a driven adapter that calls no technology: it holds callbacks or names types, it integrates nothing",
		"internal/adapter/host calls a use case: a driving adapter lives under adapter/",
	}
	if !slices.Equal(found, want) {
		t.Errorf("the rule rejects %v, want %v", found, want)
	}
	if driven != 6 {
		t.Errorf("%d driven adapters read, want 6", driven)
	}
}

func TestDrivenAdaptersCallATechnology(t *testing.T) {
	if testing.Short() {
		t.Skip("loads and type-checks the modules of the workspace")
	}
	checkDrivenAdaptersCallATechnology(t, load.Workspace(t, "libs/core/kernel"))
}

func TestMain(m *testing.M) {
	os.Exit(load.Main(m))
}

// The libraries that frame the wire are no technology a call reaches outside the process through.
func TestTechnologyCallRule_ExcludesTheWireFraming(t *testing.T) {
	for path, want := range map[string]bool{
		"connectrpc.com/connect": false, "google.golang.org/protobuf/proto": false,
		"os": true, "net/http": true, "modernc.org/sqlite": true, "time": false,
	} {
		if got := isTechnologyCall(path); got != want {
			t.Errorf("a call into %s reaches outside the process: %v, want %v", path, got, want)
		}
	}
}

// A call through a generated Connect client is a call across a process boundary: the constructor and the methods
// of a client declared in a *connect.go file count, a call of the connect.Client type counts, and naming a client
// type, building a request or calling the server side of the generated code does not.
func TestTechnologyCallRule_CountsGeneratedConnectClientCalls(t *testing.T) {
	dir := load.WriteTreeFixture(t, map[string]string{
		"gen/svcconnect/svc.connect.go": "package svcconnect\n\ntype SvcClient interface{ Ping() }\n\ntype svcClient struct{}\n\n" +
			"func (*svcClient) Ping() {}\n\nfunc NewSvcClient() SvcClient { return &svcClient{} }\n\n" +
			"type SvcHandler struct{}\n\nfunc (SvcHandler) Serve() {}\n\nfunc NewSvcHandler() SvcHandler { return SvcHandler{} }\n",
		"adapter/index/calls/calls.go":   "package calls\n\nimport \"fixture/gen/svcconnect\"\n\nfunc Ping() { svcconnect.NewSvcClient().Ping() }\n",
		"adapter/index/method/method.go": "package method\n\nimport \"fixture/gen/svcconnect\"\n\nfunc Ping(c svcconnect.SvcClient) { c.Ping() }\n",
		"adapter/index/named/named.go":   "package named\n\nimport \"fixture/gen/svcconnect\"\n\ntype Named struct{ Client svcconnect.SvcClient }\n",
		"adapter/index/server/server.go": "package server\n\nimport \"fixture/gen/svcconnect\"\n\nfunc Serve() { svcconnect.NewSvcHandler().Serve() }\n",
		"adapter/index/ctor/ctor.go":     "package ctor\n\nimport \"fixture/gen/svcconnect\"\n\nvar _ = svcconnect.NewSvcClient\n",
	})
	found, _ := getAdapterTechnologyFindings(load.LoadFixture(t, dir), "fixture", map[string]string{})
	want := []string{
		"adapter/index/ctor is a driven adapter that calls no technology: it holds callbacks or names types, it integrates nothing",
		"adapter/index/named is a driven adapter that calls no technology: it holds callbacks or names types, it integrates nothing",
		"adapter/index/server is a driven adapter that calls no technology: it holds callbacks or names types, it integrates nothing",
	}
	if !slices.Equal(found, want) {
		t.Errorf("the rule rejects %v, want %v", found, want)
	}
}

func TestConnectClientCallRule_ReadsTheClientTypeOfTheLibrary(t *testing.T) {
	library := types.NewPackage("connectrpc.com/connect", "connect")
	fset := token.NewFileSet()
	method := func(receiver string) *types.Func {
		recv := types.NewVar(token.NoPos, library, "c", types.NewPointer(types.NewNamed(types.NewTypeName(token.NoPos, library, receiver, nil), types.NewStruct(nil, nil), nil)))
		return types.NewFunc(token.NoPos, library, "CallUnary", types.NewSignatureType(recv, nil, nil, nil, nil, false))
	}
	function := types.NewFunc(token.NoPos, library, "NewRequest", types.NewSignatureType(nil, nil, nil, nil, nil, false))
	for name, tc := range map[string]struct {
		fn   *types.Func
		want bool
	}{"a method of Client": {method("Client"), true}, "a method of another type": {method("Request"), false}, "a function": {function, false}} {
		if got := isConnectClientCall(fset, tc.fn); got != tc.want {
			t.Errorf("%s is a client call: %v, want %v", name, got, tc.want)
		}
	}
}

// A package nested in an adapter family that a sibling of the family imports and that implements no port holds what
// the family shares, and calls no technology of its own. One that implements a port, one nobody in its family
// imports, and one imported from another family are adapters and still have to call a technology.
func TestTechnologyRule_LetsAFamilyShareItsPlumbing(t *testing.T) {
	dir := load.WriteTreeFixture(t, map[string]string{
		"port/storeport/store.go":                  "package storeport\n\ntype Store interface{ Save() }\n",
		"internal/adapter/plug/lease/lease.go":     "package lease\n\ntype Lease struct{ Release func() }\n",
		"internal/adapter/plug/store/store.go":     "package store\n\ntype Store struct{}\n\nfunc (Store) Save() {}\n",
		"internal/adapter/plug/user/user.go":       "package user\n\nimport (\n\t\"os\"\n\n\t\"fixture/internal/adapter/plug/lease\"\n\t\"fixture/internal/adapter/plug/store\"\n)\n\nfunc Use(l lease.Lease, s store.Store) error { _ = l; s.Save(); _, err := os.ReadFile(\"x\"); return err }\n",
		"internal/adapter/plug/lonely/lonely.go":   "package lonely\n\ntype Lonely struct{ Release func() }\n",
		"internal/adapter/other/hollow/hollow.go":  "package hollow\n\ntype Hollow struct{ Release func() }\n",
		"internal/adapter/other/user/user.go":      "package user\n\nimport (\n\t\"os\"\n\n\t\"fixture/internal/adapter/other/hollow\"\n)\n\nfunc Use(h hollow.Hollow) error { _ = h; _, err := os.ReadFile(\"x\"); return err }\n",
		"internal/adapter/plug/foreign/foreign.go": "package foreign\n\nimport \"fixture/internal/adapter/other/hollow\"\n\nfunc Use(h hollow.Hollow) error { _ = h; return nil }\n",
	})
	found, _ := getAdapterTechnologyFindings(load.LoadFixture(t, dir), "fixture", map[string]string{})
	want := []string{
		"internal/adapter/plug/foreign is a driven adapter that calls no technology: it holds callbacks or names types, it integrates nothing",
		"internal/adapter/plug/lonely is a driven adapter that calls no technology: it holds callbacks or names types, it integrates nothing",
		"internal/adapter/plug/store is a driven adapter that calls no technology: it holds callbacks or names types, it integrates nothing",
	}
	if !slices.Equal(found, want) {
		t.Errorf("the rule rejects %v, want %v", found, want)
	}
}
