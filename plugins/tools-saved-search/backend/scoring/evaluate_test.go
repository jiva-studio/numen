package scoring_test

import (
	"testing"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host"
	"github.com/jiva-studio/numen/plugins/tools-saved-search/backend/query"
	"github.com/jiva-studio/numen/plugins/tools-saved-search/backend/scoring"
)

const floor = 0.4

func similar(similarity float64) host.ConditionScore {
	return host.ConditionScore{Similarity: &similarity, IsMatch: similarity >= floor}
}

func matching(isMatch bool) host.ConditionScore {
	return host.ConditionScore{IsMatch: isMatch}
}

func evaluate(t *testing.T, text, path string, scores ...host.ConditionScore) scoring.Outcome {
	t.Helper()
	compiled, err := query.Compile(text)
	if err != nil {
		t.Fatalf("Compile(%q) = %v", text, err)
	}
	if len(scores) != len(compiled.Leaves) {
		t.Fatalf("%d scores for the %d leaves of %q", len(scores), len(compiled.Leaves), text)
	}
	return scoring.Evaluate(compiled.Root, compiled.Leaves, scoring.Scores(scores), floor, path)
}

func isScore(outcome scoring.Outcome, want float64) bool {
	return outcome.Score != nil && *outcome.Score == want
}

func TestMeaningMatchesASimilarityAtTheFloor(t *testing.T) {
	at := evaluate(t, `meaning("tides")`, "Notes/Tides.md", similar(0.4))
	below := evaluate(t, `meaning("tides")`, "Notes/Tides.md", similar(0.39))

	if !at.IsMatch || !isScore(at, 0.4) {
		t.Errorf("at the floor: %+v, want a match scored 0.4", at)
	}
	if below.IsMatch {
		t.Errorf("below the floor: %+v, want no match", below)
	}
}

func TestMeaningDoesNotHoldForAChunkWithoutAVector(t *testing.T) {
	outcome := evaluate(t, `meaning("tides")`, "Notes/Tides.md", host.ConditionScore{})

	if outcome.IsMatch || outcome.Score != nil {
		t.Errorf("outcome = %+v, want no match and no score", outcome)
	}
}

func TestNegatedMeaningDropsAChunkCloserToItThanToThePositiveLeafNodes(t *testing.T) {
	outcome := evaluate(t, `meaning("tides") && !meaning("lighthouses")`, "Notes/Keeper.md", similar(0.5), similar(0.8))

	if outcome.IsMatch {
		t.Errorf("outcome = %+v, want the chunk closer to lighthouses dropped", outcome)
	}
}

func TestNegatedMeaningKeepsAChunkCloserToThePositiveLeafNodes(t *testing.T) {
	outcome := evaluate(t, `meaning("tides") && !meaning("lighthouses")`, "Notes/Tides.md", similar(0.8), similar(0.5))

	if !outcome.IsMatch || !isScore(outcome, 0.8) {
		t.Errorf("outcome = %+v, want a match scored 0.8", outcome)
	}
}

func TestNegatedMeaningComparesWithEveryPositiveMeaningLeaf(t *testing.T) {
	outcome := evaluate(t, `(meaning("tides") || meaning("harbor")) && !meaning("lighthouses")`, "Notes/Harbor.md",
		similar(0.3), similar(0.9), similar(0.6))

	if !outcome.IsMatch || !isScore(outcome, 0.9) {
		t.Errorf("outcome = %+v, want a match: the chunk is closer to harbor than to lighthouses", outcome)
	}
}

func TestNegatedMeaningBelowTheFloorKeepsTheChunk(t *testing.T) {
	outcome := evaluate(t, `(meaning("tides") || words("harbor")) && !meaning("lighthouses")`, "Notes/Harbor.md",
		similar(0.2), matching(true), similar(0.35))

	if !outcome.IsMatch {
		t.Errorf("outcome = %+v, want a match: the similarity to lighthouses is below the floor", outcome)
	}
}

func TestNegatedMeaningWithoutAPositiveMeaningLeafDropsAChunkAtTheFloor(t *testing.T) {
	dropped := evaluate(t, `words("harbor") && !meaning("lighthouses")`, "Notes/Keeper.md", matching(true), similar(0.6))
	kept := evaluate(t, `words("harbor") && !meaning("lighthouses")`, "Notes/Harbor.md", matching(true), similar(0.3))

	if dropped.IsMatch || !kept.IsMatch {
		t.Errorf("at 0.6: %+v, at 0.3: %+v, want the first dropped and the second kept", dropped, kept)
	}
}

func TestNegatedWordsDropsAChunkThatMatches(t *testing.T) {
	dropped := evaluate(t, `meaning("tides") && !words("anchor")`, "Notes/Tides.md", similar(0.8), matching(true))
	kept := evaluate(t, `meaning("tides") && !words("anchor")`, "Notes/Tides.md", similar(0.8), matching(false))

	if dropped.IsMatch || !kept.IsMatch {
		t.Errorf("with the word: %+v, without: %+v, want the first dropped and the second kept", dropped, kept)
	}
}

