package editor

import (
	"context"
	"net/http"
	"testing"

	"connectrpc.com/connect"

	v1 "github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1"

	"github.com/jiva-studio/numen/modules/libs/core/domain"
	"github.com/jiva-studio/numen/modules/libs/core/internal/adapter/filesystem"
	"github.com/jiva-studio/numen/modules/libs/core/internal/testsupport"
	derived "github.com/jiva-studio/numen/modules/libs/core/internal/text"
	"github.com/jiva-studio/numen/modules/libs/core/port"
	"github.com/jiva-studio/numen/modules/libs/core/usecase/note"
	"github.com/jiva-studio/numen/modules/libs/core/usecase/source"
)

// What read the scan a test asks about, and the bytes it read.
const (
	reader  = "ocr"
	scanned = "7c3a91f4"
)

// A note stands in the vault beside the scan and the recording, as a file
// neither run is for.
const idea = "notes/idea.md"

// A link note stands there too, pointing at a video. What is at an address is
// made from the note the way a reading is made from a scan.
const (
	pointed  = "notes/talk.url"
	pointsAt = "https://www.youtube.com/watch?v=dQw4w9WgXcQ"
)

// asking is a run a test hands the window: what it makes of a source it is
// given, and what it was given.
type asking struct {
	takes port.StartOutcome
	vault string
	path  string
	times int
}

func (a *asking) Start(v domain.Vault, path string) port.StartOutcome {
	a.times++
	a.vault, a.path = string(v.ID), path
	return a.takes
}

// claimed is a store a run holds one name in. A run holds the name it writes
// under for as long as it takes, and a claim on it is refused.
type claimed struct {
	stored
	name string
}

func (c claimed) Claim(_ context.Context, name string) (func() error, error) {
	if name == c.name {
		return nil, port.ErrClaimed
	}
	return func() error { return nil }, nil
}

// openRunWindow is a window over a vault holding a scan, a recording and a note, with
// the runs a test hands it and whatever a run before this one wrote down.
//
// read is what the index says each source's text came from; a source it does
// not name is one nothing has produced a text of.
func openRunWindow(
	t *testing.T,
	held port.DerivedStore,
	read indexed,
	scans, hears Runner,
) (*API, http.Handler) {
	t.Helper()
	vault := testsupport.NewVault(t, map[string]string{
		book:    "the bytes of a scan",
		talk:    sound,
		idea:    "# an idea\n",
		pointed: "[InternetShortcut]\nURL=" + pointsAt + "\n",
	})
	api := &API{
		Readers:   filesystem.VaultReaders{},
		Highlight: &source.Highlight{Sources: read, Derived: storing{held}},
	}
	// A note is read to find out where it points, which is what says whether
	// anything is made from it.
	api.Notes.Read = &noteRead
	// What a url carries follows from what fetches it, so a window that reaches
	// no address at all says a url carries nothing.
	api.Imports = &source.ImportURL{By: reachingASite{}}
	api.show(vault)
	setPasses(api, func(on *passes) { on.recognises, on.transcribes = scans, hears })
	return api, api.NewHandler(http.NotFoundHandler())
}

// noteRead is how a note is read, which is where a link note says it points.
var noteRead = note.NewRead(filesystem.VaultReaders{})

// willRun is a run that begins what it is given now, and willQueue one that
// puts it in line behind the work already going.
func willRun() *asking   { return &asking{takes: port.Began} }
func willQueue() *asking { return &asking{takes: port.Queued} }

// newEmptyIndex is an index holding no produced text at all.
func newEmptyIndex() indexed { return indexed{} }

// The artifacts a test asks for, as the schema names them.
const (
	readingOf    = v1.ArtifactKind_ARTIFACT_KIND_OCR
	transcriptOf = v1.ArtifactKind_ARTIFACT_KIND_TRANSCRIPT
	correctedOf  = v1.ArtifactKind_ARTIFACT_KIND_TRANSCRIPT_CORRECTED
	articleOf    = v1.ArtifactKind_ARTIFACT_KIND_ARTICLE
	copyOf       = v1.ArtifactKind_ARTIFACT_KIND_COPY
)

// makes asks for an artifact of a file to be made, and answers with what came
// back.
func makes(api *API, path string, of v1.ArtifactKind) (*connect.Response[v1.CreateArtifactResponse], error) {
	return api.CreateArtifact(context.Background(), connect.NewRequest(&v1.CreateArtifactRequest{
		Path: path, Kind: of,
	}))
}

