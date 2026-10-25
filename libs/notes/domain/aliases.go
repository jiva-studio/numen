package domain

import (
	"github.com/jiva-studio/numen/libs/notes/domain/artifacts"
	"github.com/jiva-studio/numen/libs/notes/domain/name"
)

type (
	// From name subpackage
	SourceKind  = name.SourceKind
	Fingerprint = name.Fingerprint
	URL         = name.URL

	// From artifacts subpackage
	ArtifactFile        = artifacts.ArtifactFile
	ArtifactName        = artifacts.ArtifactName
	CorrectionLine      = artifacts.CorrectionLine
	Rect                = artifacts.Rect
	HighlightBox        = artifacts.HighlightBox
	OCRBox              = artifacts.OCRBox
	OCRBlock            = artifacts.OCRBlock
	OCRPageStart        = artifacts.OCRPageStart
	OCRPart             = artifacts.OCRPart
	Place               = artifacts.Place
	ByteSpan            = artifacts.ByteSpan
	UnitSpan            = artifacts.UnitSpan
	ProofreadLine       = artifacts.ProofreadLine
	ProofreadPage       = artifacts.ProofreadPage
	ProofreadCorrection = artifacts.ProofreadCorrection
	ScanProofreading    = artifacts.ScanProofreading
	SpeechWindow        = artifacts.SpeechWindow
	SpeechProofreading  = artifacts.SpeechProofreading
	TranscriptCue       = artifacts.TranscriptCue
)

const (
	// From name subpackage
	KindNote      = name.KindNote
	KindBook      = name.KindBook
	KindRecording = name.KindRecording
	KindURL       = name.KindURL
	NoteExtension = name.NoteExtension
	URLExtension  = name.URLExtension

	// From artifacts subpackage
	ASR                 = artifacts.ASR
	OCR                 = artifacts.OCR
	Captions            = artifacts.Captions
	Article             = artifacts.Article
	CopyExtension       = artifacts.CopyExtension
	CopyType            = artifacts.CopyType
	ArtifactText        = artifacts.ArtifactText
	ArtifactPartial     = artifacts.ArtifactPartial
	ArtifactCorrections = artifacts.ArtifactCorrections
	ArtifactParts       = artifacts.ArtifactParts
	ArtifactBoxes       = artifacts.ArtifactBoxes
	ArtifactProofread   = artifacts.ArtifactProofread
	ArtifactAnswer      = artifacts.ArtifactAnswer
	ArtifactProducer    = artifacts.ArtifactProducer
	ArtifactCopy        = artifacts.ArtifactCopy
	ArtifactIndexClaim  = artifacts.ArtifactIndexClaim
	ArtifactReviewLog   = artifacts.ArtifactReviewLog
	MostHighlights      = artifacts.MostHighlights
)

var (
	// From name subpackage
	ErrUnnameable      = name.ErrUnnameable
	ErrNotAURL         = name.ErrNotAURL
	Filename           = name.Filename
	GetReducedFilename = name.GetReducedFilename
	ChooseMatch        = name.ChooseMatch
	FoldName           = name.FoldName
	Basename           = name.Basename
	LinkName           = name.LinkName
	ParseURL           = name.ParseURL

	// From artifacts subpackage
	Recordings            = artifacts.Recordings
	NewArtifactName       = artifacts.NewArtifactName
	NewCopyName           = artifacts.NewCopyName
	NewIndexClaimName     = artifacts.NewIndexClaimName
	NewReviewLogName      = artifacts.NewReviewLogName
	IsTimedProducer       = artifacts.IsTimedProducer
	GetAddressProducers   = artifacts.GetAddressProducers
	GetArtifactNames      = artifacts.GetArtifactNames
	GetAddressTextNames   = artifacts.GetAddressTextNames
	GetCorrectedBoxes     = artifacts.GetCorrectedBoxes
	GetCorrectedProse     = artifacts.GetCorrectedProse
	FindUnaskedCorrection = artifacts.FindUnaskedCorrection
	MediaType             = artifacts.MediaType
	RecordingExtensions   = artifacts.RecordingExtensions
)
