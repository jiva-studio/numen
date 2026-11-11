package main

import (
	"go/ast"
	"strings"
	"unicode"
)

// restateThresholdPercent is the share of the words of a doc comment, stop words left out, that the name and the
// signature of the declaration already hold, from which the comment is refused.
const restateThresholdPercent = 80

var stopWords = toSet(
	"a an the of to for in is are was were be " +
		"been by that which and or with from on as it its " +
		"this these those at if when then into each one any does " +
		"do has have whose whether so not than there their can will " +
		"may",
)

// synonyms maps verbs that mean the same operation in a name and in a sentence to one word, so that
// `GetPlugin finds no plugin` and `SetVault stores the vault` are read as what they say.
var synonyms = map[string]string{
	"find": "get", "read": "get", "fetch": "get", "return": "get", "load": "get", "retrieve": "get", "obtain": "get", "lookup": "get",
	"put": "set", "store": "set", "save": "set", "write": "set", "update": "set", "assign": "set",
	"make": "create", "build": "create", "new": "create", "construct": "create",
	"remove": "delete", "drop": "delete",
}

func toSet(words string) map[string]bool {
	set := map[string]bool{}
	for _, word := range strings.Fields(words) {
		set[word] = true
	}
	return set
}

// splitWords cuts text into lower-case words at camelCase humps, digits and every character that is not a letter.
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

// stem reduces a word to the form a name and a sentence share: no plural or third-person ending, and one word for the
// verbs that mean the same operation.
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

// isSameWord reports whether a word of a comment is a word of a name: equal, or one is the start of the other
// (`window` and the package name `windowsession`) when both have four letters or more.
func isSameWord(commentWord, nameWord string) bool {
	if commentWord == nameWord {
		return true
	}
	if len(commentWord) < 4 || len(nameWord) < 4 {
		return false
	}
	return strings.HasPrefix(nameWord, commentWord) || strings.HasPrefix(commentWord, nameWord)
}

// restatement counts the words of a comment, stop words left out, and how many of them the names already hold.
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

// isRestating reports whether a comment says nothing the names do not.
func isRestating(comment string, names []string) (matched, total int, ok bool) {
	matched, total = restatement(comment, names)
	return matched, total, total > 0 && matched*100 >= restateThresholdPercent*total
}

// listTypeNames lists the identifiers a type expression names, such as `domain.Vault` or `[]manifest.Discovered`.
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

// listSignatureNames lists the names a function declaration shows: its own, its receiver type, and the names and the
// types of its parameters and results.
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

// stripComment returns the text of a comment group without the comment markers.
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
