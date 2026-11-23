package searchquery

import (
	"errors"
	"fmt"
	"path"
	"slices"
	"strings"

	"github.com/jiva-studio/numen/libs/shared/note/domain"
)

// LinkDirection is the end of a link the document of a link function is at.
type LinkDirection int

const (
	// DirectionTo: the document writes a link to the target.
	DirectionTo LinkDirection = iota
	// DirectionFrom: the target writes a link to the document.
	DirectionFrom
	// DirectionAbove: the document is a parent of the target, written at either end.
	DirectionAbove
	// DirectionBelow: the document is a child of the target, written at either end.
	DirectionBelow
)

// LinkFunction is a link function of the language. Role is empty for any role; IsTransitive follows the
// hierarchy to any depth.
type LinkFunction struct {
	Name         string
	Role         domain.LinkRole
	Direction    LinkDirection
	IsTransitive bool
	Summary      string
	Description  string
	Example      string
}

const targetHelp = " The target is a note name, a vault path or note://<id>; a name several notes answer to is an error that lists them. The second argument requires the link's type."

func listLinkFunctions() []LinkFunction {
	return []LinkFunction{
		{
			Name: "hasLinkTo", Direction: DirectionTo,
			Summary:     "documents that link to the target",
			Description: "documents with a link of any role to the target, a note or a file", Example: `hasLinkTo("Entropy")`,
		},
		{
			Name: "hasLinkFrom", Direction: DirectionFrom,
			Summary:     "documents the target links to",
			Description: "documents the target note links to, by any role", Example: `hasLinkFrom("Reading list")`,
		},
		{
			Name: "hasReferenceTo", Role: domain.RoleRef, Direction: DirectionTo,
			Summary:     "notes with a ref link to the target",
			Description: "notes with a ref link (a wikilink in the text) to the target", Example: `hasReferenceTo("Entropy")`,
		},
		{
			Name: "hasReferenceFrom", Role: domain.RoleRef, Direction: DirectionFrom,
			Summary:     "documents the target refers to",
			Description: "documents the target note links to by a ref link", Example: `hasReferenceFrom("Entropy")`,
		},
		{
			Name: "hasJumpTo", Role: domain.RoleJump, Direction: DirectionTo,
			Summary:     "notes with a jump link to the target",
			Description: "notes with a jump link to the target", Example: `hasJumpTo("Entropy")`,
		},
		{
			Name: "hasJumpFrom", Role: domain.RoleJump, Direction: DirectionFrom,
			Summary:     "notes the target links to by a jump link",
			Description: "notes the target note links to by a jump link", Example: `hasJumpFrom("Entropy")`,
		},
		{
			Name: "isParentOf", Direction: DirectionAbove,
			Summary:     "notes directly above the target",
			Description: "notes directly above the target, written at either end", Example: `isParentOf("Entropy")`,
		},
		{
			Name: "isChildOf", Direction: DirectionBelow,
			Summary:     "notes directly under the target",
			Description: "notes directly under the target, written at either end", Example: `isChildOf("Physics")`,
		},
		{
			Name: "isAncestorOf", Direction: DirectionAbove, IsTransitive: true,
			Summary:     "notes above the target at any depth",
			Description: "notes above the target at any depth up to the limit", Example: `isAncestorOf("Entropy")`,
		},
		{
			Name: "isDescendantOf", Direction: DirectionBelow, IsTransitive: true,
			Summary:     "notes under the target at any depth",
			Description: "notes under the target at any depth up to the limit", Example: `isDescendantOf("Physics") && meaning("heat")`,
		},
		{
			Name: "hasAttachment", Role: domain.RoleAttachment, Direction: DirectionTo,
			Summary:     "notes that attach the file",
			Description: "notes with an attachment link to the file path or address", Example: `hasAttachment("scans/1897-letter.pdf")`,
		},
		{
			Name: "isAttachedTo", Role: domain.RoleAttachment, Direction: DirectionFrom,
			Summary:     "files the target note attaches",
			Description: "indexed files the target note attaches", Example: `isAttachedTo("Letters")`,
		},
	}
}

// FindLinkFunction finds only the functions that read the link graph; words, meaning and the file conditions
// are not among them.
func FindLinkFunction(name string) (LinkFunction, bool) {
	functions := listLinkFunctions()
	index := slices.IndexFunc(functions, func(f LinkFunction) bool { return f.Name == name })
	if index < 0 {
		return LinkFunction{}, false
	}
	return functions[index], true
}

