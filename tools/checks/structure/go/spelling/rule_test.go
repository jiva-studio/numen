package spelling

import (
	_ "embed"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"slices"
	"strings"
	"unicode"

	"github.com/jiva-studio/numen/tools/checks/structure/go/load"
)

//go:embed british.txt
var britishList string

//go:embed allowed.txt
var allowedList string

func listFamilyEndings() []string {
	return []string{
		"s", "e", "es", "ed", "d", "ing", "er", "ers", "ation", "ations", "ational", "able", "ability", "ment", "ments",
		"ful", "less", "ist", "ists", "y", "ies", "ive", "ively", "ly", "ness", "ite", "ites", "al", "ally", "hood", "hoods",
	}
}

type family struct {
	british, american string
	isBare            bool
}

// dictionary maps British words to the American spelling: whole words, and stems that take an ending.
type dictionary struct {
	words    map[string]string
	families []family
}

func parseDictionary(text string) (dictionary, error) {
	dict := dictionary{words: map[string]string{}}
	for number, line := range strings.Split(text, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) != 2 {
			return dict, fmt.Errorf("line %d of the word list is %q, want a British and an American word", number+1, line)
		}
		british, american := fields[0], fields[1]
		switch {
		case strings.HasSuffix(british, "*") || strings.HasSuffix(british, "~"):
			if british[len(british)-1] != american[len(american)-1] {
				return dict, fmt.Errorf("line %d of the word list marks only one of its stems", number+1)
			}
			dict.families = append(dict.families, family{
				british: strings.TrimRight(british, "*~"), american: strings.TrimRight(american, "*~"), isBare: strings.HasSuffix(british, "*"),
			})
		case british == american:
			return dict, fmt.Errorf("line %d of the word list spells %q the same in both", number+1, british)
		default:
			dict.words[british] = american
		}
	}
	return dict, nil
}

func (d dictionary) find(word string) (string, bool) {
	if american, ok := d.words[word]; ok {
		return american, true
	}
	for _, one := range d.families {
		if !strings.HasPrefix(word, one.british) {
			continue
		}
		ending := word[len(one.british):]
		if ending == "" && one.isBare || slices.Contains(listFamilyEndings(), ending) {
			return one.american + ending, true
		}
	}
	return "", false
}

func splitWords(text string) []string {
	var words []string
	var current []rune
	flush := func() {
		if len(current) > 0 {
			words = append(words, strings.ToLower(string(current)))
			current = current[:0]
		}
	}
	runes := []rune(text)
	for i, r := range runes {
		switch {
		case !unicode.IsLetter(r):
			flush()
			continue
		case unicode.IsUpper(r) && len(current) > 0 && (unicode.IsLower(runes[i-1]) ||
			i+1 < len(runes) && unicode.IsLower(runes[i+1]) && unicode.IsUpper(runes[i-1])):
			flush()
		}
		current = append(current, r)
	}
	flush()
	return words
}

// spellingHit is a British word found in a file, with the line and the American spelling.
type spellingHit struct {
	line              int
	british, american string
}

func (h spellingHit) String() string {
	return fmt.Sprintf("%d %s %s", h.line, h.british, h.american)
}

func listBritishWords(path string, src any, dict dictionary) ([]spellingHit, error) {
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, path, src, parser.ParseComments|parser.SkipObjectResolution)
	if err != nil || ast.IsGenerated(file) {
		return nil, err
	}
	var found []spellingHit
	scan := func(pos token.Pos, text string) {
		for offset, line := range strings.Split(text, "\n") {
			for _, word := range splitWords(line) {
				if american, ok := dict.find(word); ok {
					found = append(found, spellingHit{line: fset.Position(pos).Line + offset, british: word, american: american})
				}
			}
		}
	}
	for _, group := range file.Comments {
		for _, one := range group.List {
			scan(one.Pos(), one.Text)
		}
	}
	ast.Inspect(file, func(node ast.Node) bool {
		switch item := node.(type) {
		case *ast.ImportSpec:
			if item.Name != nil {
				scan(item.Name.Pos(), item.Name.Name)
			}
			return false
		case *ast.Ident:
			scan(item.Pos(), item.Name)
		case *ast.BasicLit:
			if item.Kind == token.STRING {
				scan(item.Pos(), item.Value)
			}
		}
		return true
	})
	slices.SortFunc(found, func(a, b spellingHit) int {
		if a.line != b.line {
			return a.line - b.line
		}
		return strings.Compare(a.british, b.british)
	})
	return slices.Compact(found), nil
}

// allowance is one quoted name the check lets through: a word in a file, with the reason it stays.
type allowance struct {
	path, word, reason string
	used               bool
}

func parseAllowances(text string) ([]*allowance, error) {
	var found []*allowance
	for number, line := range strings.Split(text, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) < 5 {
			return nil, fmt.Errorf("line %d of the allow-list is %q, want a path, a word and a reason of at least three words", number+1, line)
		}
		found = append(found, &allowance{path: fields[0], word: fields[1], reason: strings.Join(fields[2:], " ")})
	}
	return found, nil
}

func getAllowance(allowed []*allowance, path, word string) *allowance {
	for _, one := range allowed {
		if one.path == path && one.word == word {
			return one
		}
	}
	return nil
}

func listHits(rel, path string, src any, dict dictionary, allowed []*allowance) ([]load.Finding, error) {
	hits, err := listBritishWords(path, src, dict)
	var found []load.Finding
	for _, match := range hits {
		if one := getAllowance(allowed, rel, match.british); one != nil {
			one.used = true
			continue
		}
		found = append(found, load.Finding{Position: fmt.Sprintf("%s:%d", rel, match.line), Name: match.british + " is British, write " + match.american})
	}
	return found, err
}
