package source

import (
	"testing"

	"github.com/jiva-studio/numen/modules/libs/core/internal/urlfile"
	"github.com/jiva-studio/numen/modules/libs/notes/domain"

	"github.com/jiva-studio/numen/modules/libs/core/port/vaultport"
)

const (
	talkHash  = "talkhash"
	scanPath  = "scans/page.pdf"
	linkPath  = "links/talk.url"
	linkedURL = domain.URL("https://example.org/talk")
)

// textScene is a vault holding a recording the index says a model heard, a scan and a link note, over a shelf that holds what was made from them.
type textScene struct {
	texts   Texts
	shelf   *fakeStorage
	index   *store
	readers vaultport.VaultReaders
	vault   domain.Vault
}

func newTextScene(t *testing.T) textScene {
	t.Helper()
	library := newLibrary()
	library.hold(recordingPath, domain.KindRecording, []byte("a recording"), 1)
	library.hold(scanPath, domain.KindBook, []byte("a scan"), 1)
	library.hold(linkPath, domain.KindURL, urlfile.Write(linkedURL), 1)
	index := newStore()
	index.put(first.ID, domain.Source{
		Fingerprint: domain.Fingerprint{Path: recordingPath, Kind: domain.KindRecording},
		Producer:    domain.ASR, Hash: talkHash,
	})
	shelf := newShelf()
	readers := vaults{first.ID: library}
	return textScene{
		texts:   NewTexts(readers, nil, index, fakeStores{shelf}),
		shelf:   shelf,
		index:   index,
		readers: readers,
		vault:   first,
	}
}

// withoutIndex is the same scene with an index that knows nothing.
func (s textScene) withoutIndex() textScene {
	s.index = newStore()
	s.texts = NewTexts(s.readers, nil, s.index, fakeStores{s.shelf})
	return s
}

func (s textScene) write(t *testing.T, name domain.ArtifactName, content string) {
	t.Helper()
	if err := s.shelf.WriteArtifact(t.Context(), name, []byte(content)); err != nil {
		t.Fatal(err)
	}
}

func (s textScene) claim(t *testing.T, name domain.ArtifactName) {
	t.Helper()
	if _, err := s.shelf.ClaimArtifact(t.Context(), name); err != nil {
		t.Fatal(err)
	}
}
