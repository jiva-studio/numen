package container_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	format "github.com/jiva-studio/numen/modules/libs/core/cards"
	"github.com/jiva-studio/numen/modules/libs/core/container"
	"github.com/jiva-studio/numen/modules/libs/core/domain"
	"github.com/jiva-studio/numen/modules/libs/core/markdown"
)

// stencil is the awkward one the round trip is tested against: a key nobody
// owns, prose above the first face and prose under a face's heading.
const stencil = "---\n" +
	"id: 01J8F3K2M9QRSTVWXYZ012\n" +
	"type: stencil\n" +
	"mine: keep me verbatim\n" +
	"fields:\n" +
	"  - Name\n" +
	"  - Height\n" +
	"---\n" +
	"How I show these.\n" +
	"\n" +
	"## Recognise\n" +
	"\n" +
	"The one to start with.\n" +
	"\n" +
	"### Front\n" +
	"\n" +
	"{{Name}}\n" +
	"\n" +
	"### Back\n" +
	"\n" +
	"{{Height}}\n"

// A stencil read and written straight back comes out as the body it went in
// as. Anything less is a diff the person did not ask for, on every save,
// forever.
func TestReadingAStencilAndWritingItBackChangesNothing(t *testing.T) {
	for name, raw := range map[string]string{
		"the stencil": stencil,
		"crlf":        strings.ReplaceAll(stencil, "\n", "\r\n"),
		"no frontmatter": "Above them all.\n\n## Recognise\n\nThe one to start with.\n\n" +
			"### Front\n\n{{Name}}\n\n### Back\n\n{{Height}}\n",
		"no trailing break": "---\ntype: stencil\n---\n## Recognise\n\n### Front\n\n{{Name}}\n\n### Back\n\n{{Height}}",
		"no trailing break under a face's heading": "---\ntype: stencil\n---\n" +
			"## Recognise",
		"no faces": "---\ntype: stencil\n---\n",
		"bom":      "\xef\xbb\xbf---\ntype: stencil\n---\n## Recognise\n\n### Front\n\n{{Name}}\n\n### Back\n\n{{Height}}\n",
		"comments and order": "---\n# a note to myself\nzebra: 1\n\ntype: stencil\n---\n" +
			"## Recognise\n\n### Front\n\none\n\n\ntwo\n\n### Back\n\n{{Height}}\n",
		"a heading of another name": "---\ntype: stencil\n---\n" +
			"## Recognise\n\n### Front\n\n{{Name}}\n\n### Notes\n\nThe one to start with.\n\n" +
			"### Back\n\n{{Height}}\n\n### Afterwards\n\nAnd this.\n",
		"a side written twice": "---\ntype: stencil\n---\n" +
			"## Recognise\n\n### Front\n\n{{Name}}\n\n### Front\n\nthe second one\n\n### Back\n\n{{Height}}\n",
	} {
		t.Run(name, func(t *testing.T) {
			doc, err := markdown.Open([]byte(raw))
			if err != nil {
				t.Fatalf("open: %v", err)
			}
			read := format.ReadStencil(markdown.Parse(domain.FileRef{Path: "Animal.md"}, []byte(raw)))
			body, err := container.StencilBody(read.Preamble, read.Faces, read.Tail)
			if err != nil {
				t.Fatalf("stencil body: %v", err)
			}
			if want := markdown.Normalised(doc.Body()); body != want {
				t.Errorf("the round trip changed the stencil\n want %q\n  got %q", want, body)
			}
		})
	}
}

// vault is where the fixtures a person can open in the application stand.
const vault = "../../../../tests/vault/cards"

// Every deck and stencil of the fixture vault, read and written straight back,
// comes out as the body it went in as. These are the files a person is shown,
// awkward on purpose, and a save that touched nothing is a save that changed
// nothing.
func TestReadingTheFixtureVaultAndWritingItBackChangesNothing(t *testing.T) {
	entries, err := os.ReadDir(vault)
	if err != nil {
		t.Fatalf("read the vault: %v", err)
	}
	for _, entry := range entries {
		t.Run(entry.Name(), func(t *testing.T) {
			path := filepath.Join(vault, entry.Name())
			raw, err := os.ReadFile(path)
			if err != nil {
				t.Fatalf("read: %v", err)
			}
			doc, err := markdown.Open(raw)
			if err != nil {
				t.Fatalf("open: %v", err)
			}
			n := markdown.Parse(domain.FileRef{Path: path}, raw)

			var body string
			switch n.Type {
			case domain.TypeDeck:
				body, err = format.DeckBody(format.ReadDeck(n))
			case domain.TypeStencil:
				read := format.ReadStencil(n)
				body, err = container.StencilBody(read.Preamble, read.Faces, read.Tail)
			default:
				t.Fatalf("type = %q, want a deck or a stencil", n.Type)
			}
			if err != nil {
				t.Fatalf("body: %v", err)
			}
			if want := markdown.Normalised(doc.Body()); body != want {
				t.Errorf("the round trip changed the file\n want %q\n  got %q", want, body)
			}
		})
	}
}

// A write that reaches one face reaches nothing else: the prose above the first
// face and the prose under every other face's heading come out as the bytes
// they went in as.
func TestWritingOneFaceLeavesTheRestOfTheStencilAlone(t *testing.T) {
	raw := "---\ntype: stencil\n---\n" +
		"How I show these.\n\n" +
		"## Recognise\n\nThe one to start with.\n\n### Front\n\n{{Name}}\n\n### Back\n\n{{Height}}\n\n" +
		"## Name it\n\nAnd this one the other way round.\n\n### Front\n\n{{Height}}\n\n### Back\n\n{{Name}}\n"

	read := format.ReadStencil(markdown.Parse(domain.FileRef{Path: "Animal.md"}, []byte(raw)))
	held := append([]format.Face(nil), read.Faces...)
	held[0].Back = "**{{Height}}**"

	body, err := container.StencilBody(read.Preamble, held, read.Tail)
	if err != nil {
		t.Fatalf("stencil body: %v", err)
	}

	doc, err := markdown.Open([]byte(raw))
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	want := strings.Replace(markdown.Normalised(doc.Body()), "{{Height}}\n\n## Name it",
		"**{{Height}}**\n\n## Name it", 1)
	if body != want {
		t.Errorf("the write reached further than the face\n want %q\n  got %q", want, body)
	}
}

// A stencil shows a card once through each face it carries, and a face is not
// addressed by its name here. Two faces of one name are two faces, and what
// each holds is in the file.
func TestAStencilKeepsEveryFaceItIsGiven(t *testing.T) {
	body, err := container.StencilBody("", []format.Face{
		{Name: "Recognise", Front: "{{Name}}", Back: "{{Height}}"},
		{Name: "Recognise", Front: "{{Height}}", Back: "{{Name}}"},
	}, "")
	if err != nil {
		t.Fatalf("stencil body: %v", err)
	}

	read := format.ReadStencil(domain.Note{Body: body})
	if len(read.Faces) != 2 {
		t.Fatalf("faces = %+v, want both of them", read.Faces)
	}
	if read.Faces[0].Front != "{{Name}}" || read.Faces[0].Back != "{{Height}}" {
		t.Errorf("the first face = %+v", read.Faces[0])
	}
	if read.Faces[1].Front != "{{Height}}" || read.Faces[1].Back != "{{Name}}" {
		t.Errorf("the second face = %+v", read.Faces[1])
	}
}
