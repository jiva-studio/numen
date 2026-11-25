package plugin_test

import (
	"reflect"
	"testing"

	"github.com/jiva-studio/numen/libs/plugin/catalog/searchfunction"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/plugin"
	"github.com/jiva-studio/numen/libs/plugin/catalog/manifest"
	"github.com/jiva-studio/numen/libs/shared/note/domain"
)

func TestListContributions_ListsOnlyTheSectionsTheManifestDeclares(t *testing.T) {
	got := plugin.ListContributions(manifest.Manifest{
		ID:          "org.example.reader",
		Permissions: []string{"vault:read"},
		Contributes: manifest.Contributes{
			Readers: []manifest.Reader{{ID: "reader", ContentTypes: []domain.ContentType{"application/epub+zip", "text/plain"}}},
		},
	})

	want := []plugin.Contribution{
		{Kind: plugin.ContributionReaders, Items: []plugin.ContributionItem{{Text: "reader", Detail: "application/epub+zip, text/plain"}}},
		{Kind: plugin.ContributionPermissions, Items: []plugin.ContributionItem{{Text: "vault:read"}}},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("the contributions are %+v, want %+v", got, want)
	}
}

func TestListContributions_CommandHasTheKeysOfItsBinding(t *testing.T) {
	got := plugin.ListContributions(manifest.Manifest{Contributes: manifest.Contributes{
		Commands: []manifest.Command{{ID: "org.example.open", Title: "Open"}, {ID: "org.example.close"}},
		Keybindings: []manifest.Keybinding{
			{Key: "mod+o", Command: "org.example.open"},
			{Key: "mod+shift+p", Command: "workbench.palette"},
		},
	}})

	want := []plugin.Contribution{
		{Kind: plugin.ContributionCommands, Items: []plugin.ContributionItem{
			{Text: "Open", Detail: "org.example.open", Keys: "mod+o"},
			{Text: "org.example.close", Detail: "org.example.close"},
		}},
		{Kind: plugin.ContributionKeybindings, Items: []plugin.ContributionItem{
			{Text: "workbench.palette", Keys: "mod+shift+p"},
		}},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("the contributions are %+v, want %+v", got, want)
	}
}

func TestListContributions_TitleFallsBackToID(t *testing.T) {
	got := plugin.ListContributions(manifest.Manifest{Contributes: manifest.Contributes{
		TabKinds:        []manifest.TabKind{{Kind: "graph", Title: "Graph"}, {Kind: "list"}},
		ConnectionTypes: []manifest.ConnectionType{{ID: "openai"}},
		Operations:      []manifest.Operation{{Name: "llm.chat", Title: "chat"}},
		Licenses:        []manifest.License{{ID: "supporter", Title: "Supporter"}, {ID: "custom"}},
		Themes:          []manifest.Theme{{ID: "sea", Name: "Sea"}},
		Slots:           []manifest.SlotEntry{{Slot: "statusBar.right", View: "clock"}},
		Listeners:       []manifest.Listener{{Name: "mcp", PortSetting: "port"}, {Name: "fixed", Port: 8080}},
	}})

	want := []plugin.Contribution{
		{Kind: plugin.ContributionTabKinds, Items: []plugin.ContributionItem{
			{Text: "Graph", Detail: "graph"}, {Text: "list", Detail: "list"},
		}},
		{Kind: plugin.ContributionThemes, Items: []plugin.ContributionItem{{Text: "Sea", Detail: "sea"}}},
		{Kind: plugin.ContributionSlots, Items: []plugin.ContributionItem{{Text: "statusBar.right", Detail: "clock"}}},
		{Kind: plugin.ContributionListeners, Items: []plugin.ContributionItem{
			{Text: "mcp", Detail: "port"}, {Text: "fixed", Detail: "8080"},
		}},
		{Kind: plugin.ContributionConnectionTypes, Items: []plugin.ContributionItem{{Text: "openai", Detail: "openai"}}},
		{Kind: plugin.ContributionOperations, Items: []plugin.ContributionItem{{Text: "chat", Detail: "llm.chat"}}},
		{Kind: plugin.ContributionLicenses, Items: []plugin.ContributionItem{
			{Text: "Supporter", Detail: "supporter"}, {Text: "custom", Detail: "custom"},
		}},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("the contributions are %+v, want %+v", got, want)
	}
}

func TestListContributions_ListsNothingForAnEmptyManifest(t *testing.T) {
	if got := plugin.ListContributions(manifest.Manifest{ID: "org.example.empty"}); len(got) != 0 {
		t.Errorf("an empty manifest lists %+v", got)
	}
}

func TestListContributions_SearchFunctionsShowTheirSignature(t *testing.T) {
	got := plugin.ListContributions(manifest.Manifest{Contributes: manifest.Contributes{
		SearchFunctions: searchfunction.Set{Version: 1, Functions: []searchfunction.Function{{
			Name: "rhymes", Args: []searchfunction.Arg{{Name: "word", Type: searchfunction.ArgString}},
		}}},
	}})

	want := []plugin.Contribution{
		{Kind: plugin.ContributionSearchFunctions, Items: []plugin.ContributionItem{{Text: "rhymes", Detail: "rhymes(word: string)"}}},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("the contributions are %+v, want %+v", got, want)
	}
}
