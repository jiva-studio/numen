package api

// ProblemKind names a defect found in a card model or a deck.
type ProblemKind string

const (
	ProblemKindUnspecified            ProblemKind = ""
	ProblemKindFieldDeclaredTwice     ProblemKind = "field_declared_twice"
	ProblemKindCardModelWithoutFields ProblemKind = "card_model_without_fields"
	ProblemKindTemplateMissingASide   ProblemKind = "template_missing_a_side"
	ProblemKindPlaceholderUndeclared  ProblemKind = "placeholder_undeclared"
	ProblemKindCardWithoutACardModel  ProblemKind = "card_without_a_card_model"
	ProblemKindCardModelIsNotOne      ProblemKind = "card_model_is_not_one"
	ProblemKindFieldWrittenTwice      ProblemKind = "field_written_twice"
	ProblemKindFieldNotRenamed        ProblemKind = "field_not_renamed"
	ProblemKindCardIdCarriedTwice     ProblemKind = "card_id_carried_twice"
)

// CardModel is a card model note: the fields its cards declare and the templates that lay them out.
type CardModel struct {
	Path      string          `json:"path"`
	Title     string          `json:"title"`
	Fields    []string        `json:"fields"`
	Templates []*CardTemplate `json:"templates"`
	Problems  []*Problem      `json:"problems"`
	Preamble  string          `json:"preamble"`
	Tail      string          `json:"tail"`
}

// CardModelSummary is the listing entry of a card model.
type CardModelSummary struct {
	Path   string   `json:"path"`
	Title  string   `json:"title"`
	Fields []string `json:"fields"`
}

// CardTemplate is one template of a card model; front and back are Markdown with `{{Field}}` placeholders.
type CardTemplate struct {
	Name     string `json:"name"`
	Front    string `json:"front"`
	Back     string `json:"back"`
	Preamble string `json:"preamble"`
}

// Deck is a deck note: its sections and its cards in file order.
type Deck struct {
	Path     string     `json:"path"`
	Title    string     `json:"title"`
	Preamble string     `json:"preamble"`
	Cards    []*Card    `json:"cards"`
	Sections []*Section `json:"sections"`
	Problems []*Problem `json:"problems"`
	Tail     string     `json:"tail"`
}

// Section is a heading of a deck that groups cards.
type Section struct {
	Name     string `json:"name"`
	Preamble string `json:"preamble"`
}

// Problem is a defect found in a card model or a deck. Card and Template are zero-based positions and are absent when the defect concerns neither.
type Problem struct {
	Kind     ProblemKind `json:"kind"`
	Card     *int32      `json:"card,omitempty"`
	Template *int32      `json:"template,omitempty"`
	Field    string      `json:"field"`
	Text     string      `json:"text"`
}

// Card is one card of a deck. SectionIndex is the zero-based index into the sections of the deck and is absent for a card outside any section.
type Card struct {
	CardID        string   `json:"cardId"`
	SectionIndex  *int32   `json:"sectionIndex,omitempty"`
	Heading       string   `json:"heading"`
	CardModelLink string   `json:"cardModelLink"`
	Preamble      string   `json:"preamble"`
	Values        []*Value `json:"values"`
	CardModelPath string   `json:"cardModelPath,omitempty"`
}

// Value is the text a card holds for one field of its card model.
type Value struct {
	Field string `json:"field"`
	Text  string `json:"text"`
}

// ListCardModelsRequest is the request of the ListCardModels method.
type ListCardModelsRequest struct {
	// Limit is the largest number of card models to return; a value of zero or less selects the maximum.
	Limit int32 `json:"limit"`
}

// ListCardModelsResponse is the response of the ListCardModels method.
type ListCardModelsResponse struct {
	CardModels []*CardModelSummary `json:"cardModels"`
	Total      int32               `json:"total"`
}

// CreateCardModelRequest is the request of the CreateCardModel method.
type CreateCardModelRequest struct {
	Title  string   `json:"title"`
	Path   string   `json:"path"`
	Fields []string `json:"fields"`
}

