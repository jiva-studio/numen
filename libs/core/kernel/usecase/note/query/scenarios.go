package query

// Scenarios is everything that answers a question about the notes of a vault, grouped; the container
// builds each use case on its own.
type Scenarios struct {
	Refer  Referring
	Browse Browsing
}

// Referring lists and resolves the links of notes.
type Referring struct {
	Links          ShowLinks
	Backlinks      GetBacklinks
	ResolveLinks   *ResolveLinks
	GetLinkAddress *GetLinkAddress
}

// Browsing answers questions about the notes of a vault.
type Browsing struct {
	Connection     GetConnection
	GetTitles      GetTitles
	GetTitle       *GetTitle
	IsVaultIndexed *IsVaultIndexed
}
