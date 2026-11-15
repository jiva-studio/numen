package searchquery

// The names the builtin catalog declares.
const (
	FunctionMeaning     = "meaning"
	FunctionWords       = "words"
	VariableKind        = "kind"
	VariablePath        = "path"
	VariableTitle       = "title"
	VariableNoteType    = "noteType"
	VariableModified    = "modified"
	VariableSize        = "size"
	FunctionDate        = "date"
	FunctionProperty    = "property"
	FunctionHasProperty = "hasProperty"
	FunctionHasHeading  = "hasHeading"
)

// Level is where a function decides: on one chunk or on a whole document.
type Level int

const (
	LevelChunk Level = iota
	LevelDocument
)

// ValueType is a type of the language.
type ValueType int

const (
	TypeBool ValueType = iota
	TypeString
	TypeInt
	TypeStringList
	TypeSearchResults
	TypeTimestamp
	// TypeDyn is the value of a frontmatter key: any YAML value.
	TypeDyn
)

// Function is one entry of the catalog: a global function, or a variable when IsVariable is set and Args
// is nil. A function with OptionalArgs also takes them after Args.
type Function struct {
	Name         string
	Args         []ValueType
	OptionalArgs []ValueType
	Result       ValueType
	Level        Level
	IsVariable   bool
	Description  string
	Example      string
}

// Catalog lists the functions and variables a query may name.
type Catalog struct {
	Functions []Function
}

// GetBuiltinCatalog lists words and meaning, which decide per chunk, and the functions and variables that decide
// per document: kind, path, title, noteType, modified, size, date, property, hasProperty, hasHeading and the link
// functions.
func GetBuiltinCatalog() Catalog {
	c := Catalog{Functions: []Function{
		{
			Name: FunctionWords, Args: []ValueType{TypeString}, Result: TypeSearchResults, Level: LevelChunk,
			Description: "passages whose words match an FTS5 expression: words, \"phrases\", prefix*, AND, OR, NOT, NEAR",
			Example:     `words("river NOT lake")`,
		},
		{
			Name: FunctionMeaning, Args: []ValueType{TypeString}, Result: TypeSearchResults, Level: LevelChunk,
			Description: "passages close in meaning to a text",
			Example:     `meaning("how a river changes its course")`,
		},
		{
			Name: VariableKind, Result: TypeString, Level: LevelDocument, IsVariable: true,
			Description: "the kind of the file: note, pdf, epub, audio, video, image or file",
			Example:     `kind == "pdf"`,
		},
		{
			Name: VariablePath, Result: TypeString, Level: LevelDocument, IsVariable: true,
			Description: "the path of the file in the vault, compared with startsWith",
			Example:     `path.startsWith("books/")`,
		},
		{
			Name: VariableTitle, Result: TypeString, Level: LevelDocument, IsVariable: true,
			Description: "the title of a document: the title of a note, else the file name without its extension; compared case-sensitively with ==, in, startsWith, endsWith or contains",
			Example:     `title.startsWith("Lecture")`,
		},
		{
			Name: VariableNoteType, Result: TypeString, Level: LevelDocument, IsVariable: true,
			Description: "the type key of a note, note when it has none; empty for a document that is not a note",
			Example:     `noteType == "book"`,
		},
		{
			Name: VariableModified, Result: TypeTimestamp, Level: LevelDocument, IsVariable: true,
			Description: "when the file was last changed, compared with date(…); == holds only at that instant, so compare with a range",
			Example:     `modified >= date("2026-01-01")`,
		},
		{
			Name: VariableSize, Result: TypeInt, Level: LevelDocument, IsVariable: true,
			Description: "the size of the file in bytes",
			Example:     `size > 1000000`,
		},
		{
			Name: FunctionDate, Args: []ValueType{TypeString}, Result: TypeTimestamp, Level: LevelDocument,
			Description: "an instant: YYYY-MM-DD or YYYY-MM-DDTHH:MM[:SS] with an optional Z or ±HH:MM; the local time of the machine when it has no offset",
			Example:     `modified < date("2026-06-01T12:00")`,
		},
		{
			Name: FunctionProperty, Args: []ValueType{TypeString}, Result: TypeDyn, Level: LevelDocument,
			Description: `the value of a top-level frontmatter key; a note without the key satisfies no comparison but !=, so write hasProperty("k") && property("k") != x; "x" in property("k") tests a list for an item`,
			Example:     `property("status") == "done"`,
		},
		{
			Name: FunctionHasProperty, Args: []ValueType{TypeString}, Result: TypeBool, Level: LevelDocument,
			Description: "true when the frontmatter of a note has the key, whatever its value",
			Example:     `hasProperty("source")`,
		},
		{
			Name: FunctionHasHeading, Args: []ValueType{TypeString}, Result: TypeBool, Level: LevelDocument,
			Description: "true when a note has a heading of exactly this text, ignoring case",
			Example:     `hasHeading("Summary")`,
		},
	}}
	c.Functions = append(c.Functions, getLinkCatalog()...)
	return c
}

func (c Catalog) FindFunction(name string) (Function, bool) {
	for _, f := range c.Functions {
		if f.Name == name {
			return f, true
		}
	}
	return Function{}, false
}
