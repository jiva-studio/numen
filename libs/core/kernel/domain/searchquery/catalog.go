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

// Function is one entry of the catalog: a global function, a variable when IsVariable is set and Args
// is nil, or a stage, a method of the results, when IsStage is set. A stage with several argument lists has
// them in Overloads.
type Function struct {
	Name        string
	Args        []ValueType
	Result      ValueType
	Level       Level
	IsVariable  bool
	IsStage     bool
	Overloads   [][]ValueType
	Description string
	Example     string
}

// Catalog lists the functions and variables a query may name.
type Catalog struct {
	Functions []Function
}

// GetBuiltinCatalog lists words and meaning, which decide per chunk, the variables kind and path, which decide
// per document, and the stages rerank, top and sortBy.
func GetBuiltinCatalog() Catalog {
	return Catalog{Functions: []Function{
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
			Name: StageNameRerank, Result: TypeSearchResults, IsStage: true,
			Overloads: [][]ValueType{
				{}, {TypeString}, {TypeString, TypeInt}, {TypeString, TypeString}, {TypeString, TypeString, TypeInt},
			},
			Description: "orders the results by a reranking service; the window is the number of documents it takes, 60 by default; results of later documents are dropped",
			Example:     `meaning("heat").rerank("heat transfer", 100)`,
		},
		{
			Name: StageNameTop, Args: []ValueType{TypeInt}, Result: TypeSearchResults, IsStage: true,
			Description: "keeps the first N results",
			Example:     `words("tide").top(10)`,
		},
		{
			Name: StageNameSortBy, Result: TypeSearchResults, IsStage: true,
			Overloads:   [][]ValueType{{TypeString}, {TypeString, TypeString}},
			Description: "orders the results by score, path, title, modified or size; modified and size newest or largest first",
			Example:     `(kind == "pdf").sortBy("modified")`,
		},
	}}
}

func (c Catalog) FindFunction(name string) (Function, bool) {
	for _, f := range c.Functions {
		if f.Name == name {
			return f, true
		}
	}
	return Function{}, false
}
