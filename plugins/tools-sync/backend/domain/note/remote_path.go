package note

import (
	"fmt"
	"path"
	"strings"
	"unicode"
)

// CheckRemotePath returns nil for a path that a device of the account may write into the vault: the path is
// clean and relative, has no backslash, control character or any of : * ? " < > |, no segment is a Windows device
// name (CON, NUL, COM1 and the like, with or without an extension) or ends in a dot or a space, and the path is one
// that IsSyncable names. The error wraps ErrInvalidPath.
func CheckRemotePath(notePath string) error {
	if notePath == "" || path.Clean(notePath) != notePath || strings.HasPrefix(notePath, "/") || strings.HasPrefix(notePath, "../") || notePath == ".." {
		return fmt.Errorf("%w: the path is not clean and relative", ErrInvalidPath)
	}
	if strings.ContainsFunc(notePath, isForbiddenRune) {
		return fmt.Errorf("%w: the path holds a character no platform accepts", ErrInvalidPath)
	}
	for segment := range strings.SplitSeq(notePath, "/") {
		if isDeviceName(segment) || strings.HasSuffix(segment, ".") || strings.HasSuffix(segment, " ") {
			return fmt.Errorf("%w: a name that Windows cannot hold", ErrInvalidPath)
		}
	}
	if !IsSyncable(notePath) {
		return fmt.Errorf("%w: the path is not one that syncs", ErrInvalidPath)
	}
	return nil
}

func isForbiddenRune(r rune) bool {
	return r == '\\' || unicode.IsControl(r) || strings.ContainsRune(`:*?"<>|`, r)
}

func isDeviceName(segment string) bool {
	stem, _, _ := strings.Cut(segment, ".")
	switch upper := strings.ToUpper(strings.TrimRight(stem, " ")); upper {
	case "CON", "PRN", "AUX", "NUL", "CONIN$", "CONOUT$":
		return true
	default:
		return len(upper) == 4 && (strings.HasPrefix(upper, "COM") || strings.HasPrefix(upper, "LPT")) && upper[3] >= '1' && upper[3] <= '9'
	}
}
