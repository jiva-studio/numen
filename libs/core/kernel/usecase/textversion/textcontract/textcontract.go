// Package textcontract re-exports the text port contract.
package textcontract

import "github.com/jiva-studio/numen/libs/core/kernel/port/textport"

type (
	ContentHashes   = textport.ContentHashes
	TextFileStores  = textport.TextFileStores
	TextFiles       = textport.TextFiles
	TextPlugins     = textport.TextPlugins
	TextReaders     = textport.TextReaders
	Texts           = textport.Texts
	VaultTexts      = textport.VaultTexts
	VersionListener = textport.VersionListener
)
