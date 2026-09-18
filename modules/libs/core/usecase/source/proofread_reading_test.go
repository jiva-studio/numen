package source

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/jiva-studio/numen/modules/libs/core/domain"
	"github.com/jiva-studio/numen/modules/libs/core/internal/correction"
	"github.com/jiva-studio/numen/modules/libs/core/internal/text"
	"github.com/jiva-studio/numen/modules/libs/core/port"
	"github.com/jiva-studio/numen/modules/libs/core/proofread"
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

func (c *corrector) GetName() string {
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

// proofreadable is a reading of the fixture, already written down, and what a
// ProofreadReading over it is made out of.
type proofreadable struct {
	readers port.VaultReaders
	derived port.DerivedStores
	vault   domain.Vault
	shelved *shelf
}

// put is a proofreading of that reading, by the proofreader given.
func (p proofreadable) put(t *testing.T, by port.Proofreader) ProofreadReading {
	t.Helper()
	made, err := NewProofreadReading(p.readers, p.derived, by)
	if err != nil {
		t.Fatal(err)
	}
	made.Pages = 2
	return made
}

// newProofreadReading is that reading, a proofreading of it, and the
// proofreader the proofreading was made with.
func newProofreadReading(t *testing.T, says map[int]string) (ProofreadReading, domain.Vault, *shelf, *corrector) {
	t.Helper()
	held := newProofreadable(t)
	by := &corrector{says: says}
	return held.put(t, by), held.vault, held.shelved, by
}

// newProofreadable is the fixture's reading, written down and ready to be put
// right.
func newProofreadable(t *testing.T) proofreadable {
	t.Helper()
	read, v, _, shelved, _ := reading(t, "the words", outline)
	if _, err := read.Execute(t.Context(), v, documentPath); err != nil {
		t.Fatal(err)
	}
	return proofreadable{readers: read.Readers, derived: shelves{shelved}, vault: v, shelved: shelved}
}

// corrects is a reply putting one line right.
func corrects(at int, text string) string { return fmt.Sprintf("%d|%s", at, text) }

// A proofreading with nothing to proofread with is refused where it would be
// made, so no such thing exists to be called. An installation that named no
// profile has no proofreader at all, so this is a value the settings produce
// and not a caller's slip.
func TestNoProofreadingIsMadeWithNothingToProofreadWith(t *testing.T) {
	held := newProofreadable(t)

	for name, one := range map[string]struct {
		readers port.VaultReaders
		derived port.DerivedStores
		by      port.Proofreader
		want    error
	}{
		"nothing to proofread with": {held.readers, held.derived, nil, errNothingProofreads},
		"no vault":                  {nil, held.derived, &corrector{}, errNoVaultToProofread},
		"no store":                  {held.readers, nil, &corrector{}, errNoStoreToProofread},
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := NewProofreadReading(one.readers, one.derived, one.by); !errors.Is(err, one.want) {
				t.Fatalf("made with %v", err)
			}
		})
	}

	// The control: all three, and it is made.
	if _, err := NewProofreadReading(held.readers, held.derived, &corrector{}); err != nil {
		t.Fatalf("a proofreading with everything it needs was refused: %v", err)
	}
}

func TestTheCorrectionsGoBesideTheReadingAndTheReadingIsNotTouched(t *testing.T) {
	put, v, shelved, _ := newProofreadReading(t, map[int]string{
		0: corrects(0, "the WORDS 1"),
		2: corrects(2, "the WORDS 3"),
	})
	was := readShelf(t, shelved, textNames(t, shelved).artifact)

	res, err := put.Execute(t.Context(), v, documentPath)
	if err != nil {
		t.Fatal(err)
	}
	if res.Pages != 4 || res.Read != 4 || res.Fixed != 2 {
		t.Errorf("got %+v", res)
	}

	names := textNames(t, shelved)
	if string(readShelf(t, shelved, names.artifact)) != string(was) {
		t.Error("the reading was rewritten")
	}
	if put := correction.Unpack(readShelf(t, shelved, names.corrections)); len(put) != 2 {
		t.Errorf("what is kept beside the reading is %+v", put)
	}
	var stood checkpoint
	if err := json.Unmarshal(readShelf(t, shelved, names.far), &stood); err != nil {
		t.Fatal(err)
	}
	if stood.By != "a proofreader" || stood.Pages != 4 {
		t.Errorf("got %+v", stood)
	}
}

