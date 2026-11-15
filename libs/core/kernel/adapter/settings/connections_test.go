package settings_test

import (
	"errors"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jiva-studio/numen/libs/core/kernel/adapter/settings"
	"github.com/jiva-studio/numen/libs/core/kernel/domain/connection"
	settingsdomain "github.com/jiva-studio/numen/libs/core/kernel/domain/settings"
	"github.com/jiva-studio/numen/libs/plugin/catalog/manifest"
)

const validConnectionsSection = `"connections":[{"id":"c0a1b2c3d","type":"md.numen.connections.llm/openrouter","name":"OpenRouter",` +
	`"revision":3,"fields":{"parallelRequests":4}}]`

func newConnection(id, name string) connection.Connection {
	return connection.Connection{ID: id, TypeID: "md.numen.connections.llm/openrouter", Name: name, Revision: 1, Fields: map[string]any{}}
}

func TestConnectionsIsNotAPluginId(t *testing.T) {
	if manifest.IsValidID("connections") {
		t.Error("connections is a valid plugin id, so the section would clash with a plugin")
	}
}

func TestNoOwnerDeclaresTheConnectionsSection(t *testing.T) {
	for _, path := range [][]string{{"connections"}, {"connections", "0", "name"}} {
		if err := check(path, "x"); !errors.Is(err, settingsdomain.ErrUndeclared) {
			t.Errorf("a write at %v gave %v, want ErrUndeclared", path, err)
		}
	}
}

func TestAConnectionIsReadFromTheFile(t *testing.T) {
	cfg, err := openAt(write(t, `{`+validConnectionsSection+`}`))
	if err != nil {
		t.Fatal(err)
	}

	if len(cfg.Connections) != 1 || len(cfg.Notices) != 0 {
		t.Fatalf("connections = %+v, notices = %q", cfg.Connections, cfg.Notices)
	}
	one := cfg.Connections[0]
	if one.ID != "c0a1b2c3d" || one.TypeID != "md.numen.connections.llm/openrouter" || one.Name != "OpenRouter" || one.Revision != 3 {
		t.Errorf("connection = %+v", one)
	}
	if one.Fields["parallelRequests"] != float64(4) {
		t.Errorf("fields = %+v", one.Fields)
	}
}

func TestAMalformedConnectionsSectionIsReadWithANotice(t *testing.T) {
	for what, body := range map[string]string{
		"a string":                  `{"connections":"oops"}`,
		"a list of numbers":         `{"connections":[1,2]}`,
		"an object":                 `{"connections":{"a":1}}`,
		"entries of the wrong type": `{"connections":[{"id":7,"type":[],"name":{},"revision":"x"}]}`,
	} {
		t.Run(what, func(t *testing.T) {
			cfg, err := openAt(write(t, body))
			if err != nil {
				t.Fatalf("the file was refused: %v", err)
			}
			if len(cfg.Connections) != 0 || !hasNotice(cfg.Notices, "connections") {
				t.Errorf("connections = %+v, notices = %q", cfg.Connections, cfg.Notices)
			}
			if hasNotice(cfg.Notices, "plugin id") {
				t.Errorf("the section was read as a plugin: %q", cfg.Notices)
			}
		})
	}
}

func TestInvalidConnectionsAreLeftOutOfForceAndTheFileIsNotTouched(t *testing.T) {
	body := `{"connections":[` +
		`{"id":"c0a1b2c3d","type":"a.b/x","name":"Good","revision":1},` +
		`{"id":"bad","type":"a.b/x","name":"Bad id","revision":1},` +
		`{"id":"c0a1b2c3d","type":"a.b/x","name":"Repeated id","revision":1},` +
		`{"id":"c11111111","type":"a.b/x","name":" padded ","revision":1},` +
		`{"id":"c22222222","type":"a.b/x","name":"good","revision":1},` +
		`{"id":"c33333333","type":"","name":"No type","revision":1},` +
		`{"id":"c44444444","type":"a.b/x","name":"Revision zero","revision":0}]}`
	path := write(t, body)
	cfg, err := openAt(path)
	if err != nil {
		t.Fatal(err)
	}

	cfg.DropInvalidConnections()

	if len(cfg.Connections) != 1 || cfg.Connections[0].Name != "Good" {
		t.Errorf("connections = %+v", cfg.Connections)
	}
	if len(cfg.Notices) != 6 {
		t.Errorf("notices = %q, want one for each of the six invalid entries", cfg.Notices)
	}
	if readFile(t, path) != body {
		t.Error("the bytes of the file changed")
	}
}

