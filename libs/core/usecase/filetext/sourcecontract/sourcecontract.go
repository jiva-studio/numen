package sourcecontract

import "github.com/jiva-studio/numen/libs/core/port/sourceport"

type (
	ImportedFile = sourceport.ImportedFile
	ImportedFiles = sourceport.ImportedFiles
	PassageQueries = sourceport.PassageQueries
	SourceQueries = sourceport.SourceQueries
	SourceRepository = sourceport.SourceRepository
	SourceText = sourceport.SourceText
)

var (
	ErrNotADocument = sourceport.ErrNotADocument
)
