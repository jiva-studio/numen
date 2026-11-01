// Package domain exposes the note domain types of the notes library to plugins.
package domain

import (
	notesdomain "github.com/jiva-studio/numen/libs/notes/domain"
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
type VaultProblem = notesdomain.VaultProblem
type Check = notesdomain.Check
type TextVersion = notesdomain.TextVersion
type TextVersionID = notesdomain.TextVersionID
type TextKind = notesdomain.TextKind
type TextSegment = notesdomain.TextSegment
type Selector = notesdomain.Selector
type VersionText = notesdomain.VersionText
type ByteRange = notesdomain.ByteRange

const (
	KindFile      = notesdomain.KindFile
	KindNote      = notesdomain.KindNote
	NoteExtension = notesdomain.NoteExtension
	TypeNote      = notesdomain.TypeNote
	RoleRef       = notesdomain.RoleRef
	SchemeName    = notesdomain.SchemeName
	SchemeNote    = notesdomain.SchemeNote
	CheckParse    = notesdomain.CheckParse
)

var ErrInvalidName = notesdomain.ErrInvalidName

func GetBasename(path string) string {
	return notesdomain.Basename(path)
}

func Basename(path string) string { return notesdomain.Basename(path) }

func Filename(title string) (filename string, exact bool, err error) {
	return notesdomain.Filename(title)
}

func FoldName(name string) string { return notesdomain.FoldName(name) }

func LinkName(target string) string { return notesdomain.LinkName(target) }

func ChooseMatch(from, name string, candidates []string) (bestMatch string, ambiguous bool) {
	return notesdomain.ChooseMatch(from, name, candidates)
}

func ParseAddress(raw string) Address { return notesdomain.ParseAddress(raw) }

func FindLiveVersion(versions []TextVersion, fileHash string, isIntact func(TextVersion) bool) (TextVersion, bool) {
	return notesdomain.FindLiveVersion(versions, fileHash, isIntact)
}

func NewTextVersionID(pluginID, key string) TextVersionID {
	return notesdomain.NewTextVersionID(pluginID, key)
}

func NewVersionText(version TextVersion, segments []TextSegment) VersionText {
	return notesdomain.NewVersionText(version, segments)
}

func HashContent(raw []byte) string { return notesdomain.HashContent(raw) }
