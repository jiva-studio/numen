package vault

import (
	"fmt"
	"path"
	"strings"
)

// ImportFailure is one file that stayed outside the vault, by the name it
// carries on this machine and by what stopped it.
type ImportFailure struct {
	Name string
	Why  error
}

// ImportResult is what a drop came to: what the vault now holds, and what it
// does not.
type ImportResult struct {
	Imported []string
	Errors   []ImportFailure
}

// namedInAFailureText is how many of the files that stayed outside are named
// before the rest are counted.
const namedInAFailureText = 3

// DirectChildren returns the imported paths that sit directly in the folder
// into; the root is the empty path.
func (r ImportResult) DirectChildren(into string) []string {
	children := make([]string, 0, len(r.Imported))
	for _, imported := range r.Imported {
		if path.Dir(imported) == into || (into == "" && !strings.Contains(imported, "/")) {
			children = append(children, imported)
		}
	}
	return children
}

// DescribeFailures returns one line naming the first files that stayed outside
// and counting the rest; it is empty when nothing failed.
func (r ImportResult) DescribeFailures() string {
	if len(r.Errors) == 0 {
		return ""
	}
	messages := make([]string, 0, namedInAFailureText)
	for _, one := range r.Errors[:min(len(r.Errors), namedInAFailureText)] {
		messages = append(messages, fmt.Sprintf("%s: %v", one.Name, one.Why))
	}
	if rest := len(r.Errors) - len(messages); rest > 0 {
		messages = append(messages, fmt.Sprintf("and %d more", rest))
	}
	return strings.Join(messages, "; ")
}
