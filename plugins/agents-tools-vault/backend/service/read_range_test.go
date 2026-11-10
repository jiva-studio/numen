package service_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/domain"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/hosttest"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/tools"
	"github.com/jiva-studio/numen/plugins/agents-tools-vault/backend/service"
)

// rangeRequest is one call of ReadRange as the host received it.
type rangeRequest struct {
	Path          string
	Start, Length int
}

// rangeCalls is what the host of a ranged vault received.
type rangeCalls struct {
	mu     sync.Mutex
	ranges []rangeRequest
	reads  []string
}

// rangeVault is a vault whose host returns ReadRange with the range set for a path, else the error
// set for it, else as the fake host does, and records every ReadRange and Read.
type rangeVault struct {
	*hosttest.Vault
	calls       *rangeCalls
	answers     map[string]host.FileRange
	rangeErrors map[string]error
	readRefused map[string]error
}

func newRangeVault(v *hosttest.Vault) rangeVault {
	return rangeVault{
		Vault: v, calls: &rangeCalls{},
		answers: map[string]host.FileRange{}, rangeErrors: map[string]error{}, readRefused: map[string]error{},
	}
}

func (r rangeVault) With(domain.VaultID) rangeVault { return r }

func (r rangeVault) ReadRange(ctx context.Context, filePath string, start, length int) (host.FileRange, error) {
	r.calls.mu.Lock()
	r.calls.ranges = append(r.calls.ranges, rangeRequest{Path: filePath, Start: start, Length: length})
	r.calls.mu.Unlock()
	if answer, isSet := r.answers[filePath]; isSet {
		return answer, nil
	}
	if err := r.rangeErrors[filePath]; err != nil {
		return host.FileRange{}, err
	}
	return r.Vault.ReadRange(ctx, filePath, start, length)
}

func (r rangeVault) Read(ctx context.Context, filePath string) ([]byte, domain.FileStat, error) {
	r.calls.mu.Lock()
	r.calls.reads = append(r.calls.reads, filePath)
	r.calls.mu.Unlock()
	if err := r.readRefused[filePath]; err != nil {
		return nil, domain.FileStat{}, err
	}
	return r.Vault.Read(ctx, filePath)
}

// textCalls is what the host of the counted text versions received.
type textCalls struct {
	mu                sync.Mutex
	isReadTextRefused bool
	readTexts         int
	ranges            []rangeRequest
}

// countingTexts is the fake text versions that record ReadText and ReadTextRange.
type countingTexts struct {
	host.TextReader
	calls *textCalls
}

func (c countingTexts) ReadText(ctx context.Context, path string, id domain.TextVersionID, cursor string) (host.TextPage, error) {
	c.calls.mu.Lock()
	c.calls.readTexts++
	isRefused := c.calls.isReadTextRefused
	c.calls.mu.Unlock()
	if isRefused {
		return host.TextPage{}, errors.New("ReadText reads the whole text, which a ranged read does not need")
	}
	return c.TextReader.ReadText(ctx, path, id, cursor)
}

func (c countingTexts) ReadTextRange(ctx context.Context, path string, id domain.TextVersionID, start, length int) (host.TextRange, error) {
	c.calls.mu.Lock()
	c.calls.ranges = append(c.calls.ranges, rangeRequest{Path: path, Start: start, Length: length})
	c.calls.mu.Unlock()
	return c.TextReader.ReadTextRange(ctx, path, id, start, length)
}

func callFailure(t *testing.T, registry *tools.Registry, vaultID, name, arguments string, out any) string {
	t.Helper()
	res, err := registry.InvokeTool(t.Context(), tools.Call{VaultID: vaultID, Name: name, Arguments: []byte(arguments)})
	if err != nil {
		t.Fatal(err)
	}
	if res.GetResult().GetIsError() {
		return res.GetResult().GetText()
	}
	if res.GetError() != tools.ErrorCodeUnspecified {
		t.Fatalf("%s gave error code %v", name, res.GetError())
	}
	if out != nil {
		if err := json.Unmarshal(res.GetResult().GetStructured(), out); err != nil {
			t.Fatalf("%s: %v in %s", name, err, res.GetResult().GetStructured())
		}
	}
	return ""
}

