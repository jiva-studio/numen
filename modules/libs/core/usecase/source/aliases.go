package source

import (
	"github.com/jiva-studio/numen/modules/libs/core/usecase/source/artifacts"
)

type (
	Artifact                      = artifacts.Artifact
	ArtifactText                  = artifacts.ArtifactText
	Correction                    = artifacts.Correction
	Document                      = artifacts.Document
	DocumentText                  = artifacts.DocumentText
	GetArtifactList               = artifacts.GetArtifactList
	GetArtifactText               = artifacts.GetArtifactText
	GetDocumentList               = artifacts.GetDocumentList
	GetDocumentText               = artifacts.GetDocumentText
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
	WriteCorrections              = artifacts.WriteCorrections
	WriteTranscript               = artifacts.WriteTranscript

	Transcript = artifacts.Transcript

	sourceTexts = artifacts.SourceTexts
)

const (
	MostDocumentText = artifacts.MostDocumentText

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
	ErrNotCorrectable = artifacts.ErrNotCorrectable
	ErrNoCue          = artifacts.ErrNoCue

	NewGetArtifactList  = artifacts.NewGetArtifactList
	NewGetArtifactText  = artifacts.NewGetArtifactText
	NewGetDocumentList  = artifacts.NewGetDocumentList
	NewGetDocumentText  = artifacts.NewGetDocumentText
	NewWriteCorrections = artifacts.NewWriteCorrections
	NewSourceTexts      = artifacts.NewSourceTexts
)
