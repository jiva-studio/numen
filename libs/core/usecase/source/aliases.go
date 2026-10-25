package source

import (
	"github.com/jiva-studio/numen/libs/core/usecase/source/artifacts"
)

type (
	Artifact                      = artifacts.Artifact
	ArtifactText                  = artifacts.ArtifactText
	Document                      = artifacts.Document
	GetArtifactList               = artifacts.GetArtifactList
	GetArtifactText               = artifacts.GetArtifactText
	GetDocumentList               = artifacts.GetDocumentList
	GetRawTranscript              = artifacts.GetRawTranscript
	GetRecognitionCorrectionState = artifacts.GetRecognitionCorrectionState
	GetRunProgress                = artifacts.GetRunProgress
	GetTranscriptCorrectionState  = artifacts.GetTranscriptCorrectionState
	GetTranscript                 = artifacts.GetTranscript
	RunProgress                   = artifacts.RunProgress
	RunState                      = artifacts.RunState
	SourceTexts                   = artifacts.SourceTexts
	StartOutcome                  = artifacts.StartOutcome
	StoredText                    = artifacts.StoredText

	Transcript = artifacts.Transcript

	sourceTexts = artifacts.SourceTexts
)

const (
	KindRecognition = artifacts.KindRecognition
	KindTranscript  = artifacts.KindTranscript
	KindArticle     = artifacts.KindArticle
	KindCopy        = artifacts.KindCopy

	RunNone     = artifacts.RunNone
	RunComplete = artifacts.RunComplete
	RunActive   = artifacts.RunActive
	RunStopped  = artifacts.RunStopped
	RunNoSpeech = artifacts.RunNoSpeech
	RunUnopened = artifacts.RunUnopened

	Queued = artifacts.Queued
)

var (
	NewGetArtifactList = artifacts.NewGetArtifactList
	NewGetArtifactText = artifacts.NewGetArtifactText
	NewGetDocumentList = artifacts.NewGetDocumentList
	NewSourceTexts     = artifacts.NewSourceTexts
)
