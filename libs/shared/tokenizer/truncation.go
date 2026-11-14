package tokenizer

func truncateLongestFirst(first, second, budget int) (keptFirst, keptSecond int) {
	if first+second <= budget {
		return first, second
	}
	shorter := min(first, second)
	longer := shorter
	if shorter <= budget {
		longer = max(shorter, budget-shorter)
	}
	if shorter+longer > budget {
		shorter = budget / 2
		longer = shorter + budget%2
	}
	if first > second {
		return longer, shorter
	}
	return shorter, longer
}
