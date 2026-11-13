package service_test

import (
	"slices"
	"testing"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/tools"
)

func getWriteToolArguments() map[string]string {
	return map[string]string{
		"note_rename":  `{"path":"Rocks/Basalt.md","title":"Flood Basalt"}`,
		"note_move":    `{"paths":["Rocks/Basalt.md"],"folder":"Archive"}`,
		"note_remove":  `{"paths":["Rocks/Basalt.md"]}`,
		"link_add":     `{"links":[{"from":"Rocks/Basalt.md","fingerprint":"1-1","address":"Granite","role":"jump"}]}`,
		"link_update":  `{"from":"Rocks/Basalt.md","address":"Granite","fingerprint":"1-1","label":"cousin"}`,
		"link_remove":  `{"from":"Rocks/Basalt.md","address":"Granite","fingerprint":"1-1"}`,
		"vault_add":    `{"path":"/vaults/meadow"}`,
		"vault_rename": `{"vault":"garden-vault","name":"Herbs"}`,
		"vault_remove": `{"vault":"garden-vault"}`,
		"vault_open":   `{"vault":"garden-vault"}`,
	}
}

func TestWriteToolsReadAndWriteTheVault(t *testing.T) {
	_, _, registry := newTwoVaults(t)
	infos, err := registry.ListTools(t.Context())
	if err != nil {
		t.Fatalf("list tools: %v", err)
	}
	effects := map[string]tools.Effect{}
	for _, info := range infos.Tools {
		effects[info.GetName()] = info.GetEffect()
	}
	for name := range getWriteToolArguments() {
		got, ok := effects[name]
		if !ok {
			t.Errorf("the tool %s is not listed", name)
			continue
		}
		if got != tools.ReadWrite {
			t.Errorf("the tool %s has the effect %v, want read-write", name, got)
		}
	}
}

func TestWriteToolsWithoutAVaultAreRejected(t *testing.T) {
	geology, _, registry := newTwoVaults(t)
	before, _ := geology.Content("Rocks/Basalt.md")
	for name, arguments := range getWriteToolArguments() {
		res, err := registry.InvokeTool(t.Context(), tools.Call{Name: name, Arguments: []byte(arguments)})
		if err != nil {
			t.Fatalf("call %s: %v", name, err)
		}
		if res.GetError() == tools.ErrorCodeUnknownTool {
			t.Errorf("call %s: the tool is unknown", name)
			continue
		}
		if !res.GetResult().GetIsError() || res.GetResult().GetText() != name+": the call names no vault" {
			t.Errorf("a call of %s naming no vault returned %v", name, res.GetResult())
		}
	}
	requireContent(t, geology, "Rocks/Basalt.md", before)
	if got := getVaultsByID(t, registry); len(got) != 2 {
		t.Errorf("vault_list gives %d vaults, want the 2 there were: %v", len(got), got)
	}
	if opened, ok := geology.GetOpenedVault(); ok {
		t.Errorf("the vault %s was opened", opened.ID)
	}
}

func TestTheInstructionsOfEveryWriteToolNameIt(t *testing.T) {
	listing := listVaultTools(t)
	byTool := map[string][]string{}
	for _, tool := range listing.Tools {
		byTool[tool.Name] = tool.Instructions
	}
	for name := range getWriteToolArguments() {
		if !slices.ContainsFunc(byTool[name], func(sentence string) bool { return hasToolName(sentence, name) }) {
			t.Errorf("no instruction of %s names it: %q", name, byTool[name])
		}
	}
}
