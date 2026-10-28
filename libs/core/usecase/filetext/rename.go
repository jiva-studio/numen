package filetext

import (
	"github.com/jiva-studio/numen/libs/core/internal/vaultreading"
)

// FindRemovedSources is the sources other than notes the index holds at and
// under the paths, with the content hash each was read with.
var FindRemovedSources = vaultreading.FindRemovedSources

// MoveRenamedSources moves each removed source to the path among added whose
// file holds the same content, and returns the paths it moved from.
var MoveRenamedSources = vaultreading.MoveRenamedSources
