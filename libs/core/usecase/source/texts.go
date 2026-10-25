package source

import (
	"github.com/jiva-studio/numen/libs/core/port/sourceport"
	"github.com/jiva-studio/numen/libs/core/port/vaultport"
)

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
	readers vaultport.VaultReaders,
	writers vaultport.VaultWriters,
	sources sourceport.SourceQueries,
	derived vaultport.DerivedStores,
) Texts {
	shared := NewSourceTexts(readers, sources, derived)
	location := GetCopyLocation{texts: shared}
	return Texts{
		GetNoteAddress:                GetNoteAddress{texts: shared},
		GetRunProgress:                GetRunProgress{Texts: shared},
		GetTranscriptCorrectionState:  GetTranscriptCorrectionState{Texts: shared},
		GetRecognitionCorrectionState: GetRecognitionCorrectionState{Texts: shared},
		GetTranscript:                 GetTranscript{Texts: shared},
		GetRawTranscript:              GetRawTranscript{Texts: shared},
		WriteTranscript:               WriteTranscript{Texts: shared},
		GetCopyLocation:               location,
		RemoveCopy:                    RemoveCopy{texts: shared, location: location, writers: writers},
		OpenCopy:                      OpenCopy{texts: shared},
	}
}
