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

var Basename = notesdomain.Basename
var Filename = notesdomain.Filename
var FoldName = notesdomain.FoldName
var LinkName = notesdomain.LinkName
var ChooseMatch = notesdomain.ChooseMatch
var ParseAddress = notesdomain.ParseAddress
var FindLiveVersion = notesdomain.FindLiveVersion
var NewTextVersionID = notesdomain.NewTextVersionID
var NewVersionText = notesdomain.NewVersionText
var HashContent = notesdomain.HashContent
