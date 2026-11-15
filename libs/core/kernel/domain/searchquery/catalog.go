package searchquery

// The names the builtin catalog declares.
const (
	FunctionMeaning = "meaning"
	FunctionWords   = "words"
	VariableKind    = "kind"
	VariablePath    = "path"
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

// GetBuiltinCatalog lists words and meaning, which decide per chunk, and the variables kind and path and the link
// functions, which decide per document.
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
