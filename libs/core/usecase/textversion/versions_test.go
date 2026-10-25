package textversion_test

import (
	"errors"
	"slices"
	"strings"
	"testing"

	"github.com/jiva-studio/numen/libs/notes/domain"

	"github.com/jiva-studio/numen/libs/core/port/textport"
	"github.com/jiva-studio/numen/libs/core/usecase/textversion"
)

const (
	pdfReader   = "org.numen.readers-pdf"
	ocr         = "org.numen.engines-ocr"
	proofreader = "org.numen.tools-proofreading"
	scanPath    = "books/scan.pdf"
)

// scene is one vault holding one PDF, and the three plugins of the worked example:
// a reader of the PDF's own text layer that runs on its own, a recogniser, and a
// proofreader of text in any format.
type scene struct {
	vault     domain.Vault
	versions  *textversion.Versions
	plugins   *plugins
	listener  *listener
	reader    *fakeProvider
	ocr       *fakeProvider
	fixer     *fakeProvider
	files     map[string][]byte
	repo      *memoryVersions
	providers providers
}

func newScene(t *testing.T) *scene {
	t.Helper()
	s := &scene{
		vault:    domain.Vault{ID: "v1", Name: "v1"},
		listener: &listener{},
		files:    map[string][]byte{scanPath: []byte("%PDF scan, first printing")},
		repo:     &memoryVersions{},
		plugins: &plugins{all: []textport.TextPlugin{
			{ID: ocr, Name: "OCR", Files: domain.FileInput{".pdf"}},
			{ID: pdfReader, Name: "PDF", Files: domain.FileInput{".pdf"}, IsAutoRun: true},
			{ID: proofreader, Name: "Proofreading", Formats: domain.TextFormats()},
		}},
	}
	page := func(word string, n int) domain.TextPart {
		return domain.TextPart{Text: word + " ", Place: domain.TextPlace{Page: n, Rects: []domain.Rect{{MaxX: 1, MaxY: 1}}}}
	}
	s.reader = &fakeProvider{pluginID: pdfReader, format: domain.TextFormatPositioned, files: s.files,
		make: func(string, domain.VersionText, bool) []domain.TextPart {
			return []domain.TextPart{page("layer", 0), page("teh", 1)}
		}}
	s.ocr = &fakeProvider{pluginID: ocr, format: domain.TextFormatPositioned, files: s.files,
		make: func(string, domain.VersionText, bool) []domain.TextPart {
			return []domain.TextPart{page("scanned", 0), page("teh", 1)}
		}}
	s.fixer = &fakeProvider{pluginID: proofreader, format: domain.TextFormatPositioned, files: s.files,
		make: func(_ string, input domain.VersionText, _ bool) []domain.TextPart {
			return []domain.TextPart{{Text: strings.ReplaceAll(input.Text, "teh", "the")}}
		}}
	s.providers = providers{pdfReader: s.reader, ocr: s.ocr, proofreader: s.fixer}
	versions, err := textversion.NewVersions(s.repo, s.plugins, s.providers, steppingClock())
	if err != nil {
		t.Fatal(err)
	}
	s.versions = versions
	versions.SetListener(s.listener)
	for _, p := range []*fakeProvider{s.reader, s.ocr, s.fixer} {
		p.versions, p.vault = versions, s.vault
	}
	return s
}

func (s *scene) fileHash() string { return domain.HashContent(s.files[scanPath]) }

// requireLive says which version is live and what its text says.
func (s *scene) requireLive(t *testing.T, step string, wantID domain.TextVersionID, wantText string) {
	t.Helper()
	listing, err := s.versions.ListVersions(t.Context(), s.vault, scanPath, s.fileHash())
	if err != nil {
		t.Fatalf("%s: %v", step, err)
	}
	if listing.Live != wantID {
		t.Fatalf("%s: the live version is %q, want %q", step, listing.Live, wantID)
	}
	text, err := s.versions.GetLiveText(t.Context(), s.vault, scanPath, s.fileHash())
	if err != nil {
		t.Fatalf("%s: read the live text: %v", step, err)
	}
	if text.Text != wantText {
		t.Fatalf("%s: the live text is %q, want %q", step, text.Text, wantText)
	}
}

