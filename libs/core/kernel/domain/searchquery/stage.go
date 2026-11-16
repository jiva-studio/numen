package searchquery

// The names of the stages.
const (
	StageNameRerank = "rerank"
	StageNameTop    = "top"
	StageNameSortBy = "sortBy"
)

func IsStageName(name string) bool {
	return name == StageNameRerank || name == StageNameTop || name == StageNameSortBy
}

// StageKind is what a stage does.
type StageKind int

const (
	StageRerank StageKind = iota
	StageTop
	StageSortBy
)

// Stage is one method call on the results of a query, in the order written.
type Stage struct {
	Kind     StageKind
	Rerank   RerankStage
	Count    int
	Sort     SortStage
	Position Position
}

// RerankStage orders the results of the first Window documents by a reranking service. Window 0 takes the
// window of StageLimits.
type RerankStage struct {
	Query       string
	Instruction string
	Window      int
}

// RawStage is one stage as a parser emits it, its arguments literals.
type RawStage struct {
	Name     string
	Args     []RawArg
	Position Position
}

// RawArg is a literal argument of a stage: a string, or an int when IsInt is set.
type RawArg struct {
	Text     string
	Number   int64
	IsInt    bool
	Position Position
}

func (q Query) HasStage(kind StageKind) bool {
	for _, s := range q.Stages {
		if s.Kind == kind {
			return true
		}
	}
	return false
}
