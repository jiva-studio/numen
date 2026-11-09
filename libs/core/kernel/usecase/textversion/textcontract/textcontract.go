// Package textcontract re-exports the text port contract.
package textcontract

import (
	"github.com/jiva-studio/numen/libs/core/kernel/domain/text"
	"github.com/jiva-studio/numen/libs/core/kernel/port/textport"
)

type (
	ContentHash         = textport.ContentHash
	ContentHashes       = textport.ContentHashes
	TextFile            = textport.TextFile
	TextFileStores      = textport.TextFileStores
	TextFiles           = textport.TextFiles
	TextPlugins         = textport.TextPlugins
	TextReaders         = textport.TextReaders
	Texts               = textport.Texts
	VaultTexts          = textport.VaultTexts
	VersionListener     = textport.VersionListener
	VersionRepositories = textport.VersionRepositories
	VersionRepository   = textport.VersionRepository
)

var (
	ErrInvalidVersion = text.ErrInvalidVersion
	ErrNoText         = text.ErrNoText
	ErrNotTextPlugin  = text.ErrNotTextPlugin
)
