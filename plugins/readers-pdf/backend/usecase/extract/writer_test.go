package extract_test

import (
	"os"
	"strings"
	"testing"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/domain"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/hosttest"
	"github.com/jiva-studio/numen/plugins/readers-pdf/backend/internal/testsupport"
	"github.com/jiva-studio/numen/plugins/readers-pdf/backend/pdf"
	"github.com/jiva-studio/numen/plugins/readers-pdf/backend/textlayer"
	"github.com/jiva-studio/numen/plugins/readers-pdf/backend/usecase/extract"
)

const pluginID = testsupport.PluginID

// TestMain makes the folder the pools of the tests keep the compiled PDFium in.
func TestMain(m *testing.M) {
	os.Exit(testsupport.Run(m))
}

func TestTheTextLayerOfAFileIsItsLiveVersion(t *testing.T) {
	texts, writer, raw := testsupport.NewVault(t, "books/tiny.pdf", "tiny.pdf")

	testsupport.CreateText(t, writer, extract.Request{Path: "books/tiny.pdf"})

	text := testsupport.ReadLive(t, texts, "books/tiny.pdf")
	if text.Version().Kind != extract.TextKind {
		t.Errorf("kind %q, want %q", text.Version().Kind, extract.TextKind)
	}
	if text.Version().InputHash != domain.HashContent(raw) || !text.Version().IsFromFile() {
		t.Errorf("version %+v is not made from the file as it is", text.Version())
	}
	if want := "Alpha beta gamma\nDelta epsilon zeta\n"; text.Text != want {
		t.Errorf("text %q, want %q", text.Text, want)
	}
	if got := text.GetSelector(strings.Index(text.Text, "epsilon")); !strings.HasPrefix(string(got), `{"page":2,`) {
		t.Errorf("epsilon has selector %s, want page 2", got)
	}
}

func TestARunOfTheLayerIsLitOverTheWordsItCovers(t *testing.T) {
	texts, writer, _ := testsupport.NewVault(t, "tiny.pdf", "tiny.pdf")
	testsupport.CreateText(t, writer, extract.Request{Path: "tiny.pdf"})
	text := testsupport.ReadLive(t, texts, "tiny.pdf")

	from := strings.Index(text.Text, "Delta")
	boxes := textlayer.GetBoxes(text, pdf.ByteRange{Start: from, End: from + len("Delta epsilon")})

	if len(boxes) != 2 {
		t.Fatalf("%d boxes, want one a word of the two: %+v", len(boxes), boxes)
	}
	for _, box := range boxes {
		if box.Page != 1 {
			t.Errorf("a box on page %d, want 1", box.Page)
		}
		if box.MaxX <= box.MinX || box.MaxY <= box.MinY {
			t.Errorf("box %+v covers nothing", box.Rect)
		}
	}
}

func TestTheOutlineOpensTheSectionsOfTheLayer(t *testing.T) {
	texts, writer, raw := testsupport.NewVault(t, "outline.pdf", "outline.pdf")
	testsupport.CreateText(t, writer, extract.Request{Path: "outline.pdf"})
	book, err := testsupport.NewWorkers(t).Read(raw)
	if err != nil {
		t.Fatal(err)
	}

	sections := testsupport.ReadLive(t, texts, "outline.pdf").GetHeadings()

	if len(sections) != len(book.Parts) {
		t.Fatalf("%d sections, want the %d the outline names", len(sections), len(book.Parts))
	}
	for i, part := range book.Parts {
		if sections[i].Title != part.Title || sections[i].Offset != part.Offset {
			t.Errorf("section %d is %q at %d, want %q at %d", i, sections[i].Title, sections[i].Offset, part.Title, part.Offset)
		}
	}
}

func TestAFileWithNoTextLayerGetsNoVersion(t *testing.T) {
	for name, raw := range map[string][]byte{
		"scan.pdf":      nil,
		"corrupted.pdf": []byte("INVALID_PDF_HEADER"),
	} {
		t.Run(name, func(t *testing.T) {
			var texts *hosttest.Texts
			var writer *extract.Writer
			if raw == nil {
				texts, writer, _ = testsupport.NewVault(t, name, name)
			} else {
				texts, writer, _ = testsupport.NewVaultOf(t, name, raw)
			}

			testsupport.CreateText(t, writer, extract.Request{Path: name})

			listing, err := texts.ListVersions(t.Context(), name)
			if err != nil {
				t.Fatal(err)
			}
			if len(listing.Versions) != 0 {
				t.Errorf("versions %v, want none", listing.Versions)
			}
		})
	}
}