// createArtifact asks for one artifact of one file to be made, and fails the test where
// the window would not.
func createArtifact(t *testing.T, api *API, path string, of v1.ArtifactKind) *v1.Artifact {
	t.Helper()
	out, err := makes(api, path, of)
	if err != nil {
		t.Fatalf("asked for the %s of %s and was refused: %v", of, path, err)
	}
	return out.Msg.GetArtifact()
}

// getArtifactStates is what a file carries, by the artifact each row is of.
func getArtifactStates(t *testing.T, api *API, path string) map[v1.ArtifactKind]v1.State {
	t.Helper()
	out, err := api.ListArtifacts(t.Context(), connect.NewRequest(&v1.ListArtifactsRequest{Path: path}))
	if err != nil {
		t.Fatalf("asked what %s carries and was refused: %v", path, err)
	}
	by := map[v1.ArtifactKind]v1.State{}
	for _, one := range out.Msg.GetArtifacts() {
		by[one.GetKind()] = one.GetState()
	}
	return by
}

// getRefusedCode is the code the window would not make an artifact under.
func getRefusedCode(t *testing.T, api *API, path string, of v1.ArtifactKind) connect.Code {
	t.Helper()
	out, err := makes(api, path, of)
	if err == nil {
		t.Fatalf("the %s of %s was made: %+v", of, path, out.Msg.GetArtifact())
	}
	return connect.CodeOf(err)
}

// A scan the window asks for is read, and the run is told which file of which
// vault.
func TestAScanIsReadWhenTheWindowAsksForIt(t *testing.T) {
	scans := willRun()
	api, _ := openRunWindow(t, stored{}, newEmptyIndex(), scans, willRun())

	made := createArtifact(t, api, book, readingOf)
	if made.GetState() != v1.State_STATE_RUNNING {
		t.Fatalf("the scan was answered %s", made.GetState())
	}
	if made.GetKind() != v1.ArtifactKind_ARTIFACT_KIND_OCR {
		t.Errorf("the answer is about a %s", made.GetKind())
	}
	if scans.times != 1 || scans.path != book || scans.vault != string(api.GetShownVault().ID) {
		t.Errorf("the run was asked for %q of %q, %d times", scans.path, scans.vault, scans.times)
	}
}

// A recording the window asks for is heard.
func TestARecordingIsHeardWhenTheWindowAsksForIt(t *testing.T) {
	hears := willRun()
	api, _ := openRunWindow(t, stored{}, newEmptyIndex(), willRun(), hears)

	made := createArtifact(t, api, talk, transcriptOf)
	if made.GetState() != v1.State_STATE_RUNNING {
		t.Fatalf("the recording was answered %s", made.GetState())
	}
	if made.GetKind() != v1.ArtifactKind_ARTIFACT_KIND_TRANSCRIPT {
		t.Errorf("the answer is about a %s", made.GetKind())
	}
	if hears.times != 1 || hears.path != talk || hears.vault != string(api.GetShownVault().ID) {
		t.Errorf("the run was asked for %q of %q, %d times", hears.path, hears.vault, hears.times)
	}
}

// A source named while a run of its kind is going waits its turn. Nothing is
// turned away.
func TestASourceNamedWhileARunIsGoingWaitsItsTurn(t *testing.T) {
	for _, one := range []struct {
		name string
		of   v1.ArtifactKind
		path string
	}{
		{"a scan", readingOf, book},
		{"a recording", transcriptOf, talk},
	} {
		t.Run(one.name, func(t *testing.T) {
			scans, hears := willQueue(), willQueue()
			api, _ := openRunWindow(t, stored{}, newEmptyIndex(), scans, hears)

			made := createArtifact(t, api, one.path, one.of)
			if made.GetState() != v1.State_STATE_QUEUED {
				t.Fatalf("it was answered %s", made.GetState())
			}
			if scans.times+hears.times != 1 {
				t.Error("the run was not given the source a person named")
			}
		})
	}
}

