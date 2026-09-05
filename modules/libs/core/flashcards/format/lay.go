package format

import "regexp"

// placeholderRe is `{{Field}}`, and the name inside is a field's name written
// exactly. There is no escape and no exemption: a code span holding the braces
// is filled like anything else. What holds no brace between the two pairs is a
// placeholder, so `{{a{b}}` is left as the text it is.
var placeholderRe = regexp.MustCompile(`\{\{([^{}]*)\}\}`)

// Lay fills a face of a stencil with one card: what stands before the answer
// and what stands after it, as HTML. The first field is placed by its own name
// and lays out the card's heading.
//
// A placeholder naming a field the card leaves out lays out as nothing, and
// everything around it is what the person wrote and arrives as they wrote it. A
// face with only one of its two sides lays out nothing at all.
func Lay(s Stencil, face FaceTemplate, card Card) (front, back string) {
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