// runEntry runs the menu entry of the action by the plugin, and fails where the menu offers none.
func (s *scene) runEntry(t *testing.T, action textversion.MenuAction, pluginID string) {
	t.Helper()
	menu, err := s.versions.GetMenu(t.Context(), s.vault, scanPath, s.fileHash())
	if err != nil {
		t.Fatal(err)
	}
	i := slices.IndexFunc(menu, func(e textversion.MenuEntry) bool { return e.Action == action && e.PluginID == pluginID })
	if i < 0 {
		t.Fatalf("the menu offers no %s by %s: %+v", action, pluginID, menu)
	}
	if err := s.versions.RunMenuEntry(t.Context(), s.vault, scanPath, menu[i]); err != nil {
		t.Fatalf("%s by %s: %v", action, pluginID, err)
	}
}

func (s *scene) setEnabled(t *testing.T, pluginID string, isEnabled bool) {
	t.Helper()
	s.plugins.setEnabled(pluginID, isEnabled)
	if err := s.versions.ReportPluginChanged(t.Context(), s.vault, pluginID); err != nil {
		t.Fatal(err)
	}
}

func TestThePDFExampleShowsTheLiveVersionAtEveryStep(t *testing.T) {
	s := newScene(t)
	hash := s.fileHash()[:8]
	a := domain.NewTextVersionID(pdfReader, "file-"+hash)
	b := domain.NewTextVersionID(proofreader, "from-"+strings.ReplaceAll(string(a), "/", "-"))
	c := domain.NewTextVersionID(ocr, "file-"+hash)
	d := domain.NewTextVersionID(proofreader, "from-"+strings.ReplaceAll(string(c), "/", "-"))

	// A: the file has no version, so the reader that runs on its own is asked for one.
	if err := s.versions.RequestText(t.Context(), s.vault, scanPath, s.fileHash()); err != nil {
		t.Fatal(err)
	}
	s.requireLive(t, "A from the file", a, "layer teh ")
	if got := s.listener.take(); !slices.Equal(got, []string{scanPath}) {
		t.Fatalf("putting A reported %v, want one change of %s", got, scanPath)
	}

	// B: proofread A.
	s.runEntry(t, textversion.ActionProofread, proofreader)
	s.requireLive(t, "B from A", b, "layer the ")

	// C: recognise the file.
	s.runEntry(t, textversion.ActionRecognise, ocr)
	s.requireLive(t, "C from the file", c, "scanned teh ")

	// D: proofread C, which is what the file shows now.
	s.runEntry(t, textversion.ActionProofread, proofreader)
	s.requireLive(t, "D from C", d, "scanned the ")
	s.listener.take()

	// Disable OCR: C and D are dead, and B is shown.
	s.setEnabled(t, ocr, false)
	s.requireLive(t, "OCR disabled", b, "layer the ")
	if got := s.listener.take(); !slices.Equal(got, []string{scanPath}) {
		t.Fatalf("disabling OCR reported %v, want one change of %s", got, scanPath)
	}

	// Enable OCR again: D is shown.
	s.setEnabled(t, ocr, true)
	s.requireLive(t, "OCR enabled", d, "scanned the ")
}

func TestReplacingTheFileMakesItsVersionsStale(t *testing.T) {
	s := newScene(t)
	if err := s.versions.RequestText(t.Context(), s.vault, scanPath, s.fileHash()); err != nil {
		t.Fatal(err)
	}
	s.runEntry(t, textversion.ActionProofread, proofreader)
	first := s.fileHash()

	s.files[scanPath] = []byte("%PDF scan, second printing")
	listing, err := s.versions.ListVersions(t.Context(), s.vault, scanPath, s.fileHash())
	if err != nil {
		t.Fatal(err)
	}
	if listing.Live != "" {
		t.Fatalf("%s is live after the file was replaced", listing.Live)
	}
	if _, err := s.versions.GetLiveText(t.Context(), s.vault, scanPath, s.fileHash()); !errors.Is(err, textport.ErrNoText) {
		t.Fatalf("the live text of a replaced file: got %v, want ErrNoText", err)
	}
	if menu, _ := s.versions.GetMenu(t.Context(), s.vault, scanPath, s.fileHash()); slices.ContainsFunc(menu, func(e textversion.MenuEntry) bool {
		return e.Action == textversion.ActionProofread
	}) {
		t.Fatalf("the menu offers to proofread a file with no live version: %+v", menu)
	}

	// The new content has no version, so the reader is asked again, and its
	// version of the new bytes is shown.
	if err := s.versions.RequestText(t.Context(), s.vault, scanPath, s.fileHash()); err != nil {
		t.Fatal(err)
	}
	if got := s.reader.getCreated(); got != 2 {
		t.Fatalf("the reader made %d texts, want 2", got)
	}
	s.requireLive(t, "after the replacement", domain.NewTextVersionID(pdfReader, "file-"+s.fileHash()[:8]), "layer teh ")

	// The old content's versions are still listed, and are live for those bytes.
	listing, err = s.versions.ListVersions(t.Context(), s.vault, scanPath, first)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(string(listing.Live), proofreader+"/") {
		t.Fatalf("for the first bytes the live version is %q, want the proofreading", listing.Live)
	}
}

