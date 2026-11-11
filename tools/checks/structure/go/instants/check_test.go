package instants

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"testing"
	"github.com/jiva-studio/numen/tools/checks/structure/go/load"
)

// TestCoreInstantsUseTimeType checks that an instant in domain/ and port/ is a time.Time.
// A second count and a millisecond count are both integers, so using one where
// the other is expected compiles and is off by a factor of a thousand; the zero
// of either is a valid instant in 1970.
func TestCoreInstantsUseTimeType(t *testing.T) {
	var wrong []string
	var read int
	for _, item := range []string{"domain", "port"} {
		dir := load.GetPackageDir(item)
		for _, file := range load.ParseSourceFiles(t, dir) {
			for _, found := range getInstants(file) {
				read++
				if found.Spelled != "time.Time" {
					wrong = append(wrong, fmt.Sprintf("%s/%s.%s is a %s",
						item, found.In, found.Field, found.Spelled))
				}
			}
		}
	}
	for _, one := range wrong {
		t.Error(one + ": an instant the core carries is a time.Time")
	}

	// A walk that read no instant is a rule checked against nothing, and it
	// passes.
	if read == 0 {
		t.Fatal("domain/ and port/ name no instant: the walk is not reading them")
	}
}

// finds the field and the type it is written in, and lets through the one that
// is a time and the ones that name no instant at all.
func TestInstantRule_RejectsTimeFields(t *testing.T) {
	file, err := parser.ParseFile(token.NewFileSet(), "instant.go", `package p

import "time"

type Reading struct {
	ModTime  time.Time
	SeenTime int64
	Pages    int
	runTime  int64
}
`, 0)
	if err != nil {
		t.Fatal(err)
	}
	var rejected []string
	for _, one := range getInstants(file) {
		if one.Spelled != "time.Time" {
			rejected = append(rejected, one.Field)
		}
	}
	if !slices.Equal(rejected, []string{"SeenTime"}) {
		t.Errorf("rejected %v, want SeenTime", rejected)
	}
}
