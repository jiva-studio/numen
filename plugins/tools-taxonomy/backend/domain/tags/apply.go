package tags

// Apply returns the tags of a note after the toggles: a toggled tag the note has is removed, and one it has not is
// appended, in the order of the toggles. Tags are compared as GetKey compares them, the tags the note keeps stay in its
// order and in their spelling, and each toggle is cleaned as Clean does.
func Apply(current []string, toggles []string) []string {
	return nil
}
