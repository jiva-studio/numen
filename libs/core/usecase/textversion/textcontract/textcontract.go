package textcontract

import "github.com/jiva-studio/numen/libs/core/port/textport"

type (
	MenuItem = textport.MenuItem
	TextCreation = textport.TextCreation
	TextFile = textport.TextFile
	TextFileStores = textport.TextFileStores
	TextFiles = textport.TextFiles
	TextPlugin = textport.TextPlugin
	TextPlugins = textport.TextPlugins
	TextProvider = textport.TextProvider
	TextProviders = textport.TextProviders
	TextReaders = textport.TextReaders
	Texts = textport.Texts
	VaultTexts = textport.VaultTexts
	VersionListener = textport.VersionListener
	VersionRepositories = textport.VersionRepositories
	VersionRepository = textport.VersionRepository
)

var (
	ErrInvalidVersion = textport.ErrInvalidVersion
	ErrNoText = textport.ErrNoText
	ErrNotTextPlugin = textport.ErrNotTextPlugin
	ErrSourceNotAccepted = textport.ErrSourceNotAccepted
)
