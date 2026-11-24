package hostendpoint_test

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"os"
	"path/filepath"
	"testing"

	"connectrpc.com/connect"

	"github.com/jiva-studio/numen/libs/shared/note/domain"
	v1 "github.com/jiva-studio/numen/libs/shared/protocol/gen/numen/v1"

	"github.com/jiva-studio/numen/libs/core/kernel/adapter/index"
	"github.com/jiva-studio/numen/libs/core/kernel/adapter/plugin/hostendpoint"
	vaultapi "github.com/jiva-studio/numen/libs/core/kernel/adapter/plugin/hostendpoint/vault"
	"github.com/jiva-studio/numen/libs/core/kernel/adapter/plugin/hostvault"
	"github.com/jiva-studio/numen/libs/core/kernel/adapter/vaultabsent"
	pluginsdomain "github.com/jiva-studio/numen/libs/core/kernel/domain/plugin"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/filesystem/vaultreader"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/filesystem/vaulttest"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/filesystem/vaultwriter"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/testsupport"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/testsupport/indexfile"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/file"
)

const (
	guidePath    = "Books/Guide.pdf"
	guideContent = "the guide, page one"
)

func getGuideHash(content string) string {
	sum := sha256.Sum256([]byte(content))
	return hex.EncodeToString(sum[:])
}

// memoryStore is the plugin's folder in the store of a vault, held in memory.
type memoryStore struct {
	files map[string][]byte
	asked *string
}

func (s memoryStore) Read(_ context.Context, name string) ([]byte, error) {
	if content, ok := s.files[name]; ok {
		return content, nil
	}
	return nil, os.ErrNotExist
}

func (s memoryStore) List(context.Context) ([]pluginsdomain.StoreFile, error) { return nil, nil }

func (s memoryStore) Open(_ context.Context, name string) (io.ReadSeekCloser, int64, error) {
	content, ok := s.files[name]
	if !ok {
		return nil, 0, os.ErrNotExist
	}
	return readSeekCloser{bytes.NewReader(content)}, int64(len(content)), nil
}

type readSeekCloser struct{ *bytes.Reader }

func (readSeekCloser) Close() error { return nil }

// phoneVault is a vault on disk that lists absent files, served to a plugin.
type phoneVault struct {
	client hostendpoint.HostVaultClient
	root   string
	store  memoryStore
}

func newPhoneVault(t *testing.T, held map[string]string) phoneVault {
	t.Helper()
	v := vaulttest.NewVault(t, held)
	db, err := index.Open(t.Context(), indexfile.GetPath(t), index.Unsynchronized())
	if err != nil {
		t.Fatal(err)
	}
	testsupport.CloseAfter(t, db)
	registry := db.AbsentFiles()
	readers := vaultabsent.NewReaders(vaultreader.Readers{}, registry)
	writers := vaultwriter.VaultWriters{Locks: vaultwriter.NewLocks()}
	store := memoryStore{files: map[string][]byte{}, asked: new(string)}
	storeReaders := func(_ domain.Vault, pluginID string) (vaultapi.StoreReader, error) {
		*store.asked = pluginID
		return store, nil
	}
	noIndex := func(context.Context, domain.Vault, []string) error { return nil }
	access := hostvault.NewAccess(
		hostvault.Session{Current: func() (domain.Vault, bool) { return v, true }},
		hostvault.Files{
			Stat:        file.NewHashFile(readers, writers),
			ReadFile:    file.NewReadFile(readers, writers),
			RangeReader: file.Read{Readers: readers},
		},
		hostvault.Writer{Absent: hostvault.NewAbsent(file.NewAbsent(readers, registry, writers, noIndex), storeReaders)},
		hostvault.Lookups{}, hostvault.Arranger{}, hostvault.Stores{},
	)
	return phoneVault{client: openReadEndpoint(t, access), root: v.Path, store: store}
}

func (p phoneVault) put(t *testing.T, content string) *v1.PutAbsentFilesResponse {
	t.Helper()
	res, err := p.client.PutAbsentFiles(t.Context(), connect.NewRequest(&v1.PutAbsentFilesRequest{Files: []*v1.HostAbsentFile{{
		Path: guidePath, Size: int64(len(content)), ModifiedUnixMs: 1_700_000_000_123, Hash: getGuideHash(content),
	}}}))
	if err != nil {
		t.Fatal(err)
	}
	return res.Msg
}

func (p phoneVault) hasBytes() bool {
	_, err := os.Stat(filepath.Join(p.root, filepath.FromSlash(guidePath)))
	return err == nil
}

