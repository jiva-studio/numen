package service_test

import (
	"regexp"
	"slices"
	"strings"
	"testing"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/tools"
)

func listVaultTools(t *testing.T) tools.Listing {
	t.Helper()
	_, _, registry := newTwoVaults(t)
	listing, err := registry.ListTools(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	return listing
}

func joinToolInstructions(listing tools.Listing) string {
	var sentences []string
	for _, one := range listing.Tools {
		for _, sentence := range one.Instructions {
			if !slices.Contains(sentences, sentence) {
				sentences = append(sentences, sentence)
			}
		}
	}
	return strings.Join(sentences, " ")
}

func hasToolName(text, name string) bool {
	return regexp.MustCompile(`\b` + regexp.QuoteMeta(name) + `\b`).MatchString(text)
}

func isReadOnly(tool *tools.ToolInfo) bool {
	return tool.Effect == tools.ReadOnly
}

func TestTheVaultToolInstructionsSayHowToCiteAndArrange(t *testing.T) {
	instructions := joinToolInstructions(listVaultTools(t))

	for _, want := range []string{
		"[[",
		"[[note://",
		"file://",
		"`uri`",
		"numen:",
		"start=",
		"length=",
		"library_search",
		"note_search",
		"library_read",
		"mid-sentence",
		"kinds",
		`"file"`,
		"note_list",
		"note_rename",
		"vault_open",
	} {
		if !strings.Contains(instructions, want) {
			t.Errorf("the instructions do not contain %q:\n%s", want, instructions)
		}
	}
	if strings.Contains(instructions, "`at`") {
		t.Errorf("the instructions still name the address field `at`:\n%s", instructions)
	}
}

func TestEachVaultToolInstructionIsOneSentence(t *testing.T) {
	inside := regexp.MustCompile(`[.!?]\s+\S`)
	for _, tool := range listVaultTools(t).Tools {
		for _, sentence := range tool.Instructions {
			if strings.TrimSpace(sentence) != sentence || !strings.HasSuffix(sentence, ".") || inside.MatchString(sentence) ||
				strings.Contains(sentence, "\n") {
				t.Errorf("an instruction of %s is not one sentence: %q", tool.Name, sentence)
			}
		}
	}
}

func TestEveryVaultToolWithInstructionsNamesItself(t *testing.T) {
	for _, tool := range listVaultTools(t).Tools {
		if len(tool.Instructions) == 0 {
			continue
		}
		if !slices.ContainsFunc(tool.Instructions, func(sentence string) bool { return hasToolName(sentence, tool.Name) }) {
			t.Errorf("no instruction of %s names it: %q", tool.Name, tool.Instructions)
		}
	}
}

func TestReadOnlyToolInstructionsNameNoWritingTool(t *testing.T) {
	listing := listVaultTools(t)
	var writing []string
	for _, tool := range listing.Tools {
		if !isReadOnly(tool) {
			writing = append(writing, tool.Name)
		}
	}
	for _, name := range []string{
		"note_rename", "note_remove", "note_move", "link_add", "link_update", "link_remove",
		"vault_add", "vault_rename", "vault_remove", "vault_open",
	} {
		if !slices.Contains(writing, name) {
			t.Fatalf("%s is not among the writing tools %v", name, writing)
		}
	}

	for _, tool := range listing.Tools {
		if !isReadOnly(tool) {
			continue
		}
		for _, sentence := range tool.Instructions {
			for _, name := range writing {
				if hasToolName(sentence, name) {
					t.Errorf("an instruction of %s, a read-only tool, names %s: %q", tool.Name, name, sentence)
				}
			}
		}
	}
}

func TestTheReadOnlyToolsOfTheVaultHaveInstructions(t *testing.T) {
	listing := listVaultTools(t)
	var subset tools.Listing
	for _, tool := range listing.Tools {
		if isReadOnly(tool) {
			subset.Tools = append(subset.Tools, tool)
		}
	}

	instructions := joinToolInstructions(subset)
	for _, want := range []string{"library_search", "library_read", "note_search", "`uri`"} {
		if !strings.Contains(instructions, want) {
			t.Errorf("the instructions of the read-only tools do not contain %q:\n%s", want, instructions)
		}
	}
}
