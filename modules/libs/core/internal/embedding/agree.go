package embedding

// Agreement is how near two vectors stand for the models that made them to be
// one model. The same weights answer the same text with the same vector, and
// the distance left is what arithmetic on two machines leaves.
const Agreement = 0.99

// Asked is the text two embedders are compared over. It carries several
// scripts, so two models that differ only outside one alphabet differ here.
const Asked = "Zoë Brontë — Марроуфилд — the allotment gate"

// Agreed says whether two vectors of one text came from one model.
//
// A question embedded in another space finds nothing the first indexed. Two
// providers name a model by whatever each of them calls it, so what they
// answer is compared instead.
func Agreed(a, b []float32) bool {
	if len(a) == 0 || len(a) != len(b) {
		return false
	}
	var sum float64
	for i := range a {
		sum += float64(a[i]) * float64(b[i])
	}
	return sum >= Agreement
}
