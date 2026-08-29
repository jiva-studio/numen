package domain

// VaultSummary is how much of a vault the index holds. A read model, shown to
// the user after a scan.
type VaultSummary struct {
	Notes    int
	Headings int
}
