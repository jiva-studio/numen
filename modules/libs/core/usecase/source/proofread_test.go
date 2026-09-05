package source

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/jiva-studio/numen/modules/libs/core/domain"
	"github.com/jiva-studio/numen/modules/libs/core/fixes"
	"github.com/jiva-studio/numen/modules/libs/core/proofread"
	"github.com/jiva-studio/numen/modules/libs/core/text"
)

// A corrector answers about a page with whatever it was told to say about it,
// and keeps the pages of every request it was given.
type corrector struct {
	name  string
	says  map[int]string
	asked [][]int
	// about is what each batch it was given said the text holds.
	about []string
	fail  error
	// stop is called with the number of requests made so far.
	stop func(int)
}

func (c *corrector) Name() string {
	if c.name == "" {
		return "a proofreader"
	}
	return c.name
}

func (c *corrector) Proofread(ctx context.Context, pages []proofread.Batch) (map[int]string, error) {
	var at []int
	for _, page := range pages {
		at = append(at, page.Number)
		c.about = append(c.about, page.Context)
	}
	c.asked = append(c.asked, at)
	if c.stop != nil {
		c.stop(len(c.asked))
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if c.fail != nil {
		return nil, c.fail
	}
	out := map[int]string{}
	for _, page := range pages {
		if said, has := c.says[page.Number]; has {
			out[page.Number] = said
		}
	}
	return out, nil
}

// proofreading is a reading of the fixture, already written down, and a
// Proofread over it.
func proofreading(t *testing.T, says map[int]string) (Proofread, domain.Vault, *shelf, *corrector) {
	t.Helper()
	read, v, _, shelved, _ := reading(t, "the words", "outline.pdf")
	if _, err := read.Execute(t.Context(), v, documentPath); err != nil {
		t.Fatal(err)
	}
	by := &corrector{says: says}
	return Proofread{
		Readers: read.Readers,
		Derived: shelved,
		By:      by,
		Pages:   2,
	}, v, shelved, by
}

// corrects is a reply putting one line right.
func corrects(at int, text string) string { return fmt.Sprintf("%d|%s", at, text) }

// TestNothingIsProofreadWhereNothingWasConfiguredToProofreadWith. An
// installation that named no profile has no proofreader, so nil is what the
// settings hand over and the constructor takes it without a word. A caller that
// forgot to refuse it first is answered, not brought down mid-reading.
func TestNothingIsProofreadWhereNothingWasConfiguredToProofreadWith(t *testing.T) {
	put, v, _, by := proofreading(t, nil)

	if _, err := NewProofread(put.Readers, put.Derived, nil).
		Execute(t.Context(), v, documentPath); !errors.Is(err, errNothingProofreads) {
		t.Fatalf("a reading was proofread with no proofreader: %v", err)
	}
	// The control: the same reading, through the same constructor, with a
	// proofreader.
	if _, err := NewProofread(put.Readers, put.Derived, by).
		Execute(t.Context(), v, documentPath); err != nil {
		t.Fatalf("a reading with a proofreader was refused: %v", err)
	}
}

func TestTheCorrectionsGoBesideTheReadingAndTheReadingIsNotTouched(t *testing.T) {
	put, v, shelved, _ := proofreading(t, map[int]string{
		0: corrects(0, "the WORDS 1"),
		2: corrects(2, "the WORDS 3"),
	})
	was := kept(t, shelved, textNames(t, shelved).artifact)

	res, err := put.Execute(t.Context(), v, documentPath)
	if err != nil {
		t.Fatal(err)
	}
	if res.Pages != 4 || res.Read != 4 || res.Fixed != 2 {
		t.Errorf("got %+v", res)
	}

	names := textNames(t, shelved)
	if string(kept(t, shelved, names.artifact)) != string(was) {
		t.Error("the reading was rewritten")
	}
	if put := fixes.Unpack(kept(t, shelved, names.fixes)); len(put) != 2 {
		t.Errorf("what is kept beside the reading is %+v", put)
	}
	var stood checkpoint
	if err := json.Unmarshal(kept(t, shelved, names.far), &stood); err != nil {
		t.Fatal(err)
	}
	if stood.By != "a proofreader" || stood.Pages != 4 {
		t.Errorf("got %+v", stood)
	}
}

func TestACorrectedReadingIsTheTextTheSourceIsCutFrom(t *testing.T) {
	put, v, shelved, _ := proofreading(t, map[int]string{0: corrects(0, "the WORDS 1")})

	if _, err := put.Execute(t.Context(), v, documentPath); err != nil {
		t.Fatal(err)
	}

	names := textNames(t, shelved)
	doc, err := text.Composed(t.Context(), shelved, "ocr", names.hash, kept(t, shelved, names.artifact))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(doc.Text, "the WORDS 1") {
		t.Errorf("the reading still says %q", doc.Text)
	}
}

func TestAPageWhoseReplyIsNoAnswerIsLeftAsItWasRead(t *testing.T) {
	put, v, shelved, _ := proofreading(t, map[int]string{
		0: corrects(0, "the "+proofread.Opens+"WORDS"+proofread.Closes+" 1"),
		1: corrects(1, "the WORDS 2"),
	})

	res, err := put.Execute(t.Context(), v, documentPath)
	if err != nil {
		t.Fatal(err)
	}
	if res.Refused != 1 || res.Fixed != 1 {
		t.Errorf("got %+v", res)
	}

	names := textNames(t, shelved)
	doc, err := text.Composed(t.Context(), shelved, "ocr", names.hash, kept(t, shelved, names.artifact))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(doc.Text, "the words 1") {
		t.Errorf("a page nothing answered about was changed: %q", doc.Text)
	}
}

func TestARunStoppedPartWayIsTakenUpAtThePageItStoppedOn(t *testing.T) {
	put, v, shelved, by := proofreading(t, map[int]string{
		0: corrects(0, "the WORDS 1"),
		2: corrects(2, "the WORDS 3"),
	})
	ctx, stop := context.WithCancel(t.Context())
	by.stop = func(requests int) {
		if requests == 2 {
			stop()
		}
	}

	if _, err := put.Execute(ctx, v, documentPath); !errors.Is(err, context.Canceled) {
		t.Fatalf("stopped with %v", err)
	}

	again := &corrector{says: map[int]string{2: corrects(2, "the WORDS 3")}}
	put.By = again
	res, err := put.Execute(t.Context(), v, documentPath)
	if err != nil {
		t.Fatal(err)
	}
	if res.Resumed != 2 {
		t.Errorf("took up %d pages of the reading", res.Resumed)
	}
	for _, asked := range again.asked {
		for _, page := range asked {
			if page < 2 {
				t.Errorf("asked about page %d again", page)
			}
		}
	}

	names := textNames(t, shelved)
	doc, err := text.Composed(t.Context(), shelved, "ocr", names.hash, kept(t, shelved, names.artifact))
	if err != nil {
		t.Fatal(err)
	}
	for _, said := range []string{"the WORDS 1", "the WORDS 3"} {
		if !strings.Contains(doc.Text, said) {
			t.Errorf("%q is not in %q", said, doc.Text)
		}
	}
}

func TestCorrectionsAnotherProofreaderMadeAreNotTakenUp(t *testing.T) {
	put, v, shelved, _ := proofreading(t, map[int]string{0: corrects(0, "the WORDS 1")})
	names := textNames(t, shelved)
	stood, err := json.Marshal(checkpoint{By: "somebody else", Pages: 3})
	if err != nil {
		t.Fatal(err)
	}
	if err := shelved.Write(t.Context(), names.far, stood); err != nil {
		t.Fatal(err)
	}
	if err := shelved.Write(t.Context(), names.fixes, []byte("what another one said")); err != nil {
		t.Fatal(err)
	}

	res, err := put.Execute(t.Context(), v, documentPath)
	if err != nil {
		t.Fatal(err)
	}
	if res.Resumed != 0 {
		t.Errorf("took up %d pages another proofreader had", res.Resumed)
	}
	doc, err := text.Composed(t.Context(), shelved, "ocr", names.hash, kept(t, shelved, names.artifact))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(doc.Text, "the WORDS 1") {
		t.Errorf("the reading says %q", doc.Text)
	}
}

func TestOneRunToAReading(t *testing.T) {
	put, v, shelved, by := proofreading(t, nil)
	shelved.hold(textNames(t, shelved).fixes)

	res, err := put.Execute(t.Context(), v, documentPath)
	if err != nil {
		t.Fatal(err)
	}
	if !res.Busy {
		t.Error("two runs to one reading")
	}
	if len(by.asked) != 0 {
		t.Errorf("asked about %v", by.asked)
	}
}

func TestAReadingThatIsNotThereIsNothingToProofread(t *testing.T) {
	put, v, shelved, _ := proofreading(t, nil)
	names := textNames(t, shelved)
	if err := shelved.Remove(t.Context(), names.artifact); err != nil {
		t.Fatal(err)
	}

	res, err := put.Execute(t.Context(), v, documentPath)
	if err != nil {
		t.Fatal(err)
	}
	if !res.None {
		t.Errorf("got %+v", res)
	}
}

func TestASourceIsCutAgainAsItsPagesArePutRight(t *testing.T) {
	put, v, _, _ := proofreading(t, map[int]string{0: corrects(0, "the WORDS 1")})
	cuts := 0
	put.Cut = func(context.Context, domain.Vault, string) error {
		cuts++
		return nil
	}

	if _, err := put.Execute(t.Context(), v, documentPath); err != nil {
		t.Fatal(err)
	}
	// Four pages, two to a request.
	if cuts != 2 {
		t.Errorf("cut %d times", cuts)
	}
}

func TestABatchThatWasNotCutIsAskedAboutAgain(t *testing.T) {
	put, v, shelved, by := proofreading(t, map[int]string{
		0: corrects(0, "the WORDS 1"),
		2: corrects(2, "the WORDS 3"),
	})
	cuts := 0
	put.Cut = func(context.Context, domain.Vault, string) error {
		cuts++
		if cuts == 1 {
			return errors.New("the index would not take it")
		}
		return nil
	}

	if _, err := put.Execute(t.Context(), v, documentPath); err == nil {
		t.Fatal("a cut that failed was taken as a batch done")
	}
	if _, err := shelved.Read(t.Context(), textNames(t, shelved).far); err == nil {
		t.Error("a batch nothing cut was counted")
	}

	by.asked = nil
	if _, err := put.Execute(t.Context(), v, documentPath); err != nil {
		t.Fatal(err)
	}
	// Four pages, two to a request, and the first two asked about again.
	if len(by.asked) != 2 || len(by.asked[0]) == 0 || by.asked[0][0] != 0 {
		t.Errorf("the run after it asked about %v", by.asked)
	}
}

// names are what one reading of the fixture is kept under.
type names struct {
	hash     string
	artifact string
	fixes    string
	far      string
}

// textNames finds the one reading on the shelf and says what its files are
// called.
func textNames(t *testing.T, shelved *shelf) names {
	t.Helper()
	for _, name := range shelved.names() {
		if !strings.HasSuffix(name, ".txt") {
			continue
		}
		hash := strings.TrimSuffix(strings.TrimPrefix(name, "ocr/"), ".txt")
		return names{
			hash:     hash,
			artifact: text.Artifact("ocr", hash),
			fixes:    text.Fixes("ocr", hash),
			far:      text.Proofread("ocr", hash),
		}
	}
	t.Fatal("nothing was read")
	return names{}
}

func kept(t *testing.T, shelved *shelf, name string) []byte {
	t.Helper()
	raw, err := shelved.Read(t.Context(), name)
	if err != nil {
		t.Fatal(err)
	}
	return raw
}

// A proofreadQueue takes a run of pages away and answers about them when it is
// told it is ready.
type proofreadQueue struct {
	*corrector
	left  map[string][]int
	ready map[string]bool
	// gone is a batch the proofreader will never answer about.
	gone bool
}

func leaving(says map[int]string) *proofreadQueue {
	return &proofreadQueue{
		corrector: &corrector{says: says},
		left:      map[string][]int{},
		ready:     map[string]bool{},
	}
}

func (q *proofreadQueue) Leave(_ context.Context, pages []proofread.Batch) (string, error) {
	var at []int
	for _, page := range pages {
		at = append(at, page.Number)
	}
	name := fmt.Sprintf("batch-%d", len(q.left)+1)
	q.left[name] = at
	return name, nil
}

func (q *proofreadQueue) Collect(_ context.Context, name string) (map[int]string, bool, error) {
	if q.gone {
		return nil, false, errors.New("the proofreader has forgotten this batch")
	}
	if !q.ready[name] {
		return nil, false, nil
	}
	out := map[int]string{}
	for _, at := range q.left[name] {
		if said, has := q.says[at]; has {
			out[at] = said
		}
	}
	return out, true, nil
}

// stands is what the reading says about who put it right and how far they got.
func stands(t *testing.T, shelved *shelf) checkpoint {
	t.Helper()
	var stood checkpoint
	if err := json.Unmarshal(kept(t, shelved, textNames(t, shelved).far), &stood); err != nil {
		t.Fatal(err)
	}
	return stood
}

func TestPagesAreLeftForTheProofreaderAndCollectedByAnotherRun(t *testing.T) {
	put, v, shelved, _ := proofreading(t, nil)
	left := leaving(map[int]string{0: corrects(0, "the WORDS 1")})
	put.Queue = left
	put.By = left

	res, err := put.Execute(t.Context(), v, documentPath)
	if err != nil {
		t.Fatal(err)
	}
	if !res.Waiting || res.Fixed != 0 {
		t.Errorf("got %+v", res)
	}
	stood := stands(t, shelved)
	if stood.Batch != "batch-1" || stood.Left != 2 || stood.Pages != 0 {
		t.Fatalf("the reading stands at %+v", stood)
	}

	// Nothing is collected until the proofreader has answered.
	res, err = put.Execute(t.Context(), v, documentPath)
	if err != nil {
		t.Fatal(err)
	}
	if !res.Waiting || res.Fixed != 0 {
		t.Errorf("got %+v", res)
	}

	left.ready["batch-1"] = true
	res, err = put.Execute(t.Context(), v, documentPath)
	if err != nil {
		t.Fatal(err)
	}
	if res.Fixed != 1 || res.Read != 2 || !res.Waiting {
		t.Errorf("got %+v", res)
	}
	if stood := stands(t, shelved); stood.Pages != 2 || stood.Batch != "batch-2" {
		t.Errorf("the reading stands at %+v", stood)
	}

	names := textNames(t, shelved)
	doc, err := text.Composed(t.Context(), shelved, "ocr", names.hash, kept(t, shelved, names.artifact))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(doc.Text, "the WORDS 1") {
		t.Errorf("the reading says %q", doc.Text)
	}
}

func TestABatchTheProofreaderHasForgottenIsLeftAgain(t *testing.T) {
	put, v, shelved, _ := proofreading(t, nil)
	left := leaving(nil)
	put.Queue = left
	put.By = left

	if _, err := put.Execute(t.Context(), v, documentPath); err != nil {
		t.Fatal(err)
	}
	left.gone = true
	if _, err := put.Execute(t.Context(), v, documentPath); err == nil {
		t.Fatal("a batch nobody will answer about was waited for")
	}
	if stood := stands(t, shelved); stood.Batch != "" {
		t.Errorf("the reading still waits on %+v", stood)
	}

	left.gone = false
	if _, err := put.Execute(t.Context(), v, documentPath); err != nil {
		t.Fatal(err)
	}
	if same := left.left["batch-2"]; len(same) != 2 || same[0] != 0 {
		t.Errorf("the pages left again are %v", same)
	}
}

// A reading nothing is left to be asked about is not work, and nothing is told
// about it.
func TestAReadingAtItsLastPageReportsNoProgress(t *testing.T) {
	put, v, _, _ := proofreading(t, map[int]string{0: corrects(0, "the WORDS 1")})
	if _, err := put.Execute(t.Context(), v, documentPath); err != nil {
		t.Fatal(err)
	}

	told := 0
	put.By = &corrector{}
	put.OnProgress = func(ProofreadResult) { told++ }
	res, err := put.Execute(t.Context(), v, documentPath)
	if err != nil {
		t.Fatal(err)
	}
	if res.Read != res.Pages {
		t.Fatalf("got %+v", res)
	}
	if told != 0 {
		t.Errorf("a reading with nothing left to put right was told about %d times", told)
	}
}