func TestTheTextLayerIsRecordedInTheVaultTheRequestNames(t *testing.T) {
	raw := testsupport.ReadFixture(t, "tiny.pdf")
	physics := hosttest.New("Physics")
	chemistry := hosttest.New("Chemistry")
	physics.Beside(chemistry)
	chemistry.File("tiny.pdf", string(raw))
	texts := hosttest.NewTexts("Physics", pluginID)
	other := texts.With("Chemistry")
	other.File("tiny.pdf", raw)
	other.SetStore(pluginID, chemistry.Store())
	writer := testsupport.NewWriter(t, testsupport.FilesOf(physics), physics, texts, hosttest.NewQueue())

	testsupport.CreateText(t, writer, extract.Request{Vault: "Chemistry", Path: "tiny.pdf"})

	if text := testsupport.ReadLive(t, other, "tiny.pdf"); text.Version().Kind != extract.TextKind {
		t.Errorf("Chemistry's live version is %+v", text.Version())
	}
	inShown, err := texts.ListVersions(t.Context(), "tiny.pdf")
	if err != nil {
		t.Fatal(err)
	}
	if len(inShown.Versions) != 0 {
		t.Errorf("the shown vault holds %+v, want no version", inShown.Versions)
	}
}

func TestCreateTextWritesTheLayerInATurnOfTheQueue(t *testing.T) {
	raw := testsupport.ReadFixture(t, "tiny.pdf")
	files := hosttest.OnDisk(t, "vault", map[string]string{"tiny.pdf": string(raw)})
	texts := hosttest.NewTexts("vault", pluginID)
	texts.File("tiny.pdf", raw)
	queue := hosttest.NewQueue()
	held, err := queue.EnqueueWork(t.Context(), host.Work{})
	if err != nil {
		t.Fatal(err)
	}
	texts.SetStore(pluginID, files.Store())
	writer := testsupport.NewWriter(t, testsupport.FilesOf(files), files, texts, queue)

	if err := writer.CreateText(t.Context(), extract.Request{Path: "tiny.pdf"}); err != nil {
		t.Fatalf("CreateText: %v", err)
	}
	listing, err := texts.ListVersions(t.Context(), "tiny.pdf")
	if err != nil {
		t.Fatal(err)
	}
	if len(listing.Versions) != 0 {
		t.Fatalf("versions %v while the queue is held, want none", listing.Versions)
	}
	if err := held.Release(); err != nil {
		t.Fatal(err)
	}
	writer.Wait()

	listing, err = texts.ListVersions(t.Context(), "tiny.pdf")
	if err != nil {
		t.Fatal(err)
	}
	if len(listing.Versions) != 1 {
		t.Errorf("versions %v after the turn, want the text layer", listing.Versions)
	}
	if tasks := queue.ListTasks(); len(tasks) != 0 {
		t.Errorf("tasks %v after the layer was written, want none", tasks)
	}
}

func TestTheWriterIgnoresARepeatedAndALateRequest(t *testing.T) {
	raw := testsupport.ReadFixture(t, "tiny.pdf")
	files := hosttest.OnDisk(t, "vault", map[string]string{"tiny.pdf": string(raw)})
	texts := hosttest.NewTexts("vault", pluginID)
	texts.File("tiny.pdf", raw)
	queue := hosttest.NewQueue()
	held, err := queue.EnqueueWork(t.Context(), host.Work{})
	if err != nil {
		t.Fatal(err)
	}
	texts.SetStore(pluginID, files.Store())
	writer := testsupport.NewWriter(t, testsupport.FilesOf(files), files, texts, queue)

	if err := writer.CreateText(t.Context(), extract.Request{Path: "tiny.pdf"}); err != nil {
		t.Fatal(err)
	}
	// Duplicate call while running is a no-op
	if err := writer.CreateText(t.Context(), extract.Request{Path: "tiny.pdf"}); err != nil {
		t.Fatal(err)
	}
	// Close cancels running task and unlocks
	writer.Close()
	if err := held.Release(); err != nil {
		t.Error(err)
	}

	// CreateText after Close is a no-op
	if err := writer.CreateText(t.Context(), extract.Request{Path: "tiny.pdf"}); err != nil {
		t.Fatal(err)
	}
}
