package cards

// Cut reads a deck against the stencils its cards name, and answers with what
// the two files say together and neither says alone. Stencils is keyed by what
// stands in a card's brackets, and a card naming one that is not in it is read
// against no stencil.
//
// The problems are against the deck: the deck is the file somebody would open
// to settle them.
func Cut(d Deck, stencils map[string]Stencil) []Problem {
	var out []Problem
	for at, card := range d.Cards {
		first := stencils[card.Stencil].First()
		if first == "" {
			continue
		}
		for _, v := range card.Values {
			if v.Field != first {
				continue
			}
			problem := against(at, CheckFirstFieldTwice,
				"this card writes "+first+" under a heading of its own as well as in the heading above, "+
					"and the heading above stands")
			problem.Field = first
			out = append(out, problem)
			break
		}
	}
	return out
}
