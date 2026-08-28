package cards

import "regexp"

// placeholderRe is `{{Field}}`, and the name inside is a field's name written
// exactly. Nothing escapes the braces: wherever the two characters stand on a
// face, a code span included, what runs to the next `}}` is a placeholder, and
// a face wanting those characters as text has no way to write them.
var placeholderRe = regexp.MustCompile(`\{\{([^{}]*)\}\}`)

// Lay fills a face of a stencil with one card: what stands before the answer
// and what stands after it, as markdown. The first field is placed by its own
// name and lays out the card's heading.
//
// A placeholder naming a field the card leaves out lays out as nothing, and
// everything around it is the person's markdown and arrives as they wrote it.
// A face with only one of its two sides lays out nothing at all.
func Lay(s Stencil, face Face, card Card) (front, back string) {
	if face.Front == "" || face.Back == "" {
		return "", ""
	}
	return fill(face.Front, card), fill(face.Back, card)
}

func fill(face string, card Card) string {
	return placeholderRe.ReplaceAllStringFunc(face, func(match string) string {
		name := placeholderRe.FindStringSubmatch(match)[1]
		value, _ := card.Value(name)
		return value
	})
}

// placeholders is every name a face places, in the order it places them.
func placeholders(face string) []string {
	var out []string
	for _, m := range placeholderRe.FindAllStringSubmatch(face, -1) {
		out = append(out, m[1])
	}
	return out
}
