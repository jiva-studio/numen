package chunking

import "testing"

// A CJK paragraph carries no spaces, so it is one word from end to end. The
// sentence punctuation inside it is not OCR noise, and the defaults must not
// refuse it as such.
func TestACJKParagraphWithSentencePunctuationIsClean(t *testing.T) {
	paragraph := "这是一个测试。这是另一个句子，用于验证全角标点！这样的文本真的可以通过吗？「当然可以」『没问题』、很好〜"

	defaults := Legibility{}.resolve()
	if !isLegible(paragraph, defaults) {
		t.Errorf("%q failed by the thresholds", paragraph)
	}
}

// Mixed scripts with a high ratio of unrelated symbols are what the dirty
// threshold exists to catch, and a wider isAllowedInside must not let that
// kind of token through.
func TestMixedScriptNoiseStaysDirty(t *testing.T) {
	noisy := "wo¤rda wo§rdb wo¥rdc wo¦rdd wo¨rde wo¤rdf wo§rdg wo¥rdh"

	defaults := Legibility{}.resolve()
	if isLegible(noisy, defaults) {
		t.Errorf("%q passed the thresholds", noisy)
	}
}

// A negative threshold asks for none, for both of them. A corpus written in a
// script these fractions were not measured on is indexed whole.
func TestANegativeThresholdAsksForNone(t *testing.T) {
	rubbish := "|| $$ ?? %%% ### @@@ ~~~ ^^^"
	sampleText := "the quick brown fox jumps over the lazy dog"

	retained := Legibility{Alphabetic: -1, Dirty: -1}
	for _, text := range []string{rubbish, sampleText} {
		if !isLegible(text, retained) {
			t.Errorf("with no threshold, %q failed", text)
		}
	}

	// The thresholds as they stand still refuse what they were written for.
	defaults := Legibility{}.resolve()
	if isLegible(rubbish, defaults) {
		t.Errorf("%q passed the thresholds", rubbish)
	}
	if !isLegible(sampleText, defaults) {
		t.Errorf("%q failed by the thresholds", sampleText)
	}
}
