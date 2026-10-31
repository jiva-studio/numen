package host

import (
	"github.com/jiva-studio/numen/libs/plugin-sdk-go/host/domain"
)

type VaultInfo = domain.Vault
type VaultID = domain.VaultID
type FileStat = domain.FileStat
type SourceKind = domain.SourceKind
type Note = domain.Note
type NoteRef = domain.NoteRef
type NoteType = domain.NoteType
type ResolvedLink = domain.ResolvedLink
type Link = domain.Link
type Address = domain.Address
type LinkRole = domain.LinkRole
type LinkMarker = domain.LinkMarker
type Connection = domain.Connection
type Side = domain.Side
type RelatedNote = domain.RelatedNote
type Backlink = domain.Backlink
type VaultProblem = domain.VaultProblem
type TextVersion = domain.TextVersion
type TextVersionID = domain.TextVersionID
type TextKind = domain.TextKind
type TextSegment = domain.TextSegment
type RawSelector = domain.Selector
type VersionText = domain.VersionText
type ByteRange = domain.ByteRange

const (
	KindFile       = domain.KindFile
	KindNote       = domain.KindNote
	NoteExtension  = domain.NoteExtension
	TypeNote       = domain.TypeNote
	RoleParent     = domain.RoleParent
	RoleChild      = domain.RoleChild
	RoleJump       = domain.RoleJump
	RoleRef        = domain.RoleRef
	RoleAttachment = domain.RoleAttachment
	MarkerNone     = domain.MarkerNone
	MarkerArrow    = domain.MarkerArrow
	SchemeName     = domain.SchemeName
	SchemeNote     = domain.SchemeNote
	CheckParse     = domain.CheckParse
)

var (
	NewTextVersionID = domain.NewTextVersionID
	NewVersionText   = domain.NewVersionText
	FindLiveVersion  = domain.FindLiveVersion
	HashContent      = domain.HashContent
	ParseAddress     = domain.ParseAddress
	FoldName         = domain.FoldName
	Basename         = domain.Basename
	LinkName         = domain.LinkName
	ChooseMatch      = domain.ChooseMatch
)
