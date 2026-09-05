package container

import (
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"testing"
)

// nouns are the words of ours ending in -ing or -ed that are ordinary English
// nouns, and what each one means.
//
// A type is a noun or a noun phrase: a gerund says what is happening to a
// thing rather than what the thing is, and a stranger meeting one cannot say
// what it holds or whose it is. A machine reading a name sees only how it
// ends, and no suffix tells a gerund from a word that merely finishes those
// letters — so the words that are not gerunds are written down with a gloss.
// It is a dictionary and not a list of exemptions: a reader applies the rule
// without reading it, and a word nothing is named by any more has to go.
var nouns = map[string]string{
	"embed":     "the scenario that gives a vault's chunks their vectors, in the family of verb-named use cases",
	"heading":   "a line a section of a note stands under",
	"indexing":  "the settings section about making a vault searchable, beside Recognition and Transcription",
	"rating":    "how well a card was answered",
	"reading":   "what a document was read as",
	"recording": "a sound file",
	"setting":   "one setting of the file",
}

// The rule is drawn at the exported names. An unexported type is met only
// inside the package that declares it, where the file and the package say what
// it is; an exported one travels, and is what a stranger meets cold.
var nounWords = regexp.MustCompile(`[A-Z]+(?:[a-z0-9]*)`)

// gerund is whether a name reads as a gerund or a participle: it ends in those
// letters and its last word is in no dictionary of ours.
func gerund(name string) bool {
	if !strings.HasSuffix(name, "ing") && !strings.HasSuffix(name, "ed") {
		return false
	}
	said := nounWords.FindAllString(name, -1)
	if len(said) == 0 {
		return true
	}
	_, known := nouns[strings.ToLower(said[len(said)-1])]
	return !known
}

// declared are the exported types one file declares.
func declared(file *ast.File) []string {
	var found []string
	ast.Inspect(file, func(node ast.Node) bool {
		spec, is := node.(*ast.TypeSpec)
		if is && spec.Name.IsExported() {
			found = append(found, spec.Name.Name)
		}
		return true
	})
	return found
}

// everyExportedType is every exported type of the core, against the file it
// stands in.
func everyExportedType(t *testing.T) map[string]string {
	t.Helper()
	found := map[string]string{}
	err := filepath.WalkDir("..", func(path string, entry fs.DirEntry, err error) error {
		if err != nil || entry.IsDir() || !strings.HasSuffix(path, ".go") {
			return err
		}
		if strings.HasSuffix(path, "_test.go") || strings.HasSuffix(path, ".pb.go") {
			return nil
		}
		file, err := parser.ParseFile(token.NewFileSet(), path, nil, parser.SkipObjectResolution)
		if err != nil {
			return err
		}
		for _, one := range declared(file) {
			found[one] = path
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return found
}

// A type is a noun or a noun phrase. Verb phrases name functions and methods;
// nouns name the things they act on.
//
// One word came to name a flashcards pill, a notes result and a handler at
// once, and nobody could say which from the name. Plexing could not have been
// introduced without somebody writing the word into the dictionary above and
// being asked what it means.
func TestNoTypeOfTheCoreIsNamedByAGerundOrParticiple(t *testing.T) {
	found := everyExportedType(t)
	var wrong []string
	for name, path := range found {
		if gerund(name) {
			wrong = append(wrong, path+" declares "+name)
		}
	}
	slices.Sort(wrong)
	for _, one := range wrong {
		t.Error(one + ": a type is a noun or a noun phrase")
	}

	// A walk that read no type of the core is a rule checked against nothing,
	// and it passes. The count is a floor well under what the core declares.
	if len(found) < 150 {
		t.Fatalf("%d exported types read: the walk is not reading the core", len(found))
	}
}

// A word nothing is named by any more is a word nobody has to argue for, and a
// dictionary that keeps them fills up until it reads as a census. It only
// shrinks.
func TestEveryWordOfTheDictionaryNamesAType(t *testing.T) {
	said := map[string]bool{}
	for name := range everyExportedType(t) {
		if !strings.HasSuffix(name, "ing") && !strings.HasSuffix(name, "ed") {
			continue
		}
		if words := nounWords.FindAllString(name, -1); len(words) > 0 {
			said[strings.ToLower(words[len(words)-1])] = true
		}
	}
	var gone []string
	for word := range nouns {
		if !said[word] {
			gone = append(gone, word)
		}
	}
	slices.Sort(gone)
	for _, one := range gone {
		t.Error(one + " is in the dictionary and names no type: the dictionary only shrinks")
	}
}

// What the rule refuses, read against names written to be refused. The words
// it lets through are the dictionary's, and a name not ending in those letters
// is never this rule's business however it reads.
//
// Drawn is the participle the rule names and this reading cannot see: an
// irregular one has no ending to test for, and a machine that tried would be
// reading English rather than a suffix. A person catches those.
func TestWhatTheNounRuleRefuses(t *testing.T) {
	cases := []struct {
		says    string
		name    string
		allowed bool
	}{
		{"a gerund", "Plexing", false},
		{"a participle", "Configured", false},
		{"a gerund with a qualifier in front", "PlexFiling", false},
		{"a clause read as a name", "FolderMissing", false},
		{"a noun in the dictionary", "Heading", true},
		{"a compound of one", "OpenRecording", true},
		{"a plain noun", "Vault", true},
		{"a noun whose stem is a verb", "Editor", true},
		{"an irregular participle, which has no ending", "Drawn", true},
	}

	var refused, wanted []string
	for _, one := range cases {
		if gerund(one.name) {
			refused = append(refused, one.says)
		}
		if !one.allowed {
			wanted = append(wanted, one.says)
		}
	}
	if !slices.Equal(refused, wanted) {
		t.Errorf("the rule refuses %v, want %v", refused, wanted)
	}
}
