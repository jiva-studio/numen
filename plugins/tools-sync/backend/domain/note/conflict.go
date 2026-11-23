package note

import (
	"fmt"
	"path"
	"strings"
	"time"

	"golang.org/x/text/cases"
	"golang.org/x/text/unicode/norm"
)

// GetConflictCopyPath returns the path of the copy that keeps the other text of a note changed on two computers, in the
// folder of the note, in normal form C. Number 1 has no number in its name.
func GetConflictCopyPath(notePath string, day time.Time, number int) string {
	composed := norm.NFC.String(notePath)
	stem := strings.TrimSuffix(path.Base(composed), path.Ext(composed))
	name := fmt.Sprintf("%s (conflict %s", stem, day.Format(time.DateOnly))
	if number > 1 {
		name = fmt.Sprintf("%s %d", name, number)
	}
	return path.Join(path.Dir(composed), name+").md")
}

// IsCaseClash reports whether two paths are different names that differ in letter case once both are in normal form C.
func IsCaseClash(a, b string) bool {
	composedA, composedB := norm.NFC.String(a), norm.NFC.String(b)
	return composedA != composedB && cases.Fold().String(composedA) == cases.Fold().String(composedB)
}