func TestACorrectedReadingIsTheTextTheSourceIsCutFrom(t *testing.T) {
	put, v, shelved, _ := newProofreadReading(t, map[int]string{0: corrects(0, "the WORDS 1")})

	if _, err := put.Execute(t.Context(), v, documentPath); err != nil {
		t.Fatal(err)
	}

	names := textNames(t, shelved)
	doc, err := text.ReadComposed(t.Context(), shelved, "ocr", names.hash, readShelf(t, shelved, names.artifact))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(doc.Text, "the WORDS 1") {
		t.Errorf("the reading still says %q", doc.Text)
	}
}

func TestAPageWhoseReplyIsNoAnswerIsLeftAsItWasRead(t *testing.T) {
	put, v, shelved, _ := newProofreadReading(t, map[int]string{
		0: corrects(0, "the "+proofread.Opens+"WORDS"+proofread.Closes+" 1"),
		1: corrects(1, "the WORDS 2"),
	})

	res, err := put.Execute(t.Context(), v, documentPath)
	if err != nil {
		t.Fatal(err)
	}
	if res.UncorrectedPages != 1 || res.Fixed != 1 {
		t.Errorf("got %+v", res)
	}

	names := textNames(t, shelved)
	doc, err := text.ReadComposed(t.Context(), shelved, "ocr", names.hash, readShelf(t, shelved, names.artifact))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(doc.Text, "the words 1") {
		t.Errorf("a page nothing answered about was changed: %q", doc.Text)
	}
}

func TestARunStoppedPartWayIsTakenUpAtThePageItStoppedOn(t *testing.T) {
	held := newProofreadable(t)
	by := &corrector{says: map[int]string{
		0: corrects(0, "the WORDS 1"),
		2: corrects(2, "the WORDS 3"),
	}}
	put, v, shelved := held.put(t, by), held.vault, held.shelved
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
	res, err := held.put(t, again).Execute(t.Context(), v, documentPath)
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
	doc, err := text.ReadComposed(t.Context(), shelved, "ocr", names.hash, readShelf(t, shelved, names.artifact))
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
	put, v, shelved, _ := newProofreadReading(t, map[int]string{0: corrects(0, "the WORDS 1")})
	names := textNames(t, shelved)
	stood, err := json.Marshal(checkpoint{By: "somebody else", Pages: 3})
	if err != nil {
		t.Fatal(err)
	}
	if err := shelved.Write(t.Context(), names.far, stood); err != nil {
		t.Fatal(err)
	}
	if err := shelved.Write(t.Context(), names.corrections, []byte("what another one said")); err != nil {
		t.Fatal(err)
	}

	res, err := put.Execute(t.Context(), v, documentPath)
	if err != nil {
		t.Fatal(err)
	}
	if res.Resumed != 0 {
		t.Errorf("took up %d pages another proofreader had", res.Resumed)
	}
	doc, err := text.ReadComposed(t.Context(), shelved, "ocr", names.hash, readShelf(t, shelved, names.artifact))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(doc.Text, "the WORDS 1") {
		t.Errorf("the reading says %q", doc.Text)
	}
}

func TestOneRunToAReading(t *testing.T) {
	put, v, shelved, by := newProofreadReading(t, nil)
	shelved.hold(textNames(t, shelved).corrections)

	res, err := put.Execute(t.Context(), v, documentPath)
	if err != nil {
		t.Fatal(err)
	}
	if !res.IsBusy {
		t.Error("two runs to one reading")
	}
	if len(by.asked) != 0 {
		t.Errorf("asked about %v", by.asked)
	}
}

func TestAReadingThatIsNotThereIsNothingToProofread(t *testing.T) {
	put, v, shelved, _ := newProofreadReading(t, nil)
	names := textNames(t, shelved)
	if err := shelved.Remove(t.Context(), names.artifact); err != nil {
		t.Fatal(err)
	}

	res, err := put.Execute(t.Context(), v, documentPath)
	if err != nil {
		t.Fatal(err)
	}
	if !res.IsNone {
		t.Errorf("got %+v", res)
	}
}

