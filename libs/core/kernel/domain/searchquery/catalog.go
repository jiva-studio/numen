package searchquery

import "slices"

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

// Function is one entry of the catalog: a global function, a variable when IsVariable is set and Args
// is nil, or a stage, a method of the results, when IsStage is set. A stage with several argument lists has
// them in Overloads. A function with OptionalArgs also takes them after Args. Summary says what it does in a
// few plain words; Description says it in one or two plain sentences that name the arguments. PluginID names
// the plugin that answers a function it contributes, empty for a built-in; ArgNames are the names of its
// arguments, in the order of Args.
type Function struct {
	Name         string
	Args         []ValueType
	OptionalArgs []ValueType
	Result       ValueType
	Level        Level
	IsVariable   bool
	IsStage      bool
	Overloads    [][]ValueType
	Summary      string
	Description  string
	Example      string
	PluginID     string
	ArgNames     []string
}

// Catalog lists the functions and variables a query may name.
type Catalog struct {
	Functions []Function
}

// GetBuiltinCatalog lists words and meaning, which decide per chunk, the functions and variables that decide per
// document: kind, path, title, noteType, modified, size, date, property, hasProperty, hasHeading and the link
// functions, and the stages rerank, top and sortBy.
func GetBuiltinCatalog() Catalog {
	return Catalog{Functions: slices.Concat(
		getTextFunctions(),
		getScopeVariables(),
		getStageFunctions(),
		getDocumentVariables(),
		getDocumentFunctions(),
		getLinkCatalog(),
	)}
}

func getTextFunctions() []Function {
	return []Function{
		{
			Name: FunctionWords, Args: []ValueType{TypeString}, Result: TypeSearchResults, Level: LevelChunk,
			Summary:     "passages with these words",
			Description: `Finds passages that contain the words you give. Put a phrase in quotes, end a word with * to match every word that starts with it, and join words with AND, OR, NOT and NEAR.`,
			Example:     `words("river NOT lake")`,
		},
		{
			Name: FunctionMeaning, Args: []ValueType{TypeString}, Result: TypeSearchResults, Level: LevelChunk,
			Summary:     "passages close in meaning",
			Description: "Finds passages that say something close to the text you give, even in other words.",
			Example:     `meaning("how a river changes its course")`,
		},
	}
}

func getScopeVariables() []Function {
	return []Function{
		{
			Name: VariableKind, Result: TypeString, Level: LevelDocument, IsVariable: true,
			Summary:     "the type of the file",
			Description: "The type of the file: note, pdf, epub, audio, video, image or file.",
			Example:     `kind == "pdf"`,
		},
		{
			Name: VariablePath, Result: TypeString, Level: LevelDocument, IsVariable: true,
			Summary:     "where the file lies",
			Description: `Where the file lies in the vault, such as books/rivers.pdf. Add .startsWith("books/") to keep the files of one folder.`,
			Example:     `path.startsWith("books/")`,
		},
	}
}

func getStageFunctions() []Function {
	return []Function{
		{
			Name: StageNameRerank, Result: TypeSearchResults, IsStage: true,
			Overloads: [][]ValueType{
				{}, {TypeString}, {TypeString, TypeInt}, {TypeString, TypeString}, {TypeString, TypeString, TypeInt},
			},
			Summary:     "reorder the best results with a reranking model",
			Description: "Reorders the best results with a reranking model, which reads each passage and judges how well it answers a question. Give the question (the text of the search if left out), an instruction for the model if you want one, and how many documents to reorder (the Rerank window setting if left out, 60 at first); the documents after those are dropped.",
			Example:     `meaning("heat").rerank("heat transfer", 100)`,
		},
		{
			Name: StageNameTop, Args: []ValueType{TypeInt}, Result: TypeSearchResults, IsStage: true,
			Summary:     "keep the first N",
			Description: "Keeps only the first results; give how many.",
			Example:     `words("tide").top(10)`,
		},
		{
			Name: StageNameSortBy, Result: TypeSearchResults, IsStage: true,
			Overloads:   [][]ValueType{{TypeString}, {TypeString, TypeString}},
			Summary:     "put the results in another order",
			Description: `Puts the results in order by score, path, title, modified or size. Score, modified and size go highest, newest or largest first, path and title from A to Z; add "asc" or "desc" to choose the direction.`,
			Example:     `(kind == "pdf").sortBy("modified")`,
		},
	}
}

func getDocumentVariables() []Function {
	return []Function{
		{
			Name: VariableTitle, Result: TypeString, Level: LevelDocument, IsVariable: true,
			Summary:     "the title of the note or file",
			Description: "the title of a document: the title of a note, else the file name without its extension; compared case-sensitively with ==, in, startsWith, endsWith or contains",
			Example:     `title.startsWith("Lecture")`,
		},
		{
			Name: VariableNoteType, Result: TypeString, Level: LevelDocument, IsVariable: true,
			Summary:     "the type of the note",
			Description: "the type key of a note, note when it has none; empty for a document that is not a note",
			Example:     `noteType == "book"`,
		},
		{
			Name: VariableModified, Result: TypeTimestamp, Level: LevelDocument, IsVariable: true,
			Summary:     "when the file was last changed",
			Description: "when the file was last changed, compared with date(…); == holds only at that instant, so compare with a range",
			Example:     `modified >= date("2026-01-01")`,
		},
		{
			Name: VariableSize, Result: TypeInt, Level: LevelDocument, IsVariable: true,
			Summary:     "the size of the file",
			Description: "the size of the file in bytes",
			Example:     `size > 1000000`,
		},
	}
}

func getDocumentFunctions() []Function {
	return []Function{
		{
			Name: FunctionDate, Args: []ValueType{TypeString}, Result: TypeTimestamp, Level: LevelDocument,
			Summary:     "a date and time",
			Description: "an instant: YYYY-MM-DD or YYYY-MM-DDTHH:MM[:SS] with an optional Z or ±HH:MM; the local time of the machine when it has no offset",
			Example:     `modified < date("2026-06-01T12:00")`,
		},
		{
			Name: FunctionProperty, Args: []ValueType{TypeString}, Result: TypeDyn, Level: LevelDocument,
			Summary:     "the value of a property",
			Description: `the value of a top-level frontmatter key; a note without the key satisfies no comparison but !=, so write hasProperty("k") && property("k") != x; "x" in property("k") tests a list for an item`,
			Example:     `property("status") == "done"`,
		},
		{
			Name: FunctionHasProperty, Args: []ValueType{TypeString}, Result: TypeBool, Level: LevelDocument,
			Summary:     "notes with this property",
			Description: "true when the frontmatter of a note has the key, whatever its value",
			Example:     `hasProperty("source")`,
		},
		{
			Name: FunctionHasHeading, Args: []ValueType{TypeString}, Result: TypeBool, Level: LevelDocument,
			Summary:     "notes with this heading",
			Description: "true when a note has a heading of exactly this text, ignoring case",
			Example:     `hasHeading("Summary")`,
		},
	}
}

func (c Catalog) FindFunction(name string) (Function, bool) {
	for _, f := range c.Functions {
		if f.Name == name {
			return f, true
		}
	}
	return Function{}, false
}
