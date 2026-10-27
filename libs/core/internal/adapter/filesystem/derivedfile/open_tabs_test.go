package derivedfile_test

import (
	"reflect"
	"testing"
	"time"

	"github.com/jiva-studio/numen/libs/notes/domain"

	"github.com/jiva-studio/numen/libs/core/internal/adapter/filesystem/derivedfile"
	"github.com/jiva-studio/numen/libs/core/internal/adapter/filesystem/vaultconfig"
)

func openVault(t *testing.T) domain.Vault {
	t.Helper()
	root := t.TempDir()
	if _, err := vaultconfig.Initialize(root, vaultconfig.DefaultServiceDir, time.Now()); err != nil {
		t.Fatal(err)
	}
	return domain.Vault{ID: "v", Path: root}
}

func TestReadOpenTabsReturnsNoneForANewVault(t *testing.T) {
	open, err := derivedfile.OpenTabsFiles{}.ReadOpenTabs(t.Context(), openVault(t))
	if err != nil || len(open.Tabs) != 0 || open.FrontID != "" {
		t.Fatalf("got %+v, %v", open, err)
	}
}

func TestReadOpenTabsReturnsSavedTabsWithState(t *testing.T) {
	v := openVault(t)
	files := derivedfile.OpenTabsFiles{}
	want := domain.OpenTabs{
		Tabs: []domain.Tab{
			{ID: "reader:viewer:a.bin", Kind: "reader:viewer", Path: "a.bin", Title: "a", State: `{"page":4}`},
			{ID: "graph:1", Kind: "graph"},
		},
		FrontID: "reader:viewer:a.bin",
	}
	if err := files.WriteOpenTabs(t.Context(), v, want); err != nil {
		t.Fatal(err)
	}
	got, err := files.ReadOpenTabs(t.Context(), v)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %+v, want %+v", got, want)
	}
}
