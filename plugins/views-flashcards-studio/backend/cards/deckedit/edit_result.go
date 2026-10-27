package deckedit

import "github.com/jiva-studio/numen/libs/notes/domain"

type EditResult struct {
	FileStat domain.FileStat
	Cards    int
}
