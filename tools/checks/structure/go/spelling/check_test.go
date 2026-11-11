// Package spelling is the check `go/spelling`, run on the tree with the planted violations that prove it.
package spelling

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/jiva-studio/numen/tools/checks/structure/go/load"
)

// No identifier, comment or string literal of the Go code is spelled in British English. The word list in
// british.txt names the words; a quoted name that is not ours to respell goes in allowed.txt with its reason.
// Generated files follow the schema they are generated from.
func TestNoGoTextIsSpeltInBritishEnglish(t *testing.T) {
	dict, err := parseDictionary(britishList)
	if err != nil {
		t.Fatal(err)
	}
	allowed, err := parseAllowances(allowedList)
	if err != nil {
		t.Fatal(err)
	}
	root, err := filepath.Abs(load.RepoRootDir)
	if err != nil {
		t.Fatal(err)
	}
	found, err := findBritishSpelling(root, load.ListWorkspaceModules(t), dict, allowed)
	if err != nil {
		t.Fatal(err)
	}
	for _, one := range found {
		t.Errorf("%s: %s", one.Position, one.Name)
	}
	for _, one := range allowed {
		if !one.used {
			t.Errorf("the allow-list entry %s %s lets nothing through", one.path, one.word)
		}
	}
}

// TestSpellingWalk_FindsThePlantedFile verifies that the walk finds a British word in a Go file of a
// module, skips testdata and hidden folders, and fails on modules without a Go file.
func TestSpellingWalk_FindsThePlantedFile(t *testing.T) {
	root := load.WriteTreeFixture(t, map[string]string{
		"m/a/a.go":        plant("package a\n\n// The request was can·celled.\nvar _ = 1\n"),
		"m/testdata/b.go": plant("package b\n\n// can·celled\n"),
		"m/.hidden/c.go":  plant("package c\n\n// can·celled\n"),
	})
	found, err := findBritishSpelling(root, []string{filepath.Join(root, "m")}, readDictionary(t), nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(found) != 1 || !strings.HasPrefix(found[0].Position, "m/a/a.go:") {
		t.Errorf("the walk found %v, want one British word in m/a/a.go", found)
	}
	empty := t.TempDir()
	if _, err := findBritishSpelling(empty, []string{empty}, readDictionary(t), nil); err == nil {
		t.Error("modules without a Go file passed")
	}
}

func findBritishSpelling(root string, modules []string, dict dictionary, allowed []*allowance) ([]load.Finding, error) {
	var found []load.Finding
	visited := map[string]bool{}
	for _, module := range modules {
		err := filepath.WalkDir(module, func(path string, entry fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if entry.IsDir() {
				if name := entry.Name(); name == "node_modules" || name == "testdata" || strings.HasPrefix(name, ".") && path != module {
					return filepath.SkipDir
				}
				return nil
			}
			if !strings.HasSuffix(path, ".go") || visited[path] {
				return nil
			}
			visited[path] = true
			rel, err := filepath.Rel(root, path)
			if err != nil {
				return err
			}
			one, err := listHits(filepath.ToSlash(rel), path, nil, dict, allowed)
			found = append(found, one...)
			return err
		})
		if err != nil {
			return nil, err
		}
	}
	if len(visited) == 0 {
		return nil, fmt.Errorf("no Go file read under %v: the walk checks nothing", modules)
	}
	return load.SortFindings(found), nil
}

func readDictionary(t *testing.T) dictionary {
	t.Helper()
	dict, err := parseDictionary(britishList)
	if err != nil {
		t.Fatal(err)
	}
	return dict
}

func plant(text string) string {
	return strings.ReplaceAll(text, "·", "")
}

// The word list finds the British spelling in whole words, in stems with their endings and inside names, and
// passes the American spelling and the words that only look British.
func TestSpellingRule_FindsBritishWords(t *testing.T) {
	dict := readDictionary(t)
	for text, want := range map[string]string{
		"isCan·celled": "canceled", "can·celling": "canceling", "Organ·isation": "organization", "organ·isational": "organizational",
		"ana·lyse": "analyze", "lic·ence": "license", "catal·ogue": "catalog", "gr·ey": "gray", "col·our": "color",
		"Behav·iours": "behaviors", "neigh·bour": "neighbor", "cen·tre": "center", "fav·ourite": "favorite", "hon·oured": "honored",
		"lab·our": "labor", "trav·elled": "traveled", "model·ling": "modeling", "ful·fil": "fulfill", "def·ence": "defense",
		"arte·fact": "artifact", "judge·ment": "judgment", "real·ise": "realize", "serial·isation": "serialization",
		"emphas·ised": "emphasized", "minim·ising": "minimizing", "col·ourful": "colorful", "cen·tred": "centered",
	} {
		var got string
		for _, word := range splitWords(plant(text)) {
			if american, ok := dict.find(word); ok {
				got = american
			}
		}
		if got != want {
			t.Errorf("%s is read as %q, want %q", plant(text), got, want)
		}
	}
	for _, text := range []string{
		"canceled", "canceling", "organization", "analysis", "analyses", "license", "licensed", "catalog", "gray", "color", "behavior",
		"center", "favorite", "otherwise", "exercise", "promise", "premise", "advise", "surprise", "compromise", "precise",
		"emphasis", "organism", "optimism", "realist", "capitalism", "polaris", "traveler", "fulfilled", "defender", "colonel",
		"licenseKey", "HTTPServer", "mechanism", "criticism", "specialist", "normalize", "cancelation",
	} {
		for _, word := range splitWords(text) {
			if american, ok := dict.find(word); ok {
				t.Errorf("%s is read as the British spelling of %s", word, american)
			}
		}
	}
}

// The check reads identifiers, comments and string literals, reports the line of each, skips the path of an
// import and a generated file, and lets through a word that the allow-list names for the file.
func TestSpellingRule_ReadsIdentifiersCommentsAndStrings(t *testing.T) {
	dict := readDictionary(t)
	src := plant(strings.Join([]string{
		"package a",
		"",
		`import col·ourLib "example.com/col·our"`,
		"",
		"// Ana·lyse the input.",
		"func isCan·celled() string {",
		"\t/* the",
		"\tgr·ey zone */",
		"\treturn \"fav·ourite\" + `",
		"lic·ence`",
		"}",
		"",
		"var _ = col·ourLib.X",
		"",
	}, "\n"))
	found, err := listHits("a.go", "a.go", src, dict, nil)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{
		plant("a.go:3 col·our is British, write color"),
		plant("a.go:5 ana·lyse is British, write analyze"),
		plant("a.go:6 can·celled is British, write canceled"),
		plant("a.go:8 gr·ey is British, write gray"),
		plant("a.go:9 fav·ourite is British, write favorite"),
		plant("a.go:10 lic·ence is British, write license"),
		plant("a.go:13 col·our is British, write color"),
	}
	got := load.FormatFindings(found)
	slices.Sort(want)
	if !slices.Equal(got, want) {
		t.Errorf("the rule reports\n%s\nwant\n%s", strings.Join(got, "\n"), strings.Join(want, "\n"))
	}

	allowed, err := parseAllowances(plant("a.go col·our the package is named by its author\na.go lic·ence a quoted name of the upstream format\na.go neigh·bour never appears in this file\n"))
	if err != nil {
		t.Fatal(err)
	}
	found, err = listHits("a.go", "a.go", src, dict, allowed)
	if err != nil {
		t.Fatal(err)
	}
	if got := load.FormatFindings(found); len(got) != 4 {
		t.Errorf("the allow-list leaves %v, want the 4 other words", got)
	}
	if !allowed[0].used || !allowed[1].used || allowed[2].used {
		t.Errorf("the entries used are %v %v %v, want true true false", allowed[0].used, allowed[1].used, allowed[2].used)
	}

	generated, err := listHits("b.go", "b.go", plant("// Code generated by buf. DO NOT EDIT.\n\npackage b\n\nvar col·our int\n"), dict, nil)
	if err != nil || len(generated) != 0 {
		t.Errorf("a generated file reports %v, %v, want nothing", generated, err)
	}
}

func TestSpellingRule_RejectsAnEntryWithoutAReason(t *testing.T) {
	if _, err := parseAllowances(plant("a.go col·our\n")); err == nil {
		t.Error("an allow-list entry without a reason is accepted")
	}
	if _, err := parseDictionary(plant("col·our\n")); err == nil {
		t.Error("a word list line without an American word is accepted")
	}
	if _, err := parseDictionary("color color\n"); err == nil {
		t.Error("a word list line that spells a word the same in both is accepted")
	}
}

func TestMain(m *testing.M) {
	os.Exit(load.Main(m))
}
