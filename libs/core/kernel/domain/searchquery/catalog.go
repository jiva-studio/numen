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
// them in Overloads. Summary says what it does in a few plain words; Description says it in one or two plain
// sentences that name the arguments.
type Function struct {
	Name        string
	Args        []ValueType
	Result      ValueType
	Level       Level
	IsVariable  bool
	IsStage     bool
	Overloads   [][]ValueType
	Summary     string
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