func newRangeLibrary(t *testing.T) (library, rangeVault, *textCalls, *tools.Registry) {
	t.Helper()
	l := newLibrary(t)
	vault := newRangeVault(l.vault)
	counted := &textCalls{}
	registry, err := service.New(vault.vaults()).WithLibrary(func(id domain.VaultID) service.Library {
		on := l.texts.With(id)
		return service.Library{Versions: on, Texts: countingTexts{TextReader: on, calls: counted}, Search: l.search.With(id)}
	}).Registry()
	if err != nil {
		t.Fatal(err)
	}
	return l, vault, counted, registry
}

func TestFileReadReturnsTheRangeTheHostGaveAndTheSizeOfTheFile(t *testing.T) {
	_, vault, _, registry := newRangeLibrary(t)
	stat := domain.FileStat{Path: "data.txt", Size: 4096, ModTime: time.Unix(0, 1700000000000000000)}
	vault.answers["data.txt"] = host.FileRange{Text: "line 2", Start: 8, Stat: stat}

	var out fileReadAnswer
	if failure := callFailure(t, registry, testVault, "file_read", `{"path":"data.txt","start":7,"length":6}`, &out); failure != "" {
		t.Fatal(failure)
	}
	if len(vault.calls.ranges) != 1 || vault.calls.ranges[0] != (rangeRequest{Path: "data.txt", Start: 7, Length: 6}) {
		t.Errorf("the host received the ranges %+v, want data.txt from 7 for 6 bytes", vault.calls.ranges)
	}
	if len(vault.calls.reads) != 0 {
		t.Errorf("file_read read %v whole, want only the range", vault.calls.reads)
	}
	if out.Path != "data.txt" || out.Text != "line 2" || out.Start != 8 || out.Length != 6 || out.Size != 4096 {
		t.Errorf("file_read returned %+v, want line 2 from 8 for 6 bytes of a file of 4096", out)
	}
	if want := "4096-" + strconv.FormatInt(stat.ModTime.UnixNano(), 10); out.Fingerprint != want {
		t.Errorf("the fingerprint is %q, want %q from the file stat of the range", out.Fingerprint, want)
	}
}

func TestFileReadWithoutALengthRequests16000BytesFromTheHost(t *testing.T) {
	_, vault, _, registry := newRangeLibrary(t)
	vault.answers["data.txt"] = host.FileRange{Text: "line 1\n", Start: 0, Stat: domain.FileStat{Path: "data.txt", Size: 7}}

	if failure := callFailure(t, registry, testVault, "file_read", `{"path":"data.txt"}`, nil); failure != "" {
		t.Fatal(failure)
	}
	if len(vault.calls.ranges) != 1 || vault.calls.ranges[0].Start != 0 || vault.calls.ranges[0].Length != 16000 {
		t.Errorf("the host received the ranges %+v, want one from 0 for 16000 bytes", vault.calls.ranges)
	}
}

func TestFileReadOfARangeThatIsNotTextNamesLibraryRead(t *testing.T) {
	_, vault, _, registry := newRangeLibrary(t)
	vault.rangeErrors[guidePath] = fmt.Errorf("read %s: %w", guidePath, host.ErrNotText)

	failure := callFailure(t, registry, testVault, "file_read", `{"path":"`+guidePath+`"}`, nil)
	want := guidePath + " is not a text file; library_read reads the text of a PDF, a book or a recording"
	if !strings.Contains(failure, want) {
		t.Errorf("file_read of a binary gave %q, want %q", failure, want)
	}
}

func TestFileReadMovesAStartInsideACharacterToTheNextCharacter(t *testing.T) {
	l := newLibrary(t)
	l.vault.File("notes/Moor.txt", "Öl und Ödland")

	var out fileReadAnswer
	if failure := l.call(t, "file_read", `{"path":"notes/Moor.txt","start":1,"length":5}`, &out); failure != "" {
		t.Fatal(failure)
	}
	if out.Text != "l un" || out.Start != 2 || out.Length != 4 || out.Size != len("Öl und Ödland") {
		t.Errorf("file_read from the middle of Ö gave %+v, want l un from 2 for 4 bytes of %d", out, len("Öl und Ödland"))
	}
}

func TestLibraryReadOfNonTextFileStatesThatARecognitionOrTranscriptionCreatesOne(t *testing.T) {
	l := newLibrary(t)
	failure := l.call(t, "library_read", `{"path":"`+scanPath+`"}`, nil)
	want := scanPath + " has no text version; a recognition or a transcription creates one"
	if !strings.Contains(failure, want) {
		t.Errorf("library_read of a scan with no text version gave %q, want %q", failure, want)
	}
}