func getLinkCatalog() []Function {
	functions := listLinkFunctions()
	out := make([]Function, len(functions))
	for i, f := range functions {
		out[i] = Function{
			Name: f.Name, Args: []ValueType{TypeString}, OptionalArgs: []ValueType{TypeString}, Result: TypeBool,
			Level: LevelDocument, Summary: f.Summary, Description: f.Description + "." + targetHelp, Example: f.Example,
		}
	}
	return out
}

// IsLinkMatch reports whether a link has the role of the function (any when empty) and the type asked for
// (any when empty).
func IsLinkMatch(l domain.ResolvedLink, role domain.LinkRole, linkType string) bool {
	return (role == "" || l.Role == role) && (linkType == "" || l.Type == linkType)
}

// ErrNoTarget is a target of a link function that no note or indexed file answers to.
var ErrNoTarget = errors.New("no note or indexed file answers to the target")

// AmbiguousTargetError is a target name several notes answer to.
type AmbiguousTargetError struct {
	Target string
	Paths  []string
}

func (e *AmbiguousTargetError) Error() string {
	return fmt.Sprintf("%q matches %d notes: %s; write the path", e.Target, len(e.Paths), strings.Join(e.Paths, ", "))
}

// ChooseTarget picks the note a target name means among the notes that answer to it: an exact path, with or
// without the extension, or the one candidate. Several candidates without an exact path return
// *AmbiguousTargetError; no candidate returns "".
func ChooseTarget(name string, candidates []string) (string, error) {
	chosen, isAmbiguous := domain.ChooseMatch("", name, candidates)
	if !isAmbiguous {
		return chosen, nil
	}
	paths := slices.Clone(candidates)
	slices.Sort(paths)
	return "", &AmbiguousTargetError{Target: name, Paths: paths}
}

// CleanPath returns a written file address as a vault path: cleaned, without a leading slash. A value that
// climbs above the vault root is returned unchanged.
func CleanPath(value string) string {
	if relative := path.Clean(value); relative == ".." || strings.HasPrefix(relative, "../") {
		return value
	}
	return strings.TrimPrefix(path.Clean("/"+value), "/")
}

// GetFileName returns the folded last segment of a written file address, the key GroupFilesByName files a
// path under.
func GetFileName(value string) string {
	return domain.NormalizeName(path.Base(CleanPath(value)))
}

// GroupFilesByName returns the paths keyed by GetFileName.
func GroupFilesByName(paths []string) map[string][]string {
	out := make(map[string][]string, len(paths))
	for _, p := range paths {
		name := GetFileName(p)
		out[name] = append(out[name], p)
	}
	return out
}

// IsFileAddress reports whether an address is written as a vault path: a name or an asset.
func IsFileAddress(target domain.Address) bool {
	return target.Scheme == domain.SchemeName || target.Scheme == domain.SchemeAsset
}

// ChooseFile returns the file a link written in a note points at, by the priority a wikilink resolves by
// (domain.ChooseMatch): the path from the vault root, the path from the folder of the note, the one file of
// that name anywhere, else the nearest of several. It returns "" for no file or an address that is not a
// path.
func ChooseFile(from string, target domain.Address, files map[string][]string) string {
	if !IsFileAddress(target) {
		return ""
	}
	value := CleanPath(target.Value)
	chosen, _ := domain.ChooseMatch(from, value, files[GetFileName(value)])
	return chosen
}

// GetRelatives returns the notes one hierarchy step above or below the notes whose links and backlinks are
// given, written at either end: a parent link of a note and a child link to it put its other end above. Only
// links resolved into the vault count.
func GetRelatives(
	direction LinkDirection, linkType string, vaultID domain.VaultID, links, backlinks map[string][]domain.ResolvedLink,
) []string {
	forward, backward := domain.RoleParent, domain.RoleChild
	if direction == DirectionBelow {
		forward, backward = domain.RoleChild, domain.RoleParent
	}
	var out []string
	for _, list := range links {
		for _, l := range list {
			if IsLinkMatch(l, forward, linkType) && l.To != "" && (l.ToVault == "" || l.ToVault == vaultID) {
				out = append(out, l.To)
			}
		}
	}
	for _, list := range backlinks {
		for _, l := range list {
			if IsLinkMatch(l, backward, linkType) {
				out = append(out, l.From)
			}
		}
	}
	slices.Sort(out)
	return slices.Compact(out)
}
