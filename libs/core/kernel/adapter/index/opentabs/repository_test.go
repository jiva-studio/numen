package opentabs_test

import (
	windowdomain "github.com/jiva-studio/numen/libs/core/kernel/domain/window"
	"reflect"
	"testing"

	"github.com/jiva-studio/numen/libs/core/kernel/adapter/index"
	"github.com/jiva-studio/numen/libs/core/kernel/adapter/index/opentabs"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/testsupport"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/testsupport/indexfile"
	"github.com/jiva-studio/numen/libs/note/domain"
)

func getFirstVault() domain.Vault { return domain.Vault{ID: "01VAULT0000000000000000000"} }

func getSecond() domain.Vault { return domain.Vault{ID: "01VAULT0000000000000000001"} }

func openRepository(t *testing.T) *opentabs.Repository {
	t.Helper()
	db, err := index.Open(t.Context(), indexfile.Path(t), index.Unsynchronized())
	if err != nil {
		t.Fatal(err)
	}
	testsupport.CloseAfter(t, db)
	return db.OpenTabs()
}

func TestReadOpenTabsReturnsNoneForANewVault(t *testing.T) {
	open, err := openRepository(t).ReadOpenTabs(t.Context(), getFirstVault())
	if err != nil || len(open.Tabs) != 0 || open.ActiveID != "" {
		t.Fatalf("got %+v, %v", open, err)
	}
}

func TestReadOpenTabsReturnsTabsInOrderWithStateAndFront(t *testing.T) {
	repo := openRepository(t)
	want := windowdomain.OpenTabs{
		Tabs: []windowdomain.Tab{
			{ID: "reader:viewer:a.bin", Kind: "reader:viewer", Path: "a.bin", Title: "a", State: `{"page":4}`},
			{ID: "graph:1", Kind: "graph"},
		},
		ActiveID: "reader:viewer:a.bin",
	}
	if err := repo.WriteOpenTabs(t.Context(), getFirstVault(), want); err != nil {
		t.Fatal(err)
	}
	got, err := repo.ReadOpenTabs(t.Context(), getFirstVault())
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %+v, want %+v", got, want)
	}
}

func TestWriteOpenTabsReplacesTheTabsOfOneVaultOnly(t *testing.T) {
	repo := openRepository(t)
	otherTabs := windowdomain.OpenTabs{Tabs: []windowdomain.Tab{{ID: "note:b.md", Kind: "note", Path: "b.md"}}}
	if err := repo.WriteOpenTabs(t.Context(), getSecond(), otherTabs); err != nil {
		t.Fatal(err)
	}
	if err := repo.WriteOpenTabs(t.Context(), getFirstVault(), windowdomain.OpenTabs{Tabs: []windowdomain.Tab{{ID: "x", Kind: "x"}}}); err != nil {
		t.Fatal(err)
	}
	if err := repo.WriteOpenTabs(t.Context(), getFirstVault(), windowdomain.OpenTabs{}); err != nil {
		t.Fatal(err)
	}
	gotFirst, err := repo.ReadOpenTabs(t.Context(), getFirstVault())
	if err != nil {
		t.Fatal(err)
	}
	if len(gotFirst.Tabs) != 0 {
		t.Errorf("first vault has %+v", gotFirst)
	}
	got, err := repo.ReadOpenTabs(t.Context(), getSecond())
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got, otherTabs) {
		t.Errorf("second vault has %+v", got)
	}
}
