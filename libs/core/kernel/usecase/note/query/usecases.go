package query

// UseCases is everything that answers a question about the notes of a vault, grouped; the container
// builds each use case on its own.
type UseCases struct {
	Refer  LinkQueries
	Browse BrowseQueries
}

// LinkQueries lists and resolves the links of notes.
type LinkQueries struct {
	Links        ShowLinks
	Backlinks    ListBacklinks
	ResolveLinks *ResolveLinks
}

// BrowseQueries answers questions about the notes of a vault.
type BrowseQueries struct {
	Connection GetConnection
}
