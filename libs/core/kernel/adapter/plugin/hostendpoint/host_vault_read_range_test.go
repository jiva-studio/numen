package hostendpoint_test

import (
	"context"
	"fmt"
	vaultapi "github.com/jiva-studio/numen/libs/core/kernel/adapter/plugin/hostendpoint/vault"
	"net/http"
	"strings"
	"testing"

	"connectrpc.com/connect"

	v1 "github.com/jiva-studio/numen/libs/core/protocol/gen/numen/v1"
	"github.com/jiva-studio/numen/libs/core/protocol/gen/numen/v1/numenv1connect"

	"github.com/jiva-studio/numen/libs/note/domain"

	"github.com/jiva-studio/numen/libs/core/kernel/adapter/plugin/hostendpoint"
	"github.com/jiva-studio/numen/libs/core/kernel/adapter/plugin/hostvault"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/filesystem/vaultfixture"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/filesystem/vaultreader"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/filesystem/vaultwriter"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/plugin/transport"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/file"
)

// readerIdentity knows the one plugin that reads and grants it every permission.
type readerIdentity struct{}

func (readerIdentity) GetPluginIDByToken(token string) (string, bool) {
	return "md.numen.a", token == "token-a"
}

func (readerIdentity) CheckPluginPermission(string, string) error { return nil }

func openReadEndpoint(t *testing.T, access *vaultapi.Access) numenv1connect.HostVaultServiceClient {
	t.Helper()
	lent := hostendpoint.NewServices()
	lent.SetPluginVault(access)
	endpoint, err := hostendpoint.Open(t.Context(), hostendpoint.Address, hostendpoint.Callers{
		Identity: readerIdentity{}, Lent: lent,
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := endpoint.Close(context.WithoutCancel(t.Context())); err != nil {
			t.Error(err)
		}
	})
	url := fmt.Sprintf("http://127.0.0.1:%d", endpoint.GetPort())
	return numenv1connect.NewHostVaultServiceClient(http.DefaultClient, url, connect.WithInterceptors(transport.AuthInterceptor("token-a")))
}

func newDiskReadClient(t *testing.T) numenv1connect.HostVaultServiceClient {
	t.Helper()
	v := vaultfixture.NewVault(t, map[string]string{
		"Rivers.txt":      "the Danube flows east",
		"Nandu.txt":       "Ñandú runs along the Paraná",
		"gauge.log":       strings.Repeat("level 7\n", 2500),
		"sonar.bin":       "ab\xff\xfecd",
		"notes/Danube.md": "# Danube\n",
	})
	writers := vaultwriter.VaultWriters{Locks: vaultwriter.NewLocks()}
	readers := vaultreader.Readers{}
	return openReadEndpoint(t, hostvault.NewAccess(
		hostvault.Session{Current: func() (domain.Vault, bool) { return v, true }},
		hostvault.Files{
			Stat:        file.NewStatFile(readers),
			ReadFile:    file.NewReadFile(readers),
			RangeReader: file.Read{Readers: readers},
			OpenWriter:  func(v domain.Vault) (hostvault.NoteWriter, error) { return writers.Open(v) },
		},
		hostvault.Lookups{}, hostvault.Arranger{}, hostvault.Stores{},
	))
}

func readFileRange(t *testing.T, client numenv1connect.HostVaultServiceClient, path string, start, length int64) *v1.ReadFileResponse {
	t.Helper()
	res, err := client.ReadFile(t.Context(), connect.NewRequest(&v1.ReadFileRequest{Path: path, Start: start, Length: &length}))
	if err != nil {
		t.Fatalf("a ranged read of %s failed as a whole: %v", path, err)
	}
	return res.Msg
}

func TestRangedReadFileReturnsRangeTextStartAndFileStat(t *testing.T) {
	client := newDiskReadClient(t)

	got := readFileRange(t, client, "Rivers.txt", 4, 6)

	if got.GetError() != 0 || string(got.GetContent()) != "Danube" || got.GetStart() != 4 {
		t.Errorf("the range returned %v, want \"Danube\" at 4", got)
	}
	if got.GetStat().GetPath() != "Rivers.txt" || got.GetStat().GetSize() != int64(len("the Danube flows east")) || got.GetStat().GetModTime() == 0 {
		t.Errorf("the file stat is %v, want the file stat of Rivers.txt with its size", got.GetStat())
	}
}

func TestARangedReadFileMovesTheStartForwardToAWholeCharacter(t *testing.T) {
	client := newDiskReadClient(t)

	got := readFileRange(t, client, "Nandu.txt", 1, 4)

	if got.GetError() != 0 || string(got.GetContent()) != "and" || got.GetStart() != 2 {
		t.Errorf("the range returned %q at %d (error %v), want \"and\" at 2", got.GetContent(), got.GetStart(), got.GetError())
	}
}

func TestRangedReadFileOfZeroOrOver16000BytesReads16000(t *testing.T) {
	client := newDiskReadClient(t)

	for _, length := range []int64{0, 16001, 50000} {
		got := readFileRange(t, client, "gauge.log", 0, length)
		if got.GetError() != 0 || len(got.GetContent()) != 16000 || got.GetStat().GetSize() != 20000 {
			t.Errorf("a read of %d bytes returned %d bytes of a file of %d (error %v), want 16000 of 20000",
				length, len(got.GetContent()), got.GetStat().GetSize(), got.GetError())
		}
	}
}

func TestRangedReadFileOfNonUTF8BytesReturnsNotText(t *testing.T) {
	client := newDiskReadClient(t)

	got := readFileRange(t, client, "sonar.bin", 0, 6)

	if got.GetError() != v1.HostVaultErrorCode_HOST_VAULT_ERROR_CODE_NOT_TEXT || len(got.GetContent()) != 0 {
		t.Errorf("a range that is not UTF-8 returned %v, want NOT_TEXT and no content", got)
	}
}

func TestRangedReadFileOfFolderOrMissingPathReturnsNotFound(t *testing.T) {
	client := newDiskReadClient(t)

	for _, path := range []string{"notes", "Volga.txt"} {
		if got := readFileRange(t, client, path, 0, 10); got.GetError() != v1.HostVaultErrorCode_HOST_VAULT_ERROR_CODE_NOT_FOUND {
			t.Errorf("a ranged read of %s returned %v, want NOT_FOUND", path, got)
		}
	}
}

func TestReadFileWithoutALengthReturnsTheWholeFileAsBytes(t *testing.T) {
	client := newDiskReadClient(t)

	res, err := client.ReadFile(t.Context(), connect.NewRequest(&v1.ReadFileRequest{Path: "sonar.bin", Start: 3}))
	if err != nil {
		t.Fatal(err)
	}

	if res.Msg.GetError() != 0 || string(res.Msg.GetContent()) != "ab\xff\xfecd" || res.Msg.GetStart() != 0 || res.Msg.GetStat().GetSize() != 6 {
		t.Errorf("a read with no length returned %v, want the six bytes of sonar.bin from 0", res.Msg)
	}
}

func TestRangedReadFileWithoutAVaultReturnsNoVault(t *testing.T) {
	client := openReadEndpoint(t, nil)

	got := readFileRange(t, client, "Rivers.txt", 0, 10)

	if got.GetError() != v1.HostVaultErrorCode_HOST_VAULT_ERROR_CODE_NO_VAULT {
		t.Errorf("a ranged read with no vault returned %v, want NO_VAULT", got)
	}
}