func TestAConnectionWhoseTypeIsNotInstalledOrWhoseFieldIsNotDeclaredStaysInForce(t *testing.T) {
	cfg, err := openAt(write(t, `{"connections":[{"id":"c0a1b2c3d","type":"gone.plugin/x","name":"Old","revision":2,"fields":{"removedField":"v"}}]}`))
	if err != nil {
		t.Fatal(err)
	}

	cfg.DropInvalidConnections()

	if len(cfg.Connections) != 1 || cfg.Connections[0].Fields["removedField"] != "v" || len(cfg.Notices) != 0 {
		t.Errorf("connections = %+v, notices = %q", cfg.Connections, cfg.Notices)
	}
}

func TestChangeConnectionsPreservesOtherSections(t *testing.T) {
	const before = "{\n  \"app\":   {\"appearance\": {\"mode\": \"dark\"}},\n  \"md.numen.alpha\": {\"model\": \"small\"}\n}\n"
	path := write(t, before)

	err := settings.NewFiles().ChangeConnections(path, func(list []connection.Connection) ([]connection.Connection, error) {
		one := newConnection("c0a1b2c3d", "Work")
		one.Fields = map[string]any{"parallelRequests": 2}
		return append(list, one), nil
	})
	if err != nil {
		t.Fatal(err)
	}

	after := readFile(t, path)
	if !strings.Contains(after, `"id":"c0a1b2c3d"`) || !strings.Contains(after, `"revision":1`) {
		t.Errorf("the section is missing from:\n%s", after)
	}
	if !strings.Contains(after, "\"app\":   {\"appearance\": {\"mode\": \"dark\"}}") || !strings.Contains(after, `"md.numen.alpha": {"model": "small"}`) {
		t.Errorf("the rest of the file changed:\n%s", after)
	}
	cfg, err := openAt(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(cfg.Connections) != 1 || cfg.Connections[0].Name != "Work" || cfg.Connections[0].Fields["parallelRequests"] != float64(2) {
		t.Errorf("read back %+v", cfg.Connections)
	}
}

func TestChangeConnectionsSkipsTheWriteOnFailureOrAnUnchangedList(t *testing.T) {
	path := write(t, `{`+validConnectionsSection+`}`)
	before := readFile(t, path)
	errRefused := errors.New("refused")

	if err := settings.NewFiles().ChangeConnections(path, func([]connection.Connection) ([]connection.Connection, error) {
		return nil, errRefused
	}); !errors.Is(err, errRefused) {
		t.Errorf("error = %v", err)
	}
	if err := settings.NewFiles().ChangeConnections(path, func(list []connection.Connection) ([]connection.Connection, error) {
		return list, nil
	}); err != nil {
		t.Fatal(err)
	}
	if after := readFile(t, path); after != before {
		t.Errorf("the file changed:\n%s", after)
	}
}

func TestChangeConnectionsCreatesTheFileAndRefusesInvalidJSON(t *testing.T) {
	missing := filepath.Join(t.TempDir(), "numen.json")
	add := func(list []connection.Connection) ([]connection.Connection, error) {
		return append(list, newConnection("c0a1b2c3d", "Work")), nil
	}
	if err := settings.NewFiles().ChangeConnections(missing, add); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(readFile(t, missing), "c0a1b2c3d") {
		t.Error("the file was not created with the connection")
	}

	broken := write(t, `{"connections": [`)
	if err := settings.NewFiles().ChangeConnections(broken, add); err == nil {
		t.Error("a file that is not JSON was written")
	}
}

func TestConnectionsKeepTheirOrder(t *testing.T) {
	path := write(t, `{}`)
	files := settings.NewFiles()
	for _, id := range []string{"c00000001", "c00000002", "c00000003"} {
		if err := files.ChangeConnections(path, func(list []connection.Connection) ([]connection.Connection, error) {
			return append(list, newConnection(id, id)), nil
		}); err != nil {
			t.Fatal(err)
		}
	}

	cfg, err := openAt(path)
	if err != nil {
		t.Fatal(err)
	}
	var ids []string
	for _, one := range cfg.Connections {
		ids = append(ids, one.ID)
	}
	if strings.Join(ids, ",") != "c00000001,c00000002,c00000003" {
		t.Errorf("order = %v", ids)
	}
}
