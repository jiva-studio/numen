package searchquery

import (
	"errors"
	"fmt"
	"path"
	"slices"
	"strings"

	"golang.org/x/text/unicode/norm"

	"github.com/jiva-studio/numen/libs/note/domain"
)

// LinkDirection is which end of a link the document of a link function stands at.
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
	Description  string
	Example      string
}

const targetHelp = " The target is a note name, a vault path or note://<id>; a name several notes answer to is an error that lists them. The second argument requires the link's type."

var linkFunctions = []LinkFunction{
	{Name: "hasLinkTo", Direction: DirectionTo, Description: "documents with a link of any role to the target, a note or a file", Example: `hasLinkTo("Entropy")`},
	{Name: "hasLinkFrom", Direction: DirectionFrom, Description: "documents the target note links to, by any role", Example: `hasLinkFrom("Reading list")`},
	{Name: "hasReferenceTo", Role: domain.RoleRef, Direction: DirectionTo, Description: "notes with a ref link (a wikilink in the text) to the target", Example: `hasReferenceTo("Entropy")`},
	{Name: "hasReferenceFrom", Role: domain.RoleRef, Direction: DirectionFrom, Description: "documents the target note links to by a ref link", Example: `hasReferenceFrom("Entropy")`},
	{Name: "hasJumpTo", Role: domain.RoleJump, Direction: DirectionTo, Description: "notes with a jump link to the target", Example: `hasJumpTo("Entropy")`},
	{Name: "hasJumpFrom", Role: domain.RoleJump, Direction: DirectionFrom, Description: "notes the target note links to by a jump link", Example: `hasJumpFrom("Entropy")`},
	{Name: "isParentOf", Direction: DirectionAbove, Description: "notes directly above the target, written at either end", Example: `isParentOf("Entropy")`},
	{Name: "isChildOf", Direction: DirectionBelow, Description: "notes directly under the target, written at either end", Example: `isChildOf("Physics")`},
	{Name: "isAncestorOf", Direction: DirectionAbove, IsTransitive: true, Description: "notes above the target at any depth up to the limit", Example: `isAncestorOf("Entropy")`},
	{Name: "isDescendantOf", Direction: DirectionBelow, IsTransitive: true, Description: "notes under the target at any depth up to the limit", Example: `isDescendantOf("Physics") && meaning("heat")`},
	{Name: "hasAttachment", Role: domain.RoleAttachment, Direction: DirectionTo, Description: "notes with an attachment link to the file path or address", Example: `hasAttachment("scans/1897-letter.pdf")`},
	{Name: "isAttachedTo", Role: domain.RoleAttachment, Direction: DirectionFrom, Description: "indexed files the target note attaches", Example: `isAttachedTo("Letters")`},
}

// FindLinkFunction returns the link function of a name.
func FindLinkFunction(name string) (LinkFunction, bool) {
	index := slices.IndexFunc(linkFunctions, func(f LinkFunction) bool { return f.Name == name })
	if index < 0 {
		return LinkFunction{}, false
	}
	return linkFunctions[index], true
}

func getLinkCatalog() []Function {
	out := make([]Function, len(linkFunctions))
	for i, f := range linkFunctions {
		out[i] = Function{
			Name: f.Name, Args: []ValueType{TypeString}, OptionalArgs: []ValueType{TypeString}, Result: TypeBool,
			Level: LevelDocument, Description: f.Description + "." + targetHelp, Example: f.Example,
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

// CleanPath returns a written file address as a vault path: composed, cleaned, without a leading slash. A
// value that climbs above the vault root is returned composed and otherwise unchanged.
func CleanPath(value string) string {
	composed := norm.NFC.String(value)
	cleaned := strings.TrimPrefix(path.Clean("/"+composed), "/")
	if relative := path.Clean(composed); relative == ".." || strings.HasPrefix(relative, "../") {
		return composed
	}
	return cleaned
}

// ListFilePaths returns the vault paths a link written in a note may point at as a file, in priority order:
// the value from the vault root, then from the folder of the note. A name and an asset address are paths;
// any other address is itself.
func ListFilePaths(from string, target domain.Address) []string {
	if target.Scheme != domain.SchemeName && target.Scheme != domain.SchemeAsset {
		return []string{target.String()}
	}
	fromRoot := CleanPath(target.Value)
	fromFolder := CleanPath(path.Join(path.Dir(from), target.Value))
	if fromFolder == fromRoot {
		return []string{fromRoot}
	}
	return []string{fromRoot, fromFolder}
}

// ChooseFilePath returns the path or address a link written in a note points at as a file: the first of
// ListFilePaths that is indexed, else the first.
func ChooseFilePath(from string, target domain.Address, indexed map[string]bool) string {
	paths := ListFilePaths(from, target)
	for _, p := range paths {
		if indexed[p] {
			return p
		}
	}
	return paths[0]
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
