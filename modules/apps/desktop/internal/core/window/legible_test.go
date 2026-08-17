package window

import "testing"

func TestLegible(t *testing.T) {
	tests := []struct {
		name string
		text string
		want bool
	}{
		{
			name: "latin prose",
			text: latin,
			want: true,
		},
		{
			name: "cyrillic prose",
			text: cyrillic,
			want: true,
		},
		{
			name: "transliterated sanskrit",
			text: sanskrit,
			want: true,
		},
		{
			name: "sanskrit written with combining marks",
			text: "cañcalaṁ hi manaḥ kr̥ṣṇa pramāthi balavad dr̥ḍham",
			want: true,
		},
		{
			name: "a citation is spelled precisely",
			text: "As Bhagavad-gītā 2.13 says, the soul is not slain when the body is.",
			want: true,
		},
		{
			name: "a quoted line in three scripts",
			text: latin + " " + cyrillic + " " + sanskrit,
			want: true,
		},
		{
			name: "a greek epigraph recognition lost",
			text: noise,
			want: false,
		},
		{
			name: "punctuation with letters in it",
			text: "thes;e wor|ds are;n't rea;dable an;ymore",
			want: false,
		},
		{
			name: "punctuation alone",
			text: "·  ., ;; ·· .,",
			want: false,
		},
		{
			name: "nothing",
			text: "   ",
			want: false,
		},
	}

	sizes := Sizes{}.resolve()
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := legible(test.text, sizes); got != test.want {
				t.Errorf("legible(%q) = %v, want %v", test.text, got, test.want)
			}
		})
	}
}

// TestLegibleThresholdsAreConfiguration is the rule that where the thresholds sit
// depends on the corpus: the same text passes one setting and fails another.
func TestLegibleThresholdsAreConfiguration(t *testing.T) {
	text := "thes;e wor|ds are;n't rea;dable an;ymore"

	if legible(text, Sizes{Dirty: 0.25}.resolve()) {
		t.Error("a quarter of the words may carry a mark inside, and this text is past that")
	}
	if !legible(text, Sizes{Dirty: 1}.resolve()) {
		t.Error("a threshold that admits the text rejected it")
	}
	if legible(latin, Sizes{Alphabetic: 0.99}.resolve()) {
		t.Error("prose carries punctuation, and a floor of 0.99 excludes it")
	}
	if !legible(noise, Sizes{Alphabetic: -1, Dirty: 1}.resolve()) {
		t.Error("thresholds asked for nothing and still rejected a window")
	}
}