// A file carries the artifacts its kind carries and no others, and nothing is
// given to a run over one it does not.
func TestAFileOnlyCarriesTheArtifactsItsKindDoes(t *testing.T) {
	for _, one := range []struct {
		name string
		of   v1.ArtifactKind
		path string
	}{
		{"a scan asked to be heard", transcriptOf, book},
		{"a recording asked to be read", readingOf, talk},
		{"a note asked to be read", readingOf, idea},
		{"a note asked to be heard", transcriptOf, idea},
		{"a scan asked to be put right", correctedOf, book},
	} {
		t.Run(one.name, func(t *testing.T) {
			scans, hears := willRun(), willRun()
			api, _ := openRunWindow(t, stored{}, newEmptyIndex(), scans, hears)

			if code := getRefusedCode(t, api, one.path, one.of); code != connect.CodeInvalidArgument {
				t.Errorf("a file carrying no such artifact was refused %s", code)
			}
			if scans.times+hears.times != 0 {
				t.Error("a run was given a file of the wrong kind")
			}
		})
	}
}

// An artifact no file carries is refused before the vault is read at all.
func TestAnArtifactNoFileCarriesIsNotMade(t *testing.T) {
	scans, hears := willRun(), willRun()
	api, _ := openRunWindow(t, stored{}, newEmptyIndex(), scans, hears)

	if code := getRefusedCode(t, api, book, v1.ArtifactKind(99)); code != connect.CodeInvalidArgument {
		t.Errorf("an artifact nothing makes was refused %s", code)
	}
	if scans.times+hears.times != 0 {
		t.Error("a run was given a file for an artifact nothing makes")
	}
}

// A source already standing on the whole of the text a model produced is not
// put through the run again, and is told so plainly.
func TestASourceAlreadyDoneIsNotRunAgain(t *testing.T) {
	for _, one := range []struct {
		name string
		of   v1.ArtifactKind
		path string
		from string
		hash string
	}{
		{"a scan already read", readingOf, book, reader, scanned},
		{"a recording already heard", transcriptOf, talk, asr, hashed},
	} {
		t.Run(one.name, func(t *testing.T) {
			const wrote = "what the model wrote"
			scans, hears := willRun(), willRun()
			api, _ := openRunWindow(t,
				stored{derived.Artifact(one.from, one.hash): []byte(wrote)},
				indexed{one.path: {Fingerprint: domain.Fingerprint{Path: one.path}, Producer: one.from, Hash: one.hash}},
				scans, hears,
			)

			made := createArtifact(t, api, one.path, one.of)
			if made.GetState() != v1.State_STATE_DONE {
				t.Fatalf("a source already done was answered %s", made.GetState())
			}
			if scans.times+hears.times != 0 {
				t.Error("a run was given a source already done")
			}
		})
	}
}

// A source a run holds is said to be under way. A recording is heard without
// anybody asking, so a person asking for the one being heard is told that it is
// happening, and not that theirs did not start.
func TestASourceARunHoldsIsSaidToBeUnderWay(t *testing.T) {
	for _, one := range []struct {
		name string
		of   v1.ArtifactKind
		path string
		from string
		hash string
	}{
		{"a scan being read", readingOf, book, reader, scanned},
		{"a recording being heard", transcriptOf, talk, asr, hashed},
	} {
		t.Run(one.name, func(t *testing.T) {
			const far = "as far as it has got"
			scans, hears := willRun(), willRun()
			api, _ := openRunWindow(t,
				claimed{
					stored: stored{derived.Partial(one.from, one.hash): []byte(far)},
					name:   derived.Partial(one.from, one.hash),
				},
				indexed{one.path: {Fingerprint: domain.Fingerprint{Path: one.path}, Producer: one.from, Hash: one.hash}},
				scans, hears,
			)

			made := createArtifact(t, api, one.path, one.of)
			if made.GetState() != v1.State_STATE_RUNNING {
				t.Fatalf("a source a run holds was answered %s", made.GetState())
			}
			if scans.times+hears.times != 0 {
				t.Error("a run was given a source already being worked on")
			}
		})
	}
}

