package vault

import (
	"github.com/jiva-studio/numen/libs/core/internal/vaultreading"
)

// ReadWholeVault makes a vault answer: its notes are read, the text versions of
// its other files are cut, and what was cut is given its vectors.
type ReadWholeVault = vaultreading.ReadWholeVault

// ReadWholeVaultResult is what each pass did.
type ReadWholeVaultResult = vaultreading.ReadWholeVaultResult

// NewReadWholeVault is the three passes, in the order they stand in: the notes
// walked, the file texts cut, and the vectors made.
var NewReadWholeVault = vaultreading.NewReadWholeVault
