package share

import (
	"errors"
	"fmt"
	"strings"
)

// NestingError is a refusal to share Folder because of the share Shared, and says which share it is. Kind is
// ErrInsideShare or ErrHoldsShare. Both are paths of the vault; the empty path is the vault itself.
type NestingError struct {
	Kind   error
	Folder string
	Shared string
}

// Error returns one plain sentence that names the share in the way.
func (e NestingError) Error() string {
	switch {
	case errors.Is(e.Kind, ErrHoldsShare):
		return fmt.Sprintf("%s cannot be shared while its folder «%s» is shared", getName(e.Folder), e.Shared)
	case strings.EqualFold(e.Folder, e.Shared):
		return fmt.Sprintf("%s is shared already", getName(e.Folder))
	case e.Shared == "":
		return fmt.Sprintf("%s is inside the shared vault", getName(e.Folder))
	default:
		return fmt.Sprintf("%s is inside the shared folder «%s»", getName(e.Folder), e.Shared)
	}
}

// Is reports whether target is the sentinel the error stands for.
func (e NestingError) Is(target error) bool { return target == e.Kind }

func getName(folder string) string {
	if folder == "" {
		return "the vault"
	}
	return "«" + folder + "»"
}
