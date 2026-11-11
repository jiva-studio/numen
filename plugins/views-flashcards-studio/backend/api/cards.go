package api

// ProblemKind names a defect found in a card model or a deck.
type ProblemKind string

// ProblemKind values.
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
	ProblemKindCardIDCarriedTwice     ProblemKind = "card_id_carried_twice"
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

type ListCardModelsRequest struct {
	Limit int32 `json:"limit"`
}

type ListCardModelsResponse struct {
	CardModels []*CardModelSummary `json:"cardModels"`
	Total      int32               `json:"total"`
}

type CreateCardModelRequest struct {
	Title  string   `json:"title"`
	Path   string   `json:"path"`
	Fields []string `json:"fields"`
}

type CreateCardModelResponse struct {
	Path            string    `json:"path"`
	Error           ErrorCode `json:"error,omitempty"`
	IsIndexOutdated bool      `json:"isIndexOutdated"`
}

type ReadCardModelRequest struct {
	Path string `json:"path"`
}

type ReadCardModelResponse struct {
	CardModel *CardModel `json:"cardModel,omitempty"`
	Error     ErrorCode  `json:"error,omitempty"`
	At        *FileStat  `json:"at,omitempty"`
}

type WriteCardModelRequest struct {
	Path      string          `json:"path"`
	Fields    []string        `json:"fields"`
	Templates []*CardTemplate `json:"templates"`
	Baseline  *FileStat       `json:"baseline,omitempty"`
	Preamble  string          `json:"preamble"`
	Tail      string          `json:"tail"`
}

type WriteCardModelResponse struct {
	Error           ErrorCode `json:"error,omitempty"`
	At              *FileStat `json:"at,omitempty"`
	IsIndexOutdated bool      `json:"isIndexOutdated"`
}

type RenameCardModelFieldRequest struct {
	Path     string    `json:"path"`
	From     string    `json:"from"`
	To       string    `json:"to"`
	Baseline *FileStat `json:"baseline,omitempty"`
}

type RenameCardModelFieldResponse struct {
	Decks           []string         `json:"decks"`
	Cards           int32            `json:"cards"`
	NotWritten      []*UnwrittenDeck `json:"notWritten"`
	Error           ErrorCode        `json:"error,omitempty"`
	At              *FileStat        `json:"at,omitempty"`
	IsIndexOutdated bool             `json:"isIndexOutdated"`
}

// UnwrittenDeck is a deck that a field rename did not reach, with the problem that stopped it.
type UnwrittenDeck struct {
	Path    string   `json:"path"`
	Problem *Problem `json:"problem,omitempty"`
}

type CreateDeckRequest struct {
	Title string `json:"title"`
	Path  string `json:"path"`
}

type CreateDeckResponse struct {
	Path            string    `json:"path"`
	Error           ErrorCode `json:"error,omitempty"`
	IsIndexOutdated bool      `json:"isIndexOutdated"`
}

type ReadDeckRequest struct {
	Path string `json:"path"`
}

type ReadDeckResponse struct {
	Deck  *Deck     `json:"deck,omitempty"`
	Error ErrorCode `json:"error,omitempty"`
	At    *FileStat `json:"at,omitempty"`
	Bound int64     `json:"bound"`
}

type WriteDeckRequest struct {
	Path     string     `json:"path"`
	Preamble string     `json:"preamble"`
	Cards    []*Card    `json:"cards"`
	Sections []*Section `json:"sections"`
	Baseline *FileStat  `json:"baseline,omitempty"`
	Tail     string     `json:"tail"`
}

type WriteDeckResponse struct {
	Error           ErrorCode `json:"error,omitempty"`
	At              *FileStat `json:"at,omitempty"`
	Bound           int64     `json:"bound"`
	IsIndexOutdated bool      `json:"isIndexOutdated"`
}
