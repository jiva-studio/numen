package absent_test

import (
	"reflect"
	"testing"
	"time"

	"github.com/jiva-studio/numen/libs/core/kernel/adapter/index"
	"github.com/jiva-studio/numen/libs/core/kernel/adapter/index/absent"
	"github.com/jiva-studio/numen/libs/core/kernel/domain/vault"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/testsupport"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/testsupport/indexfile"
	"github.com/jiva-studio/numen/libs/shared/note/domain"
)

func getHomeVault() domain.Vault { return domain.Vault{ID: "01VAULT0000000000000000000"} }

func getOtherVault() domain.Vault { return domain.Vault{ID: "01VAULT0000000000000000001"} }

func getGuide() vault.AbsentFile {
	return vault.AbsentFile{
		Path: "Books/Guide.pdf", Size: 5 << 20, Mtime: time.Date(2026, time.November, 20, 9, 30, 0, 0, time.UTC),
		Hash: "ab12", Provider: "tools-sync",
	}
}

func openRepository(t *testing.T) *absent.Repository {
	t.Helper()
	db, err := index.Open(t.Context(), indexfile.Path(t), index.Unsynchronized())
	if err != nil {
		t.Fatal(err)
	}
	testsupport.CloseAfter(t, db)
	return db.AbsentFiles()
}

func TestListAbsentFilesReturnsNothingForANewVault(t *testing.T) {
	files, err := openRepository(t).ListAbsentFiles(t.Context(), getHomeVault())
	if err != nil || len(files) != 0 {
		t.Fatalf("got %+v, %v", files, err)
	}
}

func TestPutAbsentFilesStoresEveryFieldOfAnEntry(t *testing.T) {
	repo := openRepository(t)
	if err := repo.PutAbsentFiles(t.Context(), getHomeVault(), []vault.AbsentFile{getGuide()}); err != nil {
		t.Fatal(err)
	}
	got, ok, err := repo.GetAbsentFile(t.Context(), getHomeVault(), "Books/Guide.pdf")
	if err != nil || !ok {
		t.Fatalf("got %+v, %v, %v", got, ok, err)
	}
	if want := getGuide(); !reflect.DeepEqual(got, want) {
		t.Fatalf("got %+v, want %+v", got, want)
	}
}

func TestPutAbsentFilesReplacesTheEntryOfAPath(t *testing.T) {
	repo := openRepository(t)
	newer := getGuide()
	newer.Size, newer.Hash = 7<<20, "cd34"
	for _, file := range []vault.AbsentFile{getGuide(), newer} {
		if err := repo.PutAbsentFiles(t.Context(), getHomeVault(), []vault.AbsentFile{file}); err != nil {
			t.Fatal(err)
		}
	}
	files, err := repo.ListAbsentFiles(t.Context(), getHomeVault())
	if err != nil || len(files) != 1 || files[0].Hash != "cd34" || files[0].Size != 7<<20 {
		t.Fatalf("got %+v, %v, want one entry with the newer hash", files, err)
	}
}

func TestAbsentFilesBelongToOneVault(t *testing.T) {
	repo := openRepository(t)
	if err := repo.PutAbsentFiles(t.Context(), getHomeVault(), []vault.AbsentFile{getGuide()}); err != nil {
		t.Fatal(err)
	}
	other, err := repo.ListAbsentFiles(t.Context(), getOtherVault())
	if err != nil || len(other) != 0 {
		t.Fatalf("the other vault got %+v, %v", other, err)
	}
	if err := repo.RemoveAbsentFiles(t.Context(), getOtherVault(), []string{"Books/Guide.pdf"}); err != nil {
		t.Fatal(err)
	}
	mine, err := repo.ListAbsentFiles(t.Context(), getHomeVault())
	if err != nil || len(mine) != 1 {
		t.Fatalf("a removal in the other vault left %+v, %v, want the entry", mine, err)
	}
}

func TestRemoveAbsentFilesDeletesOnlyTheNamedPaths(t *testing.T) {
	repo := openRepository(t)
	second := getGuide()
	second.Path = "Books/Atlas.pdf"
	if err := repo.PutAbsentFiles(t.Context(), getHomeVault(), []vault.AbsentFile{getGuide(), second}); err != nil {
		t.Fatal(err)
	}
	if err := repo.RemoveAbsentFiles(t.Context(), getHomeVault(), []string{"Books/Guide.pdf"}); err != nil {
		t.Fatal(err)
	}
	_, isGuide, err := repo.GetAbsentFile(t.Context(), getHomeVault(), "Books/Guide.pdf")
	if err != nil || isGuide {
		t.Fatalf("the removed entry is still there: %v, %v", isGuide, err)
	}
	_, isAtlas, err := repo.GetAbsentFile(t.Context(), getHomeVault(), "Books/Atlas.pdf")
	if err != nil || !isAtlas {
		t.Fatalf("the other entry is gone: %v, %v", isAtlas, err)
	}
}