func TestASourceIsCutAgainAsItsPagesArePutRight(t *testing.T) {
	put, v, _, _ := newProofreadReading(t, map[int]string{0: corrects(0, "the WORDS 1")})
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
	put, v, shelved, by := newProofreadReading(t, map[int]string{
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
	hash        string
	artifact    string
	corrections string
	far         string
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
			hash:        hash,
			artifact:    text.Artifact("ocr", hash),
			corrections: text.Corrections("ocr", hash),
			far:         text.Proofread("ocr", hash),
		}
	}
	t.Fatal("nothing was read")
	return names{}
}

func readShelf(t *testing.T, shelved *shelf, name string) []byte {
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
	// isGone is a batch the proofreader will never answer about.
	isGone bool
}

func newProofreadQueue(says map[int]string) *proofreadQueue {
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
	if q.isGone {
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

// getCheckpoint is what the reading says about who put it right and how far
// they got.
func getCheckpoint(t *testing.T, shelved *shelf) checkpoint {
	t.Helper()
	var stood checkpoint
	if err := json.Unmarshal(readShelf(t, shelved, textNames(t, shelved).far), &stood); err != nil {
		t.Fatal(err)
	}
	return stood
}

func TestPagesAreLeftForTheProofreaderAndCollectedByAnotherRun(t *testing.T) {
	held := newProofreadable(t)
	left := newProofreadQueue(map[int]string{0: corrects(0, "the WORDS 1")})
	put, v, shelved := held.put(t, left), held.vault, held.shelved
	put.Queue = left

	res, err := put.Execute(t.Context(), v, documentPath)
	if err != nil {
		t.Fatal(err)
	}
	if !res.IsWaiting || res.Fixed != 0 {
		t.Errorf("got %+v", res)
	}
	stood := getCheckpoint(t, shelved)
	if stood.Batch != "batch-1" || stood.Left != 2 || stood.Pages != 0 {
		t.Fatalf("the reading stands at %+v", stood)
	}

	// Nothing is collected until the proofreader has answered.
	res, err = put.Execute(t.Context(), v, documentPath)
	if err != nil {
		t.Fatal(err)
	}
	if !res.IsWaiting || res.Fixed != 0 {
		t.Errorf("got %+v", res)
	}

	left.ready["batch-1"] = true
	res, err = put.Execute(t.Context(), v, documentPath)
	if err != nil {
		t.Fatal(err)
	}
	if res.Fixed != 1 || res.Read != 2 || !res.IsWaiting {
		t.Errorf("got %+v", res)
	}
	if stood := getCheckpoint(t, shelved); stood.Pages != 2 || stood.Batch != "batch-2" {
		t.Errorf("the reading stands at %+v", stood)
	}

	names := textNames(t, shelved)
	doc, err := text.ReadComposed(t.Context(), shelved, "ocr", names.hash, readShelf(t, shelved, names.artifact))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(doc.Text, "the WORDS 1") {
		t.Errorf("the reading says %q", doc.Text)
	}
}

func TestABatchTheProofreaderHasForgottenIsLeftAgain(t *testing.T) {
	held := newProofreadable(t)
	left := newProofreadQueue(nil)
	put, v, shelved := held.put(t, left), held.vault, held.shelved
	put.Queue = left

	if _, err := put.Execute(t.Context(), v, documentPath); err != nil {
		t.Fatal(err)
	}
	left.isGone = true
	if _, err := put.Execute(t.Context(), v, documentPath); err == nil {
		t.Fatal("a batch nobody will answer about was waited for")
	}
	if stood := getCheckpoint(t, shelved); stood.Batch != "" {
		t.Errorf("the reading still waits on %+v", stood)
	}

	left.isGone = false
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
	held := newProofreadable(t)
	v := held.vault
	first := held.put(t, &corrector{says: map[int]string{0: corrects(0, "the WORDS 1")}})
	if _, err := first.Execute(t.Context(), v, documentPath); err != nil {
		t.Fatal(err)
	}

	told := 0
	put := held.put(t, &corrector{})
	put.OnProgress = func(ProofreadReadingResult) { told++ }
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
