package host

import (
	notesdomain "github.com/jiva-studio/numen/libs/notes/domain"
)

type VaultInfo = notesdomain.Vault
type VaultID = notesdomain.VaultID
type FileStat = notesdomain.FileStat
type SourceKind = notesdomain.SourceKind
type Note = notesdomain.Note
type NoteRef = notesdomain.NoteRef
type NoteType = notesdomain.NoteType
type ResolvedLink = notesdomain.ResolvedLink
type Link = notesdomain.Link
type Address = notesdomain.Address
type LinkRole = notesdomain.LinkRole
type LinkMarker = notesdomain.LinkMarker
type Connection = notesdomain.Connection
type Side = notesdomain.Side
type RelatedNote = notesdomain.RelatedNote
type Backlink = notesdomain.Backlink
type VaultProblem = notesdomain.VaultProblem
type TextVersion = notesdomain.TextVersion
type TextVersionID = notesdomain.TextVersionID
type TextKind = notesdomain.TextKind
type TextSegment = notesdomain.TextSegment
type RawSelector = notesdomain.Selector
type VersionText = notesdomain.VersionText
type ByteRange = notesdomain.ByteRange

const (
	KindFile       = notesdomain.KindFile
	KindNote       = notesdomain.KindNote
	NoteExtension  = notesdomain.NoteExtension
	TypeNote       = notesdomain.TypeNote
	RoleParent     = notesdomain.RoleParent
	RoleChild      = notesdomain.RoleChild
	RoleJump       = notesdomain.RoleJump
	RoleRef        = notesdomain.RoleRef
	RoleAttachment = notesdomain.RoleAttachment
	MarkerNone     = notesdomain.MarkerNone
	MarkerArrow    = notesdomain.MarkerArrow
	SchemeName     = notesdomain.SchemeName
	SchemeNote     = notesdomain.SchemeNote
	CheckParse     = notesdomain.CheckParse
)

var (
	NewTextVersionID = notesdomain.NewTextVersionID
	NewVersionText   = notesdomain.NewVersionText
	FindLiveVersion  = notesdomain.FindLiveVersion
	HashContent      = notesdomain.HashContent
	ParseAddress     = notesdomain.ParseAddress
	FoldName         = notesdomain.FoldName
	Basename         = notesdomain.Basename
	LinkName         = notesdomain.LinkName
	ChooseMatch      = notesdomain.ChooseMatch
)
