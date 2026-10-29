package format

import "regexp"

// placeholderRe is `{{Field}}`, and the name inside is a field's name written
// exactly. There is no escape and no exemption: a code span holding the braces
// is filled like anything else. What holds no brace between the two pairs is a
// placeholder, so `{{a{b}}` is left as the text it is.
var placeholderRe = regexp.MustCompile(`\{\{([^{}]*)\}\}`)

// Lay fills a template of a card model with one card: the content before the
// answer and the content after it, as HTML. The first field is placed by its own
// name and lays out the card's heading.
//
// A placeholder naming a field the card omits lays out as empty text, and
// everything around it is the user's text, unchanged. A
// template with only one of its two sides lays out nothing at all.
func Lay(template CardTemplate, card Card) (front, back string) {
	if template.Front == "" || template.Back == "" {
		return "", ""
	}
	return fillPlaceholders(template.Front, card), fillPlaceholders(template.Back, card)
}

func fillPlaceholders(template string, card Card) string {
	return placeholderRe.ReplaceAllStringFunc(template, func(match string) string {
		name := placeholderRe.FindStringSubmatch(match)[1]
		value, _ := card.Value(name)
		return value
	})
}

// placeholders is every name a template places, in the order it places them.
func placeholders(template string) []string {
	var out []string
	// gomutants:disable-next-line INTEGER_INCREMENT reason="the mutated value is -2: any negative count returns every match"
	for _, m := range placeholderRe.FindAllStringSubmatch(template, -1) {
		out = append(out, m[1])
	}
	return out
}