// A source nothing holds is not under way, whatever a run wrote before it
// stopped. It is named afresh and waits its turn.
func TestASourceNothingHoldsIsNotUnderWay(t *testing.T) {
	for _, one := range []struct {
		name string
		of   v1.ArtifactKind
		path string
		from string
		hash string
	}{
		{"a scan", readingOf, book, reader, scanned},
		{"a recording", transcriptOf, talk, asr, hashed},
	} {
		t.Run(one.name, func(t *testing.T) {
			api, _ := openRunWindow(t,
				stored{derived.Partial(one.from, one.hash): []byte("as far as it got")},
				indexed{one.path: {Fingerprint: domain.Fingerprint{Path: one.path}, Producer: one.from, Hash: one.hash}},
				willQueue(), willQueue(),
			)

			// What the run left is on disk, and the source is told apart from
			// one nothing has touched until it is named afresh.
			if state := getArtifactStates(t, api, one.path)[one.of]; state != v1.State_STATE_STOPPED {
				t.Errorf("a run that stopped left the source %s", state)
			}
			made := createArtifact(t, api, one.path, one.of)
			if made.GetState() != v1.State_STATE_QUEUED {
				t.Fatalf("it was answered %s", made.GetState())
			}
		})
	}
}

// A source a run holds and has finished is one already done. A finished run
// gives the name it was writing under back.
func TestASourceDoneIsDoneEvenWhereAPartialStands(t *testing.T) {
	scans := willRun()
	api, _ := openRunWindow(t,
		stored{
			derived.Artifact(reader, scanned): []byte("what the model wrote"),
			derived.Partial(reader, scanned):  []byte("what it wrote on the way"),
		},
		indexed{book: {Fingerprint: domain.Fingerprint{Path: book}, Producer: reader, Hash: scanned}},
		scans, willRun(),
	)

	if made := createArtifact(t, api, book, readingOf); made.GetState() != v1.State_STATE_DONE {
		t.Errorf("it was answered %s", made.GetState())
	}
	if scans.times != 0 {
		t.Error("a run was given a scan already read")
	}
}

// A source whose own bytes are the text has had no run over it, and asking for
// one begins it.
func TestASourceCarryingItsOwnTextHasNotBeenRun(t *testing.T) {
	scans := willRun()
	api, _ := openRunWindow(t, stored{},
		indexed{book: {Fingerprint: domain.Fingerprint{Path: book}, Hash: scanned}},
		scans, willRun(),
	)

	if made := createArtifact(t, api, book, readingOf); made.GetState() != v1.State_STATE_RUNNING {
		t.Fatalf("the scan was answered %s", made.GetState())
	}
	if scans.times != 1 {
		t.Errorf("the run was asked for %d times", scans.times)
	}
}

// A path the vault does not hold is not something a person can act on, and is
// refused as an error.
func TestARunOverAPathTheVaultDoesNotHoldIsNotFound(t *testing.T) {
	api, _ := openRunWindow(t, stored{}, newEmptyIndex(), willRun(), willRun())

	for of, path := range map[v1.ArtifactKind]string{
		readingOf:    "library/nothing.pdf",
		transcriptOf: "talks/nothing.mp3",
	} {
		if code := getRefusedCode(t, api, path, of); code != connect.CodeNotFound {
			t.Errorf("asked for a run over nothing and was refused %s", code)
		}
	}
}

// A vault is in the window before what runs behind it is, so a run asked for in
// between reaches no runner and the caller asks again.
func TestARunAskedForBeforeThePassesAreUpIsAskedAgain(t *testing.T) {
	api, _ := openRunWindow(t, stored{}, newEmptyIndex(), nil, nil)

	for of, path := range map[v1.ArtifactKind]string{readingOf: book, transcriptOf: talk} {
		if code := getRefusedCode(t, api, path, of); code != connect.CodeUnavailable {
			t.Errorf("asked before the passes were up and was refused %s", code)
		}
	}
}

// Listing what a file carries changes nothing, and no run is given a source for
// a question that only asks.
func TestListingWhatAFileCarriesBeginsNoRun(t *testing.T) {
	scans, hears := willRun(), willRun()
	api, _ := openRunWindow(t, stored{}, newEmptyIndex(), scans, hears)

	for _, path := range []string{book, talk} {
		getArtifactStates(t, api, path)
	}
	if scans.times+hears.times != 0 {
		t.Error("a run was given a source by a question that only asked")
	}
}

// reachingASite is a downloader that reaches a video and nothing else: what a
// url carries follows from what downloads it, and a test says which that is.
type reachingASite struct{ port.Downloader }

func (reachingASite) GetDownloadModel(domain.URL) port.DownloadModel {
	return port.DownloadModel{Tool: "a test", Producer: derived.Captions}
}
