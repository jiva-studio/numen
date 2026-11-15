package tokenizer_test

import "testing"

// Greedy longest match takes ▁abc and then d (−11); the best total is ▁ab and cd (−6).
func TestEncodeChoosesTheBestTotalScoreNotTheLongestPiece(t *testing.T) {
	checkEncode(t, loadFixture(t), "abcd", getSingleIDs(t, "▁ab", "cd"))
}

// ▁ x y and ▁ xy both total −14; the path whose last piece starts earliest wins.
func TestEncodeKeepsThePathWhoseLastPieceStartsEarliestOnEqualScores(t *testing.T) {
	checkEncode(t, loadFixture(t), "xy", getSingleIDs(t, "▁", "xy"))
}

// The lowest score of the fixture is −20, so an unknown rune costs −30. Neither g nor m is a piece.
// gh k (−29.5) beats unknown hk (−31): a penalty above −28.5 would choose the unknown.
// unknown np (−31) beats mn p (−31.5): a penalty below −30.5 would choose mn p.
func TestEncodeScoresAnUnknownRuneTenBelowTheLowestPiece(t *testing.T) {
	tok := loadFixture(t)
	checkEncode(t, tok, "ghk", getSingleIDs(t, "▁", "gh", "k"))
	checkEncode(t, tok, "mnp", getSingleIDs(t, "▁", "<unk>", "np"))
}

func TestEncodeFusesConsecutiveUnknownRunesIntoOneID(t *testing.T) {
	tok := loadFixture(t)
	checkEncode(t, tok, "ʘʘ", getSingleIDs(t, "▁", "<unk>"))
	checkEncode(t, tok, "ʘhkʘ", getSingleIDs(t, "▁", "<unk>", "hk", "<unk>"))
}
