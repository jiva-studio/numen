package notes_test

import (
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"testing"
)

// module is what the packages of this module are named under.
const module = "github.com/jiva-studio/numen/modules/libs/notes"

// thirdParty is the code outside the standard library these packages may name.
var thirdParty = []string{"gopkg.in/yaml.v3", "golang.org/x/text/"}

// The packages here reach only each other, the standard library and the few third-party packages
// listed: nothing they answer waits on a disk, a database or a clock.
func TestNoPackageHereReachesOutsideTheVocabulary(t *testing.T) {
	var read int
	err := filepath.WalkDir(".", func(path string, entry fs.DirEntry, err error) error {
		if err == nil && entry.IsDir() && entry.Name() == "gorules" {
			return filepath.SkipDir
		}
		if err != nil || entry.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return err
		}
		file, err := parser.ParseFile(token.NewFileSet(), path, nil, parser.ImportsOnly)
		if err != nil {
			return err
		}
		read++
		for _, one := range file.Imports {
			to, err := strconv.Unquote(one.Path.Value)
			if err != nil {
				return err
			}
			if strings.HasPrefix(to, module) || !strings.Contains(strings.Split(to, "/")[0], ".") ||
				slices.ContainsFunc(thirdParty, func(allowed string) bool { return to == allowed || strings.HasPrefix(to, allowed) }) {
				continue
			}
			t.Errorf("%s names %s", path, to)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if read < 20 {
		t.Fatalf("%d files read: the walk is not reading the packages", read)
	}
	if _, err := os.Stat("domain"); err != nil {
		t.Fatal(err)
	}
}
