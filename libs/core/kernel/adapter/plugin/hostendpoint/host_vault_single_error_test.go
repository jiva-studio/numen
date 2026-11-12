package hostendpoint

import (
	"context"
	"errors"
	"fmt"
	vaultapi "github.com/jiva-studio/numen/libs/core/kernel/adapter/plugin/hostendpoint/vault"
	"io/fs"
	"net/http"
	"net/http/httptest"
	"strings"
	"syscall"
	"testing"

	"connectrpc.com/connect"

	v1 "github.com/jiva-studio/numen/libs/core/protocol/gen/numen/v1"
	"github.com/jiva-studio/numen/libs/core/protocol/gen/numen/v1/numenv1connect"
	"github.com/jiva-studio/numen/libs/plugin/catalog/manifest"

	"github.com/jiva-studio/numen/libs/note/domain"
	"github.com/jiva-studio/numen/libs/note/edit"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/vault"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/plugin/transport"
)

// failingFileVault is a vault whose Write and Create fail with err.
type failingFileVault struct {
	*memoryVault
	err error
}

func (f failingFileVault) WriteFile(context.Context, string, []byte, domain.FileStat) (domain.FileStat, error) {
	return domain.FileStat{}, f.err
}

func (f failingFileVault) CreateFile(context.Context, string, []byte) error {
	return f.err
}

func newFailingFileClient(t *testing.T, err error) numenv1connect.HostVaultServiceClient {
	t.Helper()
	hosted := newCallers(nil)
	hosted.vault = vaultOf(failingFileVault{memoryVault: newMemoryVault(map[string]string{"Ganges.md": "# Ganges\n"}), err: err})
	return newVaultClient(t, hosted, "token-a")
}

func getInternalMessage(t *testing.T, err error) string {
	t.Helper()
	var connectErr *connect.Error
	if !errors.As(err, &connectErr) {
		t.Fatalf("the call returned %v, want a connect error with code internal", err)
	}
	if connectErr.Code() != connect.CodeInternal {
		t.Fatalf("the call failed with code %v (%v), want internal", connectErr.Code(), err)
	}
	return connectErr.Message()
}

func TestCreateFileOverAFileInThePathReturnsNotADirectory(t *testing.T) {
	client := newFailingFileClient(t, &fs.PathError{Op: "open", Path: "a.md/x.txt", Err: syscall.ENOTDIR})

	res, err := client.CreateFile(t.Context(), connect.NewRequest(&v1.CreateFileRequest{Path: "a.md/x.txt", Content: []byte("text")}))

	if err == nil {
		t.Fatalf("CreateFile returned %v and no error, want an internal error that says not a directory", res.Msg)
	}
	if message := getInternalMessage(t, err); !strings.Contains(message, "not a directory") {
		t.Errorf("the error message is %q, want it to contain %q", message, "not a directory")
	}
}

func TestWriteFileWithANoteErrorReturnsInternalWithTheErrorText(t *testing.T) {
	for name, cause := range map[string]error{
		"too large":       edit.ErrTooLarge,
		"unreadable":      edit.ErrUnreadable,
		"body unwritable": edit.ErrBodyUnwritable,
	} {
		t.Run(name, func(t *testing.T) {
			failure := fmt.Errorf("write Ganges.md: %w", cause)
			client := newFailingFileClient(t, failure)

			res, err := client.WriteFile(t.Context(), connect.NewRequest(&v1.WriteFileRequest{Path: "Ganges.md", Content: []byte("# Ganga\n")}))

			if err == nil {
				t.Fatalf("WriteFile returned %v and no error, want an internal error with the text %q", res.Msg, failure)
			}
			if message := getInternalMessage(t, err); !strings.Contains(message, failure.Error()) {
				t.Errorf("the error message is %q, want it to contain %q", message, failure.Error())
			}
		})
	}
}

func TestCreateFileOverAnExistingFileReturnsAlreadyExists(t *testing.T) {
	client := newFailingFileClient(t, fmt.Errorf("create Ganges.md: %w", vault.ErrAlreadyExists))

	res, err := client.CreateFile(t.Context(), connect.NewRequest(&v1.CreateFileRequest{Path: "Ganges.md", Content: []byte("text")}))
	if err != nil {
		t.Fatal(err)
	}

	if res.Msg.GetError() != v1.HostVaultErrorCode_HOST_VAULT_ERROR_CODE_ALREADY_EXISTS {
		t.Errorf("CreateFile returned %v, want error ALREADY_EXISTS", res.Msg)
	}
}

const tooLargePath = "Danube.md"

// tooLargeMoveVault is the vault of rivers whose move of Danube.md fails with edit.ErrTooLarge.
type tooLargeMoveVault struct{ *fakeNoteVault }

func (v tooLargeMoveVault) MoveNote(ctx context.Context, from, to string) (domain.MoveResult, error) {
	if from == tooLargePath {
		return domain.MoveResult{From: from, To: to}, fmt.Errorf("move %s: %w", from, edit.ErrTooLarge)
	}
	return v.fakeNoteVault.MoveNote(ctx, from, to)
}

func TestMoveNotesReturnsTooLargeForTheNoteThatIsTooLarge(t *testing.T) {
	hostCallers := newCallers(nil)
	hostCallers.vault = vaultOf(tooLargeMoveVault{newFakeNoteVault()})
	endpoint := &Endpoint{callers: hostCallers.over(nil, manifest.PermissionVaultWrite)}
	mux := http.NewServeMux()
	mux.Handle(vaultapi.NewHandler(endpoint, endpoint.callers.Lent))
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)
	client := numenv1connect.NewHostVaultServiceClient(server.Client(), server.URL, connect.WithInterceptors(transport.AuthInterceptor("token-a")))

	res, err := client.MoveNotes(t.Context(), connect.NewRequest(&v1.HostVaultServiceMoveNotesRequest{
		Paths: []string{tooLargePath, "Rivers/Rhine.md"}, Folder: "Archive",
	}))
	if err != nil {
		t.Fatalf("the move failed as a whole: %v", err)
	}

	moves := res.Msg.GetMoves()
	if res.Msg.GetError() != 0 || len(moves) != 2 {
		t.Fatalf("the move returned error %v with %d entries, want 2 entries", res.Msg.GetError(), len(moves))
	}
	if m := moves[0]; m.GetError() != v1.HostVaultErrorCode_HOST_VAULT_ERROR_CODE_TOO_LARGE {
		t.Errorf("the entry of %s is %v, want error TOO_LARGE", tooLargePath, m)
	}
	if m := moves[1]; m.GetError() != 0 || m.GetMove().GetTo() == "" {
		t.Errorf("the entry of Rivers/Rhine.md is %v, want it moved with no error", m)
	}
}
