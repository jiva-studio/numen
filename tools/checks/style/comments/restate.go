package main

import (
	_ "embed"
	"encoding/json"
	"go/ast"
	"strings"
	"unicode"
)

//go:embed restate.json
var restateJSON []byte

// restateRules is restate.json, the one definition of the threshold, the stop words and the synonyms that
// check.mjs reads as well.
type restateRules struct {
	ThresholdPercent int                 `json:"thresholdPercent"`
	MinPrefixLength  int                 `json:"minPrefixLength"`
	StopWords        []string            `json:"stopWords"`
	Synonyms         map[string][]string `json:"synonyms"`
}

var (
	rules                   = readRestateRules()
	restateThresholdPercent = rules.ThresholdPercent
	minPrefixLength         = rules.MinPrefixLength
	stopWords               = toSet(rules.StopWords)
	synonyms                = invertSynonyms(rules.Synonyms)
)

func readRestateRules() restateRules {
	var read restateRules
	if err := json.Unmarshal(restateJSON, &read); err != nil {
		panic("restate.json: " + err.Error())
	}
	return read
}

func invertSynonyms(groups map[string][]string) map[string]string {
	inverted := map[string]string{}
	for canonical, verbs := range groups {
		for _, verb := range verbs {
			inverted[verb] = canonical
		}
	}
	return inverted
}

func toSet(words []string) map[string]bool {
	set := map[string]bool{}
	for _, word := range words {
		set[word] = true
	}
	return set
}

func splitWords(text string) []string {
	var words []string
	var current []rune
	flush := func() {
		if len(current) > 0 {
			words = append(words, strings.ToLower(string(current)))
			current = nil
		}
	}
	runes := []rune(text)
	for i, r := range runes {
		if !unicode.IsLetter(r) && !unicode.IsDigit(r) {
			flush()
			continue
		}
		if unicode.IsUpper(r) && len(current) > 0 {
			previousLower := unicode.IsLower(runes[i-1]) || unicode.IsDigit(runes[i-1])
			acronymEnds := unicode.IsUpper(runes[i-1]) && i+1 < len(runes) && unicode.IsLower(runes[i+1])
			if previousLower || acronymEnds {
				flush()
			}
		}
		current = append(current, r)
	}
	flush()
	return words
}

func stem(word string) string {
	switch {
	case strings.HasSuffix(word, "ies") && len(word) > 4:
		word = word[:len(word)-3] + "y"
	case len(word) > 4 && (strings.HasSuffix(word, "sses") || strings.HasSuffix(word, "xes") || strings.HasSuffix(word, "ches") || strings.HasSuffix(word, "shes")):
		word = word[:len(word)-2]
	case len(word) > 3 && strings.HasSuffix(word, "s") && !strings.HasSuffix(word, "ss") && !strings.HasSuffix(word, "us"):
		word = word[:len(word)-1]
	}
	if canonical, ok := synonyms[word]; ok {
		return canonical
	}
	return word
}

func isSameWord(commentWord, nameWord string) bool {
	if commentWord == nameWord {
		return true
	}
	if len(commentWord) < minPrefixLength || len(nameWord) < minPrefixLength {
		return false
	}
	return strings.HasPrefix(nameWord, commentWord) || strings.HasPrefix(commentWord, nameWord)
}

func restatement(comment string, names []string) (matched, total int) {
	nameWords := map[string]bool{}
	for _, name := range names {
		for _, word := range splitWords(name) {
			nameWords[stem(word)] = true
		}
	}
	for _, word := range splitWords(comment) {
		if stopWords[word] {
			continue
		}
		total++
		stemmed := stem(word)
		for name := range nameWords {
			if isSameWord(stemmed, name) {
				matched++
				break
			}
		}
	}
	return matched, total
}

func isRestating(comment string, names []string) (matched, total int, ok bool) {
	matched, total = restatement(comment, names)
	return matched, total, total > 0 && matched*100 >= restateThresholdPercent*total
}

func listTypeNames(expr ast.Expr) []string {
	var names []string
	if expr == nil {
		return nil
	}
	ast.Inspect(expr, func(n ast.Node) bool {
		if ident, ok := n.(*ast.Ident); ok {
			names = append(names, ident.Name)
		}
		return true
	})
	return names
}

func listSignatureNames(pkg string, decl *ast.FuncDecl) []string {
	names := []string{pkg, decl.Name.Name}
	add := func(fields *ast.FieldList) {
		if fields == nil {
			return
		}
		for _, field := range fields.List {
			for _, name := range field.Names {
				names = append(names, name.Name)
			}
			names = append(names, listTypeNames(field.Type)...)
		}
	}
	add(decl.Recv)
	add(decl.Type.Params)
	add(decl.Type.Results)
	return names
}

func stripComment(group *ast.CommentGroup) string {
	var lines []string
	for _, c := range group.List {
		text := c.Text
		switch {
		case strings.HasPrefix(text, "//"):
			text = strings.TrimPrefix(text, "//")
		case strings.HasPrefix(text, "/*"):
			text = strings.TrimSuffix(strings.TrimPrefix(text, "/*"), "*/")
		}
		lines = append(lines, text)
	}
	return strings.Join(lines, "\n")
}
