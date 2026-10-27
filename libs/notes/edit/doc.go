// Package edit reads a note held in a vault and changes its contents, through the few calls on its
// files that doing so takes.
//
// A note is its frontmatter and its prose. A read hands over the prose; an edit reads the file,
// changes the one thing it was asked to, and puts the file back over the file stat it read, so
// that a note changed in the meantime is left alone. The rules for a read and for an edit are
// written once here and used by every program that works a vault, whatever it reaches the files
// through.
package edit
