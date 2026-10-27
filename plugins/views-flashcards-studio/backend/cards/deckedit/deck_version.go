package deckedit

import "github.com/jiva-studio/numen/libs/plugin-sdk-go/host/domain"

// DeckVersion is a deck as the caller read it. A write over a deck that has
type DeckVersion struct {
	Path     string
	FileStat domain.FileStat
}
