package wordform

import "testing"

func TestGetStem(t *testing.T) {
	for word, want := range map[string]string{
		"dom":         "",
		"voda":        "vod",
		"pizda":       "pizd",
		"knigi":       "knig",
		"knigami":     "kniga",
		"Книга":       "Книг",
		"книгами":     "книга",
		"krishna":     "krish",
		"Kṛṣṇa":       "Kṛṣṇ",
		"abcde\u0301": "abcd",
	} {
		if got := GetStem(word); got != want {
			t.Errorf("GetStem(%q) = %q, want %q", word, got, want)
		}
	}
}

func TestFoldRune(t *testing.T) {
	for _, c := range []struct{ in, want rune }{
		{'A', 'a'}, {'Й', 'и'}, {'Ё', 'е'}, {'ṛ', 'r'}, {'ṣ', 's'}, {'ṇ', 'n'}, {'ā', 'a'}, {'я', 'я'}, {'日', '日'},
	} {
		if got := FoldRune(c.in); got != c.want {
			t.Errorf("FoldRune(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}
