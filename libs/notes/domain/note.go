package domain

// Note is a parsed markdown file. Frontmatter is kept as it was found: the
// application owns a closed set of keys and preserves everything else verbatim,
// so the parser is not allowed to normalise or drop what it does not recognise.
type Note struct {
	FileStat FileStat
	Title    string

	// Type is what the note is. A note carrying no `type` is a note,
	// which is nearly every file in a vault.
	Type NoteType

	// ID is what the note carries in its frontmatter, if it carries one. A note
	// written outside the application has none: it is indexed in full and cannot
	// be a stable target.
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

// NoteType is the `type` key of a note's frontmatter. A note without one is an
// ordinary note. The application defines one type; a plugin assigns any other
// and declares in its manifest how the index stores notes of that type.
type NoteType string

// TypeNote is an ordinary note, which is a note carrying no `type` at all and
// so nearly every note in a vault.
const TypeNote NoteType = "note"

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

// VaultProblem is a condition in a vault that was not acted on automatically.
// It is shown to the user: repairing it requires deciding the intended meaning.
//
// Every problem belongs to one note: the file to open to resolve it.
// For a link that matches two notes, that is the note the link is written in,
// not either of the candidates.
type VaultProblem struct {
	Path string
	Kind Check
	// Detail describes the problem in the terms of the file itself.
	Detail string

	// Target is where the link goes, for the checks that are about one.
	Target Address
	// Candidates is what that link could mean, when several notes answer to it.
	Candidates []string
}

// Check is the name of the rule that detected a vault problem.
//
// It is stored on the problem, so problems can be handled by rule. A new rule
// adds a new name here.
type Check string

const (
	// CheckParse is a failure while reading one file: a link with no role, an
	// unlisted role, an invalid identifier.
	CheckParse Check = "parse"
	// CheckFrontmatter is a frontmatter block that is not YAML. The note is
	// still indexed, and nothing may write to it until the block reads.
	CheckFrontmatter Check = "frontmatter"
	// CheckAmbiguous is a link that matches more than one note. It is not
	// broken — it resolves to the nearest — but the result can change when
	// either note moves.
	CheckAmbiguous Check = "ambiguous"
	// CheckDangling is a link that matches no note. Expected while a note is
	// being written; worth reviewing afterwards.
	CheckDangling Check = "dangling"
)