func TestRequestTextAsksOncePerContentAndNotForALiveFile(t *testing.T) {
	s := newScene(t)
	for range 3 {
		if err := s.versions.RequestText(t.Context(), s.vault, scanPath, s.fileHash()); err != nil {
			t.Fatal(err)
		}
	}
	if got := s.reader.getCreated(); got != 1 {
		t.Fatalf("the reader was asked %d times, want 1", got)
	}
	if got := s.ocr.getCreated(); got != 0 {
		t.Fatalf("the recogniser, which does not run on its own, was asked %d times", got)
	}
}

func TestTheMenuFollowsTheEnabledPlugins(t *testing.T) {
	s := newScene(t)
	menu, err := s.versions.GetMenu(t.Context(), s.vault, scanPath, s.fileHash())
	if err != nil {
		t.Fatal(err)
	}
	want := []textversion.MenuEntry{
		{Action: textversion.ActionRecognise, PluginID: ocr, PluginName: "OCR"},
		{Action: textversion.ActionRecognise, PluginID: pdfReader, PluginName: "PDF"},
	}
	if !slices.Equal(menu, want) {
		t.Fatalf("a file with no version: menu %+v, want %+v", menu, want)
	}
	if menu, _ := s.versions.GetMenu(t.Context(), s.vault, "notes/a.md", ""); len(menu) != 0 {
		t.Fatalf("a note is offered %+v", menu)
	}

	if err := s.versions.RequestText(t.Context(), s.vault, scanPath, s.fileHash()); err != nil {
		t.Fatal(err)
	}
	s.plugins.setEnabled(ocr, false)
	menu, err = s.versions.GetMenu(t.Context(), s.vault, scanPath, s.fileHash())
	if err != nil {
		t.Fatal(err)
	}
	a := domain.NewTextVersionID(pdfReader, "file-"+s.fileHash()[:8])
	want = []textversion.MenuEntry{
		{Action: textversion.ActionRecognise, PluginID: pdfReader, PluginName: "PDF"},
		{Action: textversion.ActionProofread, PluginID: proofreader, PluginName: "Proofreading", Version: a},
		{Action: textversion.ActionDeleteVersion, PluginID: pdfReader, PluginName: "PDF", Version: a},
	}
	if !slices.Equal(menu, want) {
		t.Fatalf("a file with one version and OCR disabled: menu %+v, want %+v", menu, want)
	}
}

func TestDeleteVersionAsksTheOwningPlugin(t *testing.T) {
	s := newScene(t)
	if err := s.versions.RequestText(t.Context(), s.vault, scanPath, s.fileHash()); err != nil {
		t.Fatal(err)
	}
	s.runEntry(t, textversion.ActionProofread, proofreader)
	s.listener.take()

	b := domain.NewTextVersionID(proofreader, "from-"+strings.ReplaceAll(string(domain.NewTextVersionID(pdfReader, "file-"+s.fileHash()[:8])), "/", "-"))
	if err := s.versions.RunMenuEntry(t.Context(), s.vault, scanPath, textversion.MenuEntry{
		Action: textversion.ActionDeleteVersion, PluginID: proofreader, Version: b,
	}); err != nil {
		t.Fatal(err)
	}
	if len(s.fixer.deleted) != 1 || len(s.reader.deleted) != 0 {
		t.Fatalf("deleted by the proofreader %v and by the reader %v, want one by the proofreader", s.fixer.deleted, s.reader.deleted)
	}
	s.requireLive(t, "after deleting B", domain.NewTextVersionID(pdfReader, "file-"+s.fileHash()[:8]), "layer teh ")
	if got := s.listener.take(); len(got) == 0 {
		t.Fatal("deleting a version reported no change")
	}
}