func TestAndReturnsTheLowerScore(t *testing.T) {
	outcome := evaluate(t, `meaning("tides") && meaning("harbor")`, "Notes/Harbor.md", similar(0.8), similar(0.6))

	if !outcome.IsMatch || !isScore(outcome, 0.6) {
		t.Errorf("outcome = %+v, want a match scored 0.6", outcome)
	}
}

func TestAndFailsWhenOneChildFails(t *testing.T) {
	outcome := evaluate(t, `meaning("tides") && words("harbor")`, "Notes/Tides.md", similar(0.8), matching(false))

	if outcome.IsMatch {
		t.Errorf("outcome = %+v, want no match", outcome)
	}
}

func TestOrReturnsTheHigherScoreOfTheMatchingChildren(t *testing.T) {
	outcome := evaluate(t, `(meaning("tides") && words("ferry")) || meaning("storms")`, "Logs/Storm.md",
		similar(0.9), matching(false), similar(0.6))

	if !outcome.IsMatch || !isScore(outcome, 0.6) {
		t.Errorf("outcome = %+v, want a match scored 0.6, the score of the matching child", outcome)
	}
}

func TestOrReturnsTheHigherOfTwoMatchingScores(t *testing.T) {
	outcome := evaluate(t, `meaning("tides") || meaning("storms")`, "Logs/Storm.md", similar(0.5), similar(0.7))

	if !outcome.IsMatch || !isScore(outcome, 0.7) {
		t.Errorf("outcome = %+v, want a match scored 0.7", outcome)
	}
}

func TestWordsAloneReturnsNoScore(t *testing.T) {
	outcome := evaluate(t, `words("harbor")`, "Notes/Harbor.md", matching(true))

	if !outcome.IsMatch || outcome.Score != nil {
		t.Errorf("outcome = %+v, want a match with no score", outcome)
	}
}

func TestAndOfMeaningAndWordsReturnsTheScoreOfMeaning(t *testing.T) {
	outcome := evaluate(t, `words("harbor") && meaning("tides")`, "Notes/Harbor.md", matching(true), similar(0.7))

	if !outcome.IsMatch || !isScore(outcome, 0.7) {
		t.Errorf("outcome = %+v, want a match scored 0.7", outcome)
	}
}

func TestNegatedPathDropsAChunkInTheFolder(t *testing.T) {
	text := `meaning("tides") && !path.startsWith("Archive/")`
	inside := evaluate(t, text, "Archive/Old Tides.md", similar(0.8))
	outside := evaluate(t, text, "Notes/Tides.md", similar(0.8))

	if inside.IsMatch || !outside.IsMatch {
		t.Errorf("inside the folder: %+v, outside: %+v, want the first dropped and the second kept", inside, outside)
	}
}

func TestPathPrefixMatchesNFDAndNFCPaths(t *testing.T) {
	nfc, nfd := "Café/", "Café/"
	for _, pair := range [][2]string{{nfc, nfd + "Menu.md"}, {nfd, nfc + "Menu.md"}, {nfc, nfc + "Menu.md"}} {
		outcome := evaluate(t, `meaning("tides") && path.startsWith("`+pair[0]+`")`, pair[1], similar(0.8))

		if !outcome.IsMatch {
			t.Errorf("prefix %q and path %q: %+v, want a match", pair[0], pair[1], outcome)
		}
	}
}

func TestKindIsTheTypeOfTheFileByItsExtension(t *testing.T) {
	kinds := map[string][]string{
		"note":  {"Notes/Tides.md"},
		"pdf":   {"Atlases/Harbor.pdf"},
		"epub":  {"Books/Sea.epub"},
		"audio": {"Talks/a.mp3", "Talks/b.m4a", "Talks/c.wav", "Talks/d.ogg", "Talks/e.opus", "Talks/f.flac"},
		"video": {"Films/a.mp4", "Films/b.mkv", "Films/c.webm", "Films/d.mov"},
		"image": {"Charts/a.png", "Charts/b.jpg", "Charts/c.jpeg", "Charts/d.webp", "Charts/e.tiff"},
	}
	for kind, paths := range kinds {
		for _, path := range paths {
			for other := range kinds {
				outcome := evaluate(t, `meaning("tides") && kind == "`+other+`"`, path, similar(0.8))

				if outcome.IsMatch != (other == kind) {
					t.Errorf("kind == %q for %s: %+v, want a match only for %q", other, path, outcome, kind)
				}
			}
		}
	}
}

func TestKindOfAnotherFileIsNoneOfTheKinds(t *testing.T) {
	for _, kind := range []string{"note", "pdf", "epub", "audio", "video", "image"} {
		outcome := evaluate(t, `meaning("tides") && kind == "`+kind+`"`, "Data/tides.csv", similar(0.8))

		if outcome.IsMatch {
			t.Errorf("kind == %q for a csv file: %+v, want no match", kind, outcome)
		}
	}
}

func TestKindInAndNotEqualHoldForTheirKinds(t *testing.T) {
	in := evaluate(t, `meaning("tides") && kind in ["pdf", "epub"]`, "Books/Sea.epub", similar(0.8))
	notEqual := evaluate(t, `meaning("tides") && kind != "note"`, "Notes/Tides.md", similar(0.8))

	if !in.IsMatch || notEqual.IsMatch {
		t.Errorf("kind in for an epub: %+v, kind != note for a note: %+v, want a match and no match", in, notEqual)
	}
}
