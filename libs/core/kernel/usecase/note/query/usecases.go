package query

// UseCases is everything that answers a question about the notes of a vault, grouped; the container
// builds each use case on its own.
type UseCases struct {
	Refer  Referring
	Browse Browsing
}

// Referring lists and resolves the links of notes.
type Referring struct {
	Links        ShowLinks
	Backlinks    ListBacklinks
	ResolveLinks *ResolveLinks
}

// Browsing answers questions about the notes of a vault.
type Browsing struct {
	Connection GetConnection
}
