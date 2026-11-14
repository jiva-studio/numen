package tokenizer

func truncateLongestFirst(first, second, budget int) (keptFirst, keptSecond int) {
	if first+second <= budget {
		return first, second
	}
	isSwapped := first > second
	shorter, longer := first, second
	if isSwapped {
		shorter, longer = second, first
	}
	if shorter > budget {
		longer = shorter
	} else {
		longer = max(shorter, budget-shorter)
	}
	if shorter+longer > budget {
		shorter = budget / 2
		longer = shorter + budget%2
	}
	if isSwapped {
		return longer, shorter
	}
	return shorter, longer
}