func TestLibraryReadOfLargeBinaryStatesThatARecognitionOrTranscriptionCreatesOne(t *testing.T) {
	const archivePath = "recordings/Long Night.wav"
	l, vault, _, registry := newRangeLibrary(t)
	l.vault.File(archivePath, "RIFF\x00\x00wave body")
	vault.readRefused[archivePath] = errors.New("the host refused the call with the code HOST_VAULT_ERROR_CODE_TOO_LARGE")
	vault.rangeErrors[archivePath] = fmt.Errorf("read %s: %w", archivePath, host.ErrNotText)

	failure := callFailure(t, registry, testVault, "library_read", `{"path":"`+archivePath+`"}`, nil)
	want := archivePath + " has no text version; a recognition or a transcription creates one"
	if !strings.Contains(failure, want) || strings.Contains(failure, "TOO_LARGE") {
		t.Errorf("library_read of a large recording with no text version gave %q, want %q", failure, want)
	}
}

func TestLibraryReadOfATextFileReadsItsRangeThroughTheHost(t *testing.T) {
	_, vault, _, registry := newRangeLibrary(t)
	vault.answers["data.txt"] = host.FileRange{Text: "line 2\n", Start: 7, Stat: domain.FileStat{Path: "data.txt", Size: 21}}

	var out libraryReadAnswer
	if failure := callFailure(t, registry, testVault, "library_read", `{"path":"data.txt","start":7,"length":7}`, &out); failure != "" {
		t.Fatal(failure)
	}
	if len(vault.calls.ranges) != 1 || vault.calls.ranges[0].Start != 7 {
		t.Errorf("the host received the ranges %+v, want one from 7", vault.calls.ranges)
	}
	if len(vault.calls.reads) != 0 {
		t.Errorf("library_read read %v whole, want only the range", vault.calls.reads)
	}
	if out.Text != "line 2\n" || out.Start != 7 || out.Length != 7 || out.Size != 21 || out.NextStart == nil || *out.NextStart != 14 {
		t.Errorf("library_read of the range gave %+v, want line 2 from 7 for 7 of 21 and next 14", out)
	}
}

func TestLibraryReadByStartAndLengthReadsOnlyThatTextRange(t *testing.T) {
	_, _, counted, registry := newRangeLibrary(t)
	counted.isReadTextRefused = true
	whole := pageOne + pageTwo + pageThree

	var out libraryReadAnswer
	if failure := callFailure(t, registry, testVault, "library_read", `{"path":"`+guidePath+`","start":3,"length":10}`, &out); failure != "" {
		t.Fatal(failure)
	}
	if counted.readTexts != 0 {
		t.Errorf("library_read called ReadText %d times, want none for a start and a length", counted.readTexts)
	}
	if len(counted.ranges) != 1 || counted.ranges[0] != (rangeRequest{Path: guidePath, Start: 3, Length: 10}) {
		t.Errorf("the host received the text ranges %+v, want %s from 3 for 10 bytes", counted.ranges, guidePath)
	}
	if out.Text != whole[3:13] || out.Start != 3 || out.Length != 10 || out.Size != len(whole) || out.NextStart == nil || *out.NextStart != 13 {
		t.Errorf("library_read gave %q at %d for %d of %d next %v, want %q at 3 for 10 of %d next 13",
			out.Text, out.Start, out.Length, out.Size, out.NextStart, whole[3:13], len(whole))
	}
	if len(out.Ranges) != 1 || getPlacePage(out.Ranges[0].Place) != 1 || out.Ranges[0].Start != 3 || out.Ranges[0].Length != 10 {
		t.Errorf("ranges %+v, want page 1 from 3 for 10", out.Ranges)
	}
}