func TestSaveVersionRejectsInvalidVersions(t *testing.T) {
	s := newScene(t)
	ctx := t.Context()
	cases := map[string]struct {
		pluginID string
		params   textversion.VersionPut
		want     error
	}{
		"a plugin that makes no text": {"org.numen.other", textversion.VersionPut{Path: scanPath, Key: "k", Format: domain.TextFormatPlain}, textport.ErrNotTextPlugin},
		"no key":                      {ocr, textversion.VersionPut{Path: scanPath, Format: domain.TextFormatPlain}, textport.ErrInvalidVersion},
		"no format":                   {ocr, textversion.VersionPut{Path: scanPath, Key: "k"}, textport.ErrInvalidVersion},
		"an input that is not there":  {ocr, textversion.VersionPut{Path: scanPath, Key: "k", Format: domain.TextFormatPlain, Input: "x/y"}, textport.ErrInvalidVersion},
		"itself as its input":         {ocr, textversion.VersionPut{Path: scanPath, Key: "k", Format: domain.TextFormatPlain, Input: domain.NewTextVersionID(ocr, "k")}, textport.ErrInvalidVersion},
	}
	for name, c := range cases {
		if _, err := s.versions.PutVersion(ctx, s.vault, c.pluginID, c.params); !errors.Is(err, c.want) {
			t.Errorf("%s: got %v, want %v", name, err, c.want)
		}
	}
	if got := s.listener.take(); len(got) != 0 {
		t.Fatalf("a refused put reported %v", got)
	}
}

func TestSaveVersionPreservesCreateTime(t *testing.T) {
	s := newScene(t)
	params := textversion.VersionPut{Path: scanPath, Key: "k", Format: domain.TextFormatPositioned, InputHash: s.fileHash(), Hash: "1"}
	first, err := s.versions.PutVersion(t.Context(), s.vault, ocr, params)
	if err != nil {
		t.Fatal(err)
	}
	params.Hash = "2"
	second, err := s.versions.PutVersion(t.Context(), s.vault, ocr, params)
	if err != nil {
		t.Fatal(err)
	}
	if !second.CreateTime.Equal(first.CreateTime) || second.Hash != "2" {
		t.Fatalf("replaced version %+v, want the first time and the new hash", second)
	}
}

func TestCreateTextRefusesAnInputThePluginDoesNotTake(t *testing.T) {
	s := newScene(t)
	if err := s.versions.CreateText(t.Context(), s.vault, proofreader, scanPath, ""); !errors.Is(err, textport.ErrInputNotAccepted) {
		t.Fatalf("a proofreader asked to read a file: got %v", err)
	}
	if err := s.versions.CreateText(t.Context(), s.vault, ocr, "notes/a.md", ""); !errors.Is(err, textport.ErrInputNotAccepted) {
		t.Fatalf("a recogniser asked to read a note: got %v", err)
	}
}

func TestGetLiveTextReadsEveryPage(t *testing.T) {
	s := newScene(t)
	s.reader.pageSize = 1
	if err := s.versions.RequestText(t.Context(), s.vault, scanPath, s.fileHash()); err != nil {
		t.Fatal(err)
	}
	text, err := s.versions.GetLiveText(t.Context(), s.vault, scanPath, s.fileHash())
	if err != nil {
		t.Fatal(err)
	}
	if text.Text != "layer teh " || text.Locate(7) != "page 2 of the file" {
		t.Fatalf("text %q located %q", text.Text, text.Locate(7))
	}
}

func TestGetVersionTextOfADisabledPluginIsNoText(t *testing.T) {
	s := newScene(t)
	if err := s.versions.RequestText(t.Context(), s.vault, scanPath, s.fileHash()); err != nil {
		t.Fatal(err)
	}
	s.plugins.setEnabled(pdfReader, false)
	a := domain.NewTextVersionID(pdfReader, "file-"+s.fileHash()[:8])
	if _, err := s.versions.GetVersionText(t.Context(), s.vault, scanPath, a); !errors.Is(err, textport.ErrNoText) {
		t.Fatalf("got %v, want ErrNoText", err)
	}
}

func TestCreateTextMarksMenuRunsUserStarted(t *testing.T) {
	s := newScene(t)
	if err := s.versions.RequestText(t.Context(), s.vault, scanPath, s.fileHash()); err != nil {
		t.Fatal(err)
	}
	s.runEntry(t, textversion.ActionRecognise, ocr)

	if !slices.Equal(s.reader.isUserStarted, []bool{false}) {
		t.Errorf("the reader run on its own was asked as %v, want not user-started", s.reader.isUserStarted)
	}
	if !slices.Equal(s.ocr.isUserStarted, []bool{true}) {
		t.Errorf("the recogniser run from the menu was asked as %v, want user-started", s.ocr.isUserStarted)
	}
}
