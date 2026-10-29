package deckedit

import "github.com/jiva-studio/numen/libs/plugin-sdk-go/host/domain"

// EditResult is the file stat and the card count after an edit.
type EditResult struct {
	FileStat domain.FileStat
	Cards    int
}
