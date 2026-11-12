package connection_test

import (
	"errors"
	"reflect"
	"slices"
	"testing"

	connectiondomain "github.com/jiva-studio/numen/libs/core/kernel/domain/connection"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/testsupport/connectiontest"

	"github.com/jiva-studio/numen/libs/core/kernel/usecase/connection"
)

func proofreading() connectiondomain.Use {
	return connectiondomain.Use{PluginID: "md.numen.tools.proofreading", PluginName: "Proofreading", Operations: []string{"llm.complete"}}
}

func embeddings() connectiondomain.Use {
	return connectiondomain.Use{PluginID: "md.numen.engines.embeddings", PluginName: "Embeddings", Operations: []string{"llm.embed"}}
}

func TestListReturnsTheUsesOfEachConnection(t *testing.T) {
	f := newFixture(t)
	work := f.createOpenRouter(t, "Work")
	home := f.createCompatible(t, "Ollama", "http://localhost:11434/v1", "")
	f.uses.use(work.ID, proofreading(), embeddings())

	entries := f.listEntries(t)

	if !reflect.DeepEqual(entries[0].Uses, []connectiondomain.Use{proofreading(), embeddings()}) {
		t.Errorf("uses of %s = %v", work.Name, entries[0].Uses)
	}
	if len(entries[1].Uses) != 0 {
		t.Errorf("uses of %s = %v, want none", home.Name, entries[1].Uses)
	}
}

func TestDeleteRemovesTheRowsThatNameTheConnectionAndReturnsWhereTheyWere(t *testing.T) {
	f := newFixture(t)
	work := f.createOpenRouter(t, "Work")
	f.uses.use(work.ID, proofreading(), embeddings())

	removed, err := f.connection.Delete.Execute(t.Context(), work.ID)

	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(removed, []connectiondomain.Use{proofreading(), embeddings()}) {
		t.Errorf("removed uses = %v", removed)
	}
	if !slices.Equal(f.uses.removed, []string{work.ID}) {
		t.Errorf("rows removed for %v, want %s", f.uses.removed, work.ID)
	}
}

func TestDeleteOfAnUnknownConnectionRemovesNoRows(t *testing.T) {
	f := newFixture(t)

	_, err := f.connection.Delete.Execute(t.Context(), "c0000000f")

	if !errors.Is(err, connectiondomain.ErrNotFound) || len(f.uses.removed) != 0 {
		t.Errorf("err = %v, rows removed for %v", err, f.uses.removed)
	}
}

func TestDeleteRemovesNoRowsWhenASecretCannotBeRemoved(t *testing.T) {
	f := newFixture(t)
	work := f.createOpenRouter(t, "Work")
	f.uses.use(work.ID, proofreading())
	f.connection = connectiontest.New(f.repository, f.types, &failingDelete{memoryKeys: f.keys}, f.uses, nil)

	_, err := f.connection.Delete.Execute(t.Context(), work.ID)

	if !errors.Is(err, connectiondomain.ErrKeyStore) || len(f.uses.removed) != 0 {
		t.Errorf("err = %v, rows removed for %v", err, f.uses.removed)
	}
}

func TestDeleteReportsRowsThatCannotBeRemoved(t *testing.T) {
	f := newFixture(t)
	work := f.createOpenRouter(t, "Work")
	f.uses.use(work.ID, proofreading())
	f.uses.failRemove = errDisk

	removed, err := f.connection.Delete.Execute(t.Context(), work.ID)

	if !errors.Is(err, errDisk) {
		t.Errorf("err = %v, want the failure to write the rows", err)
	}
	if !reflect.DeepEqual(removed, []connectiondomain.Use{proofreading()}) {
		t.Errorf("removed uses = %v, want the settings that still name the connection", removed)
	}
}

func TestUpdateSendsTheConfigurationAgainToThePluginsThatUseTheConnection(t *testing.T) {
	f := newFixture(t)
	work := f.createOpenRouter(t, "Work")

	if _, err := f.connection.Update.Execute(t.Context(), connection.UpdateRequest{ID: work.ID, Revision: work.Revision, Name: "Work 2"}); err != nil {
		t.Fatal(err)
	}

	if !slices.Equal(f.uses.refreshed, []string{work.ID}) {
		t.Errorf("refreshed for %v, want %s once", f.uses.refreshed, work.ID)
	}
}

func TestAnUpdateThatChangesNothingRefreshesNothing(t *testing.T) {
	f := newFixture(t)
	work := f.createOpenRouter(t, "Work")

	_, err := f.connection.Update.Execute(t.Context(), connection.UpdateRequest{ID: work.ID, Revision: work.Revision + 5, Name: "Work 2"})

	if !errors.Is(err, connectiondomain.ErrStaleRevision) || len(f.uses.refreshed) != 0 {
		t.Errorf("err = %v, refreshed for %v", err, f.uses.refreshed)
	}
}

func TestUpdateReportsAConfigurationThatCannotBeSent(t *testing.T) {
	f := newFixture(t)
	work := f.createOpenRouter(t, "Work")
	f.uses.failRefresh = errDisk

	_, err := f.connection.Update.Execute(t.Context(), connection.UpdateRequest{ID: work.ID, Revision: work.Revision, Name: "Work 2"})

	if !errors.Is(err, errDisk) {
		t.Errorf("err = %v, want the failure to send the configuration", err)
	}
	if stored := f.listStoredConnections(t); stored[0].Name != "Work 2" {
		t.Errorf("the change was not kept: %+v", stored)
	}
}
