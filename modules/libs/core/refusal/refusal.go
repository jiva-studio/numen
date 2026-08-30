// Package refusal turns an outcome into the refusal the protocol carries.
//
// Every window that shows a note it could not read names the reason the same
// way, so the translation is written once and not one per window.
package refusal

import (
	v1 "github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1"

	"github.com/jiva-studio/numen/modules/libs/core/usecase/note"
)

// Of says which refusal an outcome is, and whether it is one at all.
func Of(o note.Outcome) (v1.Refusal, bool) {
	switch o {
	case note.Missing:
		return v1.Refusal_REFUSAL_MISSING, true
	case note.NotANote:
		return v1.Refusal_REFUSAL_NOT_A_NOTE, true
	case note.NotText:
		return v1.Refusal_REFUSAL_NOT_TEXT, true
	case note.TooLarge:
		return v1.Refusal_REFUSAL_TOO_LARGE, true
	case note.Unreadable:
		return v1.Refusal_REFUSAL_UNREADABLE, true
	default:
		return v1.Refusal_REFUSAL_UNSPECIFIED, false
	}
}
