package searchquery

import (
	"crypto/sha256"
	"encoding/hex"
	"slices"
	"strconv"
	"strings"
)

// GetTypeName returns the name a signature writes for a type, such as list(string).
func GetTypeName(t ValueType) string {
	switch t {
	case TypeBool:
		return "bool"
	case TypeString:
		return "string"
	case TypeInt:
		return "int"
	case TypeStringList:
		return "list(string)"
	case TypeSearchResults:
		return "SearchResults"
	case TypeTimestamp:
		return "timestamp"
	case TypeDyn:
		return "dyn"
	}
	return "unknown"
}

// FormatSignatures returns each argument list of a function as "meaning(string) -> SearchResults", of a
// stage as "SearchResults.top(int) -> SearchResults" and a variable as "kind: string".
func FormatSignatures(f Function) []string {
	if f.IsVariable {
		return []string{f.Name + ": " + GetTypeName(f.Result)}
	}
	overloads := f.Overloads
	if len(overloads) == 0 {
		overloads = [][]ValueType{f.Args}
	}
	if len(f.OptionalArgs) > 0 {
		overloads = append(overloads, append(slices.Clone(f.Args), f.OptionalArgs...))
	}
	name := f.Name
	if f.IsStage {
		name = GetTypeName(TypeSearchResults) + "." + f.Name
	}
	signatures := make([]string, len(overloads))
	for i, args := range overloads {
		names := make([]string, len(args))
		for j, arg := range args {
			names[j] = GetTypeName(arg)
		}
		signatures[i] = name + "(" + strings.Join(names, ", ") + ") -> " + GetTypeName(f.Result)
	}
	return signatures
}

// GetCatalogRevision returns the hex SHA-256 of the name, kind, level, signatures, summary, description and
// example of every function of the catalog; it changes whenever one of them does.
func GetCatalogRevision(c Catalog) string {
	hash := sha256.New()
	for _, f := range c.Functions {
		fields := []string{
			f.Name, strconv.FormatBool(f.IsVariable), strconv.FormatBool(f.IsStage), strconv.Itoa(int(f.Level)),
			strings.Join(FormatSignatures(f), "\x01"), f.Summary, f.Description, f.Example,
		}
		hash.Write([]byte(strings.Join(fields, "\x00") + "\x02"))
	}
	return hex.EncodeToString(hash.Sum(nil))
}