// CreateCardModelResponse is the response of the CreateCardModel method.
type CreateCardModelResponse struct {
	Path         string    `json:"path"`
	Error        ErrorCode `json:"error,omitempty"`
	IsUnlevelled bool      `json:"isUnlevelled"`
}

// ReadCardModelRequest is the request of the ReadCardModel method.
type ReadCardModelRequest struct {
	Path string `json:"path"`
}

// ReadCardModelResponse is the response of the ReadCardModel method.
type ReadCardModelResponse struct {
	CardModel *CardModel `json:"cardModel,omitempty"`
	Error     ErrorCode  `json:"error,omitempty"`
	At        *FileStat  `json:"at,omitempty"`
}

// WriteCardModelRequest is the request of the WriteCardModel method.
type WriteCardModelRequest struct {
	Path      string          `json:"path"`
	Fields    []string        `json:"fields"`
	Templates []*CardTemplate `json:"templates"`
	Baseline  *FileStat       `json:"baseline,omitempty"`
	Preamble  string          `json:"preamble"`
	Tail      string          `json:"tail"`
}

// WriteCardModelResponse is the response of the WriteCardModel method.
type WriteCardModelResponse struct {
	Error        ErrorCode `json:"error,omitempty"`
	At           *FileStat `json:"at,omitempty"`
	IsUnlevelled bool      `json:"isUnlevelled"`
}

// RenameCardModelFieldRequest is the request of the RenameCardModelField method.
type RenameCardModelFieldRequest struct {
	Path     string    `json:"path"`
	From     string    `json:"from"`
	To       string    `json:"to"`
	Baseline *FileStat `json:"baseline,omitempty"`
}

// RenameCardModelFieldResponse is the response of the RenameCardModelField method.
type RenameCardModelFieldResponse struct {
	Decks        []string         `json:"decks"`
	Cards        int32            `json:"cards"`
	NotWritten   []*UnwrittenDeck `json:"notWritten"`
	Error        ErrorCode        `json:"error,omitempty"`
	At           *FileStat        `json:"at,omitempty"`
	IsUnlevelled bool             `json:"isUnlevelled"`
}

// UnwrittenDeck is a deck that a field rename did not reach, with the problem that stopped it.
type UnwrittenDeck struct {
	Path    string   `json:"path"`
	Problem *Problem `json:"problem,omitempty"`
}

// CreateDeckRequest is the request of the CreateDeck method.
type CreateDeckRequest struct {
	Title string `json:"title"`
	Path  string `json:"path"`
}

// CreateDeckResponse is the response of the CreateDeck method.
type CreateDeckResponse struct {
	Path  string    `json:"path"`
	Error ErrorCode `json:"error,omitempty"`
	// IsUnlevelled reports that the file was written and the index did not follow.
	IsUnlevelled bool `json:"isUnlevelled"`
}

// ReadDeckRequest is the request of the ReadDeck method.
type ReadDeckRequest struct {
	Path string `json:"path"`
}

// ReadDeckResponse is the response of the ReadDeck method.
type ReadDeckResponse struct {
	Deck  *Deck     `json:"deck,omitempty"`
	Error ErrorCode `json:"error,omitempty"`
	At    *FileStat `json:"at,omitempty"`
	// Bound is the size limit in bytes that a deck failed against; zero for any other outcome.
	Bound int64 `json:"bound"`
}

// WriteDeckRequest is the request of the WriteDeck method.
type WriteDeckRequest struct {
	Path     string     `json:"path"`
	Preamble string     `json:"preamble"`
	Cards    []*Card    `json:"cards"`
	Sections []*Section `json:"sections"`
	Baseline *FileStat  `json:"baseline,omitempty"`
	Tail     string     `json:"tail"`
}

// WriteDeckResponse is the response of the WriteDeck method.
type WriteDeckResponse struct {
	Error ErrorCode `json:"error,omitempty"`
	At    *FileStat `json:"at,omitempty"`
	// Bound is the size limit in bytes that a deck failed against; zero for any other outcome.
	Bound        int64 `json:"bound"`
	IsUnlevelled bool  `json:"isUnlevelled"`
}
