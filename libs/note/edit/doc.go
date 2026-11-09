// Package edit reads a note stored in a vault and changes its contents, through
// the small set of file calls that this requires.
//
// A note is its frontmatter and its prose. A read returns the prose; an edit
// reads the file, applies the one requested change, and writes the file back
// against the file stat it read, so a note changed in the meantime is not
// overwritten. The rules for reads and edits are defined here once and used by
// every program that works on a vault, whatever file access it uses.
package edit
