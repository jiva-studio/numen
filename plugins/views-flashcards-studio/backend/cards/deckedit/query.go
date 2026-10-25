package deckedit

// Query is the card of Mark, or Limit cards from the From-th, counted from zero. Fields keeps those values alone.
type Query struct {
	Path   string
	From   int
	Limit  int
	Mark   string
	Fields []string
}