func TestAnAbsentFileIsListedAndRefusesAReadWithFileAbsent(t *testing.T) {
	p := newPhoneVault(t, nil)
	if res := p.put(t, guideContent); res.GetError() != 0 {
		t.Fatalf("the put was answered %v", res)
	}

	listed, err := p.client.ListAbsentFiles(t.Context(), connect.NewRequest(&v1.ListAbsentFilesRequest{}))
	if err != nil || len(listed.Msg.GetFiles()) != 1 || listed.Msg.GetFiles()[0].GetPath() != guidePath ||
		listed.Msg.GetFiles()[0].GetModifiedUnixMs() != 1_700_000_000_123 {
		t.Fatalf("the list was %v (%v)", listed.Msg, err)
	}
	read, err := p.client.ReadFile(t.Context(), connect.NewRequest(&v1.ReadFileRequest{Path: guidePath}))
	if err != nil || read.Msg.GetError() != v1.ErrorCode_ERROR_CODE_FILE_ABSENT {
		t.Errorf("a read of an absent file was answered %v (%v)", read.Msg, err)
	}
}

func TestAnAbsentFileIsRefusedWhereAFileOfOtherContentIsHeld(t *testing.T) {
	p := newPhoneVault(t, map[string]string{guidePath: "my own notes in a file"})

	if res := p.put(t, guideContent); res.GetError() != v1.ErrorCode_ERROR_CODE_FILE_PRESENT {
		t.Errorf("the put was answered %v, want FILE_PRESENT", res)
	}
}

func TestAFileIsFilledFromTheStoreOfThePluginThatCalls(t *testing.T) {
	p := newPhoneVault(t, nil)
	p.put(t, guideContent)
	p.store.files["guide.part"] = []byte(guideContent)

	res, err := p.client.FillFile(t.Context(), connect.NewRequest(&v1.FillFileRequest{Path: guidePath, StoreName: "guide.part"}))

	if err != nil || res.Msg.GetError() != 0 {
		t.Fatalf("the fill was answered %v (%v)", res.Msg, err)
	}
	if got, readErr := os.ReadFile(filepath.Join(p.root, filepath.FromSlash(guidePath))); readErr != nil || string(got) != guideContent {
		t.Errorf("the vault holds %q (%v)", got, readErr)
	}
	if *p.store.asked != "md.numen.a" {
		t.Errorf("the store of %q was opened, want the plugin that called", *p.store.asked)
	}
}

func TestAFillOfOtherBytesIsRefusedWithHashMismatch(t *testing.T) {
	p := newPhoneVault(t, nil)
	p.put(t, guideContent)
	p.store.files["guide.part"] = []byte("damaged")

	res, err := p.client.FillFile(t.Context(), connect.NewRequest(&v1.FillFileRequest{Path: guidePath, StoreName: "guide.part"}))

	if err != nil || res.Msg.GetError() != v1.ErrorCode_ERROR_CODE_HASH_MISMATCH {
		t.Errorf("the fill was answered %v (%v)", res.Msg, err)
	}
	if p.hasBytes() {
		t.Error("a file is at the path after a mismatch")
	}
}

func TestAFileIsEvictedOnlyWhileItHasTheHashNamed(t *testing.T) {
	p := newPhoneVault(t, map[string]string{guidePath: guideContent + ", then my notes"})

	changed, err := p.client.EvictFile(t.Context(), connect.NewRequest(&v1.EvictFileRequest{Path: guidePath, ExpectedHash: getGuideHash(guideContent)}))
	if err != nil || changed.Msg.GetError() != v1.ErrorCode_ERROR_CODE_CHANGED || !p.hasBytes() {
		t.Fatalf("the eviction of a changed file was answered %v (%v), bytes held: %v", changed.Msg, err, p.hasBytes())
	}
	kept, err := p.client.EvictFile(t.Context(), connect.NewRequest(&v1.EvictFileRequest{
		Path: guidePath, ExpectedHash: getGuideHash(guideContent + ", then my notes"),
	}))

	if err != nil || kept.Msg.GetError() != 0 || p.hasBytes() {
		t.Fatalf("the eviction was answered %v (%v), bytes held: %v", kept.Msg, err, p.hasBytes())
	}
	listed, err := p.client.ListAbsentFiles(t.Context(), connect.NewRequest(&v1.ListAbsentFilesRequest{}))
	if err != nil || len(listed.Msg.GetFiles()) != 1 {
		t.Errorf("the list after the eviction was %v (%v)", listed.Msg, err)
	}
}

func TestAHostWithoutAbsentFilesAnswersUnimplemented(t *testing.T) {
	_, client := newDiskVaults(t, nil)

	_, err := client.ListAbsentFiles(t.Context(), connect.NewRequest(&v1.ListAbsentFilesRequest{}))

	if connect.CodeOf(err) != connect.CodeUnimplemented {
		t.Errorf("got %v, want unimplemented", err)
	}
}