func TestLibraryReadOfAPageReturnsItsRangesViaATextRange(t *testing.T) {
	_, _, counted, registry := newRangeLibrary(t)

	var out libraryReadAnswer
	if failure := callFailure(t, registry, testVault, "library_read", `{"path":"`+guidePath+`","page":2}`, &out); failure != "" {
		t.Fatal(failure)
	}
	if out.Text != pageTwo || out.Start != len(pageOne) || out.Length != len(pageTwo) {
		t.Errorf("page 2 is %q at %d for %d", out.Text, out.Start, out.Length)
	}
	if out.Size != len(pageOne)+len(pageTwo)+len(pageThree) || out.Kind != "page-text" {
		t.Errorf("size %d kind %q", out.Size, out.Kind)
	}
	if len(out.Ranges) != 1 || getPlacePage(out.Ranges[0].Place) != 2 || out.Ranges[0].Start != len(pageOne) || out.Ranges[0].Length != len(pageTwo) {
		t.Errorf("ranges %+v, want the one page, its two segments joined", out.Ranges)
	}
	if out.NextStart == nil || *out.NextStart != len(pageOne)+len(pageTwo) {
		t.Fatalf("next %v, want the end of the page", out.NextStart)
	}
	if len(counted.ranges) != 1 || counted.ranges[0].Start != len(pageOne) {
		t.Errorf("the host received the text ranges %+v, want one from the start of page 2", counted.ranges)
	}

	var rest libraryReadAnswer
	if failure := callFailure(t, registry, testVault, "library_read", `{"path":"`+guidePath+`","start":`+strconv.Itoa(*out.NextStart)+`}`, &rest); failure != "" {
		t.Fatal(failure)
	}
	if rest.Text != pageThree || rest.NextStart != nil || len(rest.Ranges) != 1 || getPlacePage(rest.Ranges[0].Place) != 3 {
		t.Errorf("the continuation is %+v, want page 3 and no next", rest)
	}
}

// repeatingTexts is the texts of the host that return the first page of every version again for
// the cursor of the page after it.
type repeatingTexts struct {
	host.TextReader
}

func (r repeatingTexts) ReadText(ctx context.Context, path string, id domain.TextVersionID, _ string) (host.TextPage, error) {
	page, err := r.TextReader.ReadText(ctx, path, id, "")
	if err != nil {
		return host.TextPage{}, err
	}
	if len(page.Segments) > 1 {
		page.Segments = page.Segments[:1]
	}
	page.NextCursor = "1"
	return page, nil
}

func TestLibraryReadOfAPageStartingOnTheSecondHostPageReturnsThatPage(t *testing.T) {
	l := newLibrary(t)
	l.texts.SetPageSize(2)
	registry, err := service.New(vaultsOf(l.vault)).WithLibrary(libraryOf(l.texts, l.search)).Registry()
	if err != nil {
		t.Fatal(err)
	}

	var out libraryReadAnswer
	if failure := callFailure(t, registry, testVault, "library_read", `{"path":"`+guidePath+`","page":3}`, &out); failure != "" {
		t.Fatal(failure)
	}
	if out.Text != pageThree || out.Start != len(pageOne)+len(pageTwo) || out.Length != len(pageThree) || out.NextStart != nil {
		t.Errorf("page 3 is %q at %d for %d next %v, want %q at %d and no next", out.Text, out.Start, out.Length, out.NextStart, pageThree, len(pageOne)+len(pageTwo))
	}
	if len(out.Ranges) != 1 || getPlacePage(out.Ranges[0].Place) != 3 {
		t.Errorf("ranges %+v, want page 3", out.Ranges)
	}
}

func TestLibraryReadOfAPageFailsWhenTheHostReturnsTheSameCursorTwice(t *testing.T) {
	l := newLibrary(t)
	registry, err := service.New(vaultsOf(l.vault)).WithLibrary(func(id domain.VaultID) service.Library {
		on := l.texts.With(id)
		return service.Library{Versions: on, Texts: repeatingTexts{TextReader: on}, Search: l.search.With(id)}
	}).Registry()
	if err != nil {
		t.Fatal(err)
	}

	failure := callFailure(t, registry, testVault, "library_read", `{"path":"`+guidePath+`","page":3}`, nil)
	if want := `returned the page at "1" twice`; !strings.Contains(failure, want) {
		t.Errorf("library_read over a host that repeats the cursor 1 gave %q, want it to contain %q", failure, want)
	}
}

func (r rangeVault) vaults() func(domain.VaultID) service.Vault {
	return func(id domain.VaultID) service.Vault {
		on := r.With(id)
		return service.Vault{Files: on, Writer: on, Index: on, Notes: on, Links: on, Arranger: on, Editor: on, Selector: on}
	}
}
