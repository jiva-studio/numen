package main

import (
	"reflect"
	"testing"
)

func TestReadSourceCountsTestAndFuzzFunctions(t *testing.T) {
	src := `package notes

import "testing"

func TestSave(t *testing.T) {}

func TestMain(m *testing.M) {}

func FuzzParse(f *testing.F) {}

func FuzzMismatched(t *testing.T) {}

func Testlower(t *testing.T) {}

func testHelper(t *testing.T) {}

func (s suite) TestMethod(t *testing.T) {}
`
	if got := readSource("notes_test.go", []byte(src)).Tests; got != 2 {
		t.Fatalf("tests = %d, want 2", got)
	}
}

func TestReadSourceCountsTestsOnlyInTestFiles(t *testing.T) {
	src := "package notes\n\nimport \"testing\"\n\nfunc TestLike(t *testing.T) {}\n"
	if got := readSource("notes.go", []byte(src)).Tests; got != 0 {
		t.Fatalf("tests = %d, want 0", got)
	}
}

func TestReadSourceSkipsCommentedOutAndQuotedCode(t *testing.T) {
	src := "package notes\n\nimport (\n\t\"net/http\"\n\t\"testing\"\n)\n\n" +
		"func TestSave(t *testing.T) {}\n\n" +
		"/*\nfunc TestRemoved(t *testing.T) {}\nvar _, _ = http.Get(\"https://removed.numen.md\")\n*/\n\n" +
		"// func TestDisabled(t *testing.T) { http.Get(\"https://disabled.numen.md\") }\n\n" +
		"const sample = `\nfunc TestQuoted(t *testing.T) {}\nhttp.Get(x)\n`\n"
	facts := readSource("notes_test.go", []byte(src))
	if facts.Tests != 1 {
		t.Errorf("tests = %d, want 1", facts.Tests)
	}
	if len(facts.Calls) != 0 || len(facts.URLs) != 0 {
		t.Errorf("calls = %v, urls = %v, want none", facts.Calls, facts.URLs)
	}
}

func TestReadSourceResolvesAnAliasedImport(t *testing.T) {
	src := `package feed

import (
	h "net/http"
	tst "testing"
)

func read() { _, _ = h.Get("https://dl.numen.md/feed") }

func TestRead(t *tst.T) {}
`
	facts := readSource("feed_test.go", []byte(src))
	want := []call{{Name: "http.Get", Target: "https://dl.numen.md/feed"}}
	if !reflect.DeepEqual(facts.Calls, want) {
		t.Errorf("calls = %v, want %v", facts.Calls, want)
	}
	if facts.Tests != 1 {
		t.Errorf("tests = %d, want 1", facts.Tests)
	}
}

func TestReadSourceResolvesADotImport(t *testing.T) {
	src := `package feed

import (
	. "net/http"
	. "testing"
)

var client = &Client{}

func read() { _, _ = Get("https://dl.numen.md/feed") }

func TestRead(t *T) {}
`
	facts := readSource("feed_test.go", []byte(src))
	want := []call{{Name: "http.Client"}, {Name: "http.Get", Target: "https://dl.numen.md/feed"}}
	if !reflect.DeepEqual(facts.Calls, want) {
		t.Errorf("calls = %v, want %v", facts.Calls, want)
	}
	if facts.Tests != 1 {
		t.Errorf("tests = %d, want 1", facts.Tests)
	}
}

func TestReadSourceIgnoresAPackageOfAnotherPathWithTheSameName(t *testing.T) {
	src := `package feed

import "example.com/fake/http"

func read() { _, _ = http.Get("https://dl.numen.md/feed") }
`
	if facts := readSource("feed.go", []byte(src)); len(facts.Calls) != 0 {
		t.Fatalf("calls = %v, want none", facts.Calls)
	}
}

func TestReadSourceFindsEveryClientUse(t *testing.T) {
	src := `package feed

import (
	"crypto/tls"
	"net"
	"net/http"

	"github.com/gorilla/websocket"
	"google.golang.org/grpc"
)

func open(address string) {
	_ = http.DefaultClient
	_ = http.Transport{}
	_, _ = http.NewRequestWithContext(nil, "GET", address, nil)
	_, _ = net.Dial("tcp", "localhost:9000")
	_ = net.Dialer{}
	_, _ = tls.Dial("tcp", address, nil)
	_, _, _ = websocket.DefaultDialer.Dial(address, nil)
	_, _ = grpc.NewClient(address)
	_ = cache.Get(address)
}
`
	facts := readSource("feed.go", []byte(src))
	want := []call{
		{Name: "http.Client"},
		{Name: "http.Client"},
		{Name: "http.NewRequest"},
		{Name: "net.Dial", Target: "localhost:9000"},
		{Name: "net.Dial"},
		{Name: "tls.Dial"},
		{Name: "websocket.Dial"},
		{Name: "grpc.Dial"},
	}
	if !reflect.DeepEqual(facts.Calls, want) {
		t.Errorf("calls = %v, want %v", facts.Calls, want)
	}
}

func TestReadSourceListsURLAndAddressLiterals(t *testing.T) {
	src := "package local\n\nimport \"net/http\"\n\n" +
		"const base = \"http://127.0.0.1:11434\"\n" +
		"const socket = `wss://live.numen.md`\n" +
		"const listen = \"[::1]:0\"\n" +
		"const clock = \"15:04\"\n" +
		"const text = \"see https://numen.md\"\n\n" +
		"func read() { _, _ = http.Get(base + \"/api\") }\n"
	facts := readSource("local.go", []byte(src))
	if want := []string{"http://127.0.0.1:11434", "wss://live.numen.md"}; !reflect.DeepEqual(facts.URLs, want) {
		t.Errorf("urls = %v, want %v", facts.URLs, want)
	}
	if want := []string{"[::1]:0", "15:04"}; !reflect.DeepEqual(facts.Addresses, want) {
		t.Errorf("addresses = %v, want %v", facts.Addresses, want)
	}
}

func TestReadSourceReadsAFileWithSyntaxErrorsAsFarAsItParses(t *testing.T) {
	src := "package feed\n\nimport \"net/http\"\n\nfunc read() { _, _ = http.Get(\"https://dl.numen.md\") }\n\nfunc broken( {\n"
	facts := readSource("feed.go", []byte(src))
	if want := []call{{Name: "http.Get", Target: "https://dl.numen.md"}}; !reflect.DeepEqual(facts.Calls, want) {
		t.Fatalf("calls = %v, want %v", facts.Calls, want)
	}
}
