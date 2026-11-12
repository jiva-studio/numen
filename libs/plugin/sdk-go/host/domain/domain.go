// Package domain exposes the note domain types of the notes library to plugins.
package domain

import (
	"path/filepath"
	"strings"

	notesdomain "github.com/jiva-studio/numen/libs/note/domain"
)

type Vault = notesdomain.Vault
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
type NewRelatedNote = notesdomain.NewRelatedNote
type Backlink = notesdomain.Backlink
type VaultProblem = notesdomain.VaultProblem
type MoveResult = notesdomain.MoveResult
type TextVersion = notesdomain.TextVersion
type TextVersionID = notesdomain.TextVersionID
type TextKind = notesdomain.TextKind
type TextSourceProperty = notesdomain.TextSourceProperty
type TextSegment = notesdomain.TextSegment

// Selector locates a segment of a text version in its file, as a JSON object. The key page is the PDF
// page counted from 1, cfi the EPUB CFI, and t the span of a recording, [from, to] in seconds. Other
// keys may be present and are ignored.
type Selector = notesdomain.Selector
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

var ErrInvalidName = notesdomain.ErrInvalidName

func Basename(path string) string { return notesdomain.Basename(path) }

func ParseAddress(raw string) Address { return notesdomain.ParseAddress(raw) }

func IsKnownRole(r LinkRole) bool { return notesdomain.IsKnownRole(r) }

func IsWritableMarker(m LinkMarker) bool { return notesdomain.IsWritableMarker(m) }

func Filename(title string) (filename string, exact bool, err error) {
	return notesdomain.Filename(title)
}

func NewVersionText(version TextVersion, segments []TextSegment) VersionText {
	return notesdomain.NewVersionText(version, segments)
}

func HashContent(raw []byte) string { return notesdomain.HashContent(raw) }

// IsNotePath reports whether a vault path is a note: its extension is NoteExtension, in any case, as
// target.isNote of a when condition holds it.
func IsNotePath(path string) bool {
	return strings.EqualFold(filepath.Ext(path), NoteExtension)
}
