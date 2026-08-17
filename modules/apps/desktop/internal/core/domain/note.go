package domain

// Note is a parsed markdown file. Frontmatter is kept as it was found: the
// application owns a closed set of keys and preserves everything else verbatim,
// so the parser is not allowed to normalise or drop what it does not recognise.
type Note struct {
	Ref   FileRef
	Title string

	// ID is what the note carries in its frontmatter, if it carries one. A note
	// written outside the application has none: it is indexed in full and simply
	// cannot be a stable target.
	ID string

	Frontmatter map[string]any
	Headings    []Heading
	Links       []Link
	Body        string

	// Problems are what was wrong with the file and could not be repaired: an
	// unreadable frontmatter block, a link with no role. They are shown rather
	// than fixed, because fixing means guessing at what the user wrote.
	Problems []string

	// FrontmatterErr is set when the block between the delimiters is not valid
	// YAML. The note is still indexed — its body is readable text either way —
	// and the file is never repaired in place, because that means guessing at
	// what the user wrote.
	FrontmatterErr string
}

// Heading is one ATX heading of a note, in document order.
//
// Line is counted from the first line of the body, and Offset is the byte the
// heading's own line begins at in the body. The frontmatter is in neither: both
// address the body the parser produced.
type Heading struct {
	Level  int
	Text   string
	Line   int
	Offset int
}

// VaultProblem is something in a vault that could not be acted on and was not
// guessed at. It is shown to the person: repairing it means deciding what they
// meant.
//
// Every problem belongs to one note: the file somebody would open to settle it.
// For a link that reaches two notes that is the note the link is written in,
// and neither of the notes it could mean.
type VaultProblem struct {
	Path  string
	Check Check
	// Detail says what is wrong, in the terms the file itself uses.
	Detail string

	// Target is where the link goes, for the checks that are about one.
	Target Address
	// Candidates is what that link could mean, when several notes answer to it.
	Candidates []string
}

// Check is one thing that can be wrong with a vault, and the name of whatever
// noticed it.
//
// It is on the problem so a person tidying a vault can take one kind at a
// time. A new rule is a new name here.
type Check string

const (
	// CheckParse is what reading one file turned up: a link with no role, a role
	// nobody decided on, an identifier that is not one.
	CheckParse Check = "parse"
	// CheckFrontmatter is a frontmatter block that is not YAML. The note is
	// still indexed, and nothing may write to it until the block reads.
	CheckFrontmatter Check = "frontmatter"
	// CheckAmbiguous is a link that reaches more than one note. It is not
	// broken — it reaches the nearest — but which one that is can change when
	// either of them moves.
	CheckAmbiguous Check = "ambiguous"
	// CheckDangling is a link that reaches nothing. Legitimate while a note is
	// being written and worth seeing afterwards.
	CheckDangling Check = "dangling"
)
