package opentabs_test

import (
	"database/sql"
	_ "embed"
	"reflect"
	"testing"

	"github.com/jiva-studio/numen/libs/core/kernel/adapter/index/opentabs"
	windowdomain "github.com/jiva-studio/numen/libs/core/kernel/domain/window"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/testsupport"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/testsupport/indexfile"
)

func openObservedRepository(t *testing.T) (*opentabs.Repository, *sql.DB) {
	t.Helper()
	db, err := sql.Open("sqlite", indexfile.GetPath(t))
	if err != nil {
		t.Fatal(err)
	}
	testsupport.CloseAfter(t, db)
	return opentabs.NewRepository(db, db), db
}

func TestWorkspaceStateStoresPaneLayoutAndOrderedTabsTogether(t *testing.T) {
	repo, db := openObservedRepository(t)
	v := getFirstVault()
	const layout = `{"kind":"split","children":[{"id":"left"},{"id":"right"}]}`
	open := windowdomain.OpenTabs{
		Tabs:     []windowdomain.Tab{{ID: "a", State: `{"scroll":2}`}, {ID: "b"}, {ID: "c"}},
		ActiveID: "a", Layout: layout,
	}
	for _, state := range []windowdomain.OpenTabs{
		open,
		{Tabs: []windowdomain.Tab{open.Tabs[2], open.Tabs[0]}, ActiveID: "c", Layout: layout},
		{Layout: layout},
		{},
	} {
		if err := repo.WriteOpenTabs(t.Context(), v, state); err != nil {
			t.Fatal(err)
		}
		got, err := repo.ReadOpenTabs(t.Context(), v)
		if err != nil || !reflect.DeepEqual(got, state) {
			t.Fatalf("read %+v, %v, want %+v", got, err, state)
		}
		if state.Layout != "" {
			var kind string
			err := db.QueryRowContext(t.Context(), `SELECT json_type(value, '$.layout') FROM app_state WHERE vault = ? AND key = 'editor.workspace'`, v.ID).Scan(&kind)
			if err != nil || kind != "object" {
				t.Fatalf("saved layout has type %q, %v", kind, err)
			}
		}
	}
}

func TestAnIdenticalWorkspaceDoesNotUpdateAppState(t *testing.T) {
	repo, db := openObservedRepository(t)
	v := getFirstVault()
	open := windowdomain.OpenTabs{Tabs: []windowdomain.Tab{{ID: "a"}, {ID: "b"}}, ActiveID: "a", Layout: `{"pane":"first"}`}
	if err := repo.WriteOpenTabs(t.Context(), v, open); err != nil {
		t.Fatal(err)
	}
	const watch = `
		CREATE TABLE writes (key TEXT);
		CREATE TRIGGER record_update AFTER UPDATE ON app_state BEGIN INSERT INTO writes VALUES ('update'); END;
		CREATE TRIGGER record_delete AFTER DELETE ON app_state BEGIN INSERT INTO writes VALUES ('delete'); END;
		CREATE TRIGGER record_insert AFTER INSERT ON app_state BEGIN INSERT INTO writes VALUES ('insert'); END;`
	if _, err := db.ExecContext(t.Context(), watch); err != nil {
		t.Fatal(err)
	}
	for _, active := range []string{"a", "b", "b"} {
		open.ActiveID = active
		if err := repo.WriteOpenTabs(t.Context(), v, open); err != nil {
			t.Fatal(err)
		}
	}
	var writes string
	if err := db.QueryRowContext(t.Context(), "SELECT group_concat(key) FROM writes").Scan(&writes); err != nil || writes != "update" {
		t.Fatalf("workspace writes = %q, %v, want one update", writes, err)
	}
}

func TestAnInvalidLayoutKeepsThePreviousWorkspace(t *testing.T) {
	repo := openRepository(t)
	open := windowdomain.OpenTabs{Tabs: []windowdomain.Tab{{ID: "a"}}, ActiveID: "a", Layout: `{"pane":"first"}`}
	if err := repo.WriteOpenTabs(t.Context(), getFirstVault(), open); err != nil {
		t.Fatal(err)
	}
	if err := repo.WriteOpenTabs(t.Context(), getFirstVault(), windowdomain.OpenTabs{Layout: "{"}); err == nil {
		t.Fatal("invalid layout was accepted")
	}
	got, err := repo.ReadOpenTabs(t.Context(), getFirstVault())
	if err != nil || !reflect.DeepEqual(got, open) {
		t.Fatalf("read %+v, %v, want %+v", got, err, open)
	}
}

//go:embed sql/save.sql
var saveAppStateSQL string

func TestAppStateScopesTheSameKeyToEachVaultAndToTheApplication(t *testing.T) {
	_, db := openObservedRepository(t)
	ctx := t.Context()
	const key = "sample.key"
	for _, update := range []struct {
		vault any
		value string
	}{
		{nil, "1"}, {"first", "2"}, {"second", "3"},
		{nil, "4"}, {"first", "5"},
	} {
		if _, err := db.ExecContext(ctx, saveAppStateSQL, update.vault, key, update.value); err != nil {
			t.Fatal(err)
		}
	}
	for _, expected := range []struct {
		vault any
		value string
	}{{nil, "4"}, {"first", "5"}, {"second", "3"}} {
		var count int
		var value string
		err := db.QueryRowContext(ctx, "SELECT count(*), min(value) FROM app_state WHERE vault IS ? AND key = ?", expected.vault, key).Scan(&count, &value)
		if err != nil || count != 1 || value != expected.value {
			t.Errorf("vault %v has %d entries, value %q, %v; want one entry with %q", expected.vault, count, value, err, expected.value)
		}
	}
}
