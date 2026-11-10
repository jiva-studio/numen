package chunking

import "testing"

func TestAWordIsSpelledWhenOnlyItsEndsCarryMarks(t *testing.T) {
	spelled := []string{
		"", "x", "hello", "Hello", "azAZ09", "don't", "well-known", "e.g.", "3.14",
		"(hello)", `"hi"`, "hello,", "((a-b))", "#a-b", "a-b#", "(3.5)",
		"...", "--", "'", "#", "a", "z", "A", "Z", "0", "9",
		"naïve", "café", "«salut»", "“quoted”", "héllo…", "héllo!!!", "¤¤", "¤é",
		"日本。語", "日本、語", "ё-ё", "l’été", "l‑été", "é'é", "é.é", "é-é", "é1é",
		"٣-٣", "e\u0301e", "é\u0301é", "(é-é)", "[é.é]", "!!é", "¤é¤", "日本\uff0c語",
		"\u2061¤",
	}
	for _, token := range spelled {
		if !isSpelled(token) {
			t.Errorf("isSpelled(%q) = false", token)
		}
	}
	misspelled := []string{
		"a#b", "a_b", "a,b", "a@b", "a!b", "a?b", "a/b", "a:b", "a`b", "a{b", "a[b",
		"(a#b)", "3#4", "#a#b", "a#b#", "ab#c", "a#bc", "((a#b))", "a#a", "z#z", "A#A", "Z#Z", "0#0", "9#9",
		"wo¤rda", "naï#ve", "é¤é", "(é¤é)", "(٣¤٣)", "é_é", "é#é", "é,é", "x\u2061y",
		"日本\u2fff語", "日本\u3040語", "日本\ufeff語", "日本\ufff0語", "é\ufffdé",
		"é/é", "é:é", "é`é", "é{é", "é@é", "é[é", "é(é",
	}
	for _, token := range misspelled {
		if isSpelled(token) {
			t.Errorf("isSpelled(%q) = true", token)
		}
	}
}

func TestIsSpelled_ASCIIWordAllowsOnlyJoiners(t *testing.T) {
	for _, joiner := range []string{"'", "-", "."} {
		if !isSpelled("a" + joiner + "b") {
			t.Errorf("isSpelled(a%sb) = false", joiner)
		}
		if !isSpelled("1" + joiner + "2") {
			t.Errorf("isSpelled(1%s2) = false", joiner)
		}
	}
	for _, other := range []string{"_", ",", ";", ":", "!", "?", "/", "\\", "`", "@", "[", "{", "|", "~", "$", "%", "&", "*", "+", "="} {
		if isSpelled("a" + other + "b") {
			t.Errorf("isSpelled(a%sb) = true", other)
		}
	}
}

func TestACJKPunctuationBlocksAreTheTwoUnicodeBlocksOfThatName(t *testing.T) {
	inside := []rune{0x3000, 0x3001, 0x3002, 0x303E, 0x303F, 0xFF00, 0xFF01, 0xFF0C, 0xFFEE, 0xFFEF}
	outside := []rune{0x2FFF, 0x3040, 0xFEFF, 0xFFF0, 0x2000, 'a', '.', 0x4E2D, 0x10000}
	for _, r := range inside {
		if !isCJKPunctuation(r) {
			t.Errorf("isCJKPunctuation(%U) = false", r)
		}
	}
	for _, r := range outside {
		if isCJKPunctuation(r) {
			t.Errorf("isCJKPunctuation(%U) = true", r)
		}
	}
}

func TestAnInnerRuneIsAllowedWhenItIsPartOfAWord(t *testing.T) {
	allowed := []rune{'a', 'é', '日', 0x0301, '5', '٣', '\'', '’', '-', '‑', '.', '。', '\uff0c', 0x3000}
	invalid := []rune{'#', '_', ',', '¤', '§', 0x2061, 0xFFFD, '"', '“', '…'}
	for _, r := range allowed {
		if !isAllowedInside(r) {
			t.Errorf("isAllowedInside(%U) = false", r)
		}
	}
	for _, r := range invalid {
		if isAllowedInside(r) {
			t.Errorf("isAllowedInside(%U) = true", r)
		}
	}
}

func TestALetterOrDigitIsOneByTheUnicodeTables(t *testing.T) {
	yes := []rune{'a', 'z', 'A', 'Z', '0', '9', 'é', 'ё', '日', '٣', 0x0905}
	no := []rune{'`', '{', '@', '[', '/', ':', ' ', '-', 0x2061, 0x0301, '¤', 0x161 - 0x100 + 0x2000}
	for _, r := range yes {
		if !isLetterOrDigit(r) {
			t.Errorf("isLetterOrDigit(%U) = false", r)
		}
	}
	for _, r := range no {
		if isLetterOrDigit(r) {
			t.Errorf("isLetterOrDigit(%U) = true", r)
		}
	}
}
