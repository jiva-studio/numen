package source

import "github.com/jiva-studio/numen/modules/libs/core/port"

// Texts is everything that reads or writes the text made from a source.
type Texts struct {
	GetNoteAddress                GetNoteAddress
	GetRunProgress                GetRunProgress
	GetTranscriptCorrectionState  GetTranscriptCorrectionState
	GetRecognitionCorrectionState GetRecognitionCorrectionState
	GetTranscript                 GetTranscript
	GetRawTranscript              GetRawTranscript
	WriteTranscript               WriteTranscript
	GetCopyLocation               GetCopyLocation
	RemoveCopy                    RemoveCopy
	OpenCopy                      OpenCopy
}

func NewTexts(
	readers port.VaultReaders,
	writers port.VaultWriters,
	sources port.SourceQueries,
	derived port.DerivedStores,
) Texts {
	shared := sourceTexts{readers: readers, sources: sources, derived: derived}
	location := GetCopyLocation{texts: shared}
	return Texts{
		GetNoteAddress:                GetNoteAddress{texts: shared},
		GetRunProgress:                GetRunProgress{texts: shared},
		GetTranscriptCorrectionState:  GetTranscriptCorrectionState{texts: shared},
		GetRecognitionCorrectionState: GetRecognitionCorrectionState{texts: shared},
		GetTranscript:                 GetTranscript{texts: shared},
		GetRawTranscript:              GetRawTranscript{texts: shared},
		WriteTranscript:               WriteTranscript{texts: shared},
		GetCopyLocation:               location,
		RemoveCopy:                    RemoveCopy{texts: shared, location: location, writers: writers},
		OpenCopy:                      OpenCopy{texts: shared},
	}
}
