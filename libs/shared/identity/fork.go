package identity

// Fork is where a chain replaces the entries of another chain of the same account: ID is the hash of the first entry that the chain has
// in their place, and Voided is how many entries of the other chain, from there on, do not count any more.
type Fork struct {
	ID     [32]byte
	Voided int
}

// FindFork reports where c replaces other, as IsPreferredOver decides it, and is false when c only extends other or other wins.
func (c Chain) FindFork(other Chain) (Fork, bool) {
	if !c.IsPreferredOver(other) {
		return Fork{}, false
	}
	at := c.getForkIndex(other)
	return Fork{ID: hashEntry(c.entries[at]), Voided: len(other.entries) - at}, true
}
