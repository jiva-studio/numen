package mcp

import (
	"fmt"
	"strings"
)

// namingOrder is how a note comes by the name it is shown under. The
// instructions and the tool that changes it say it in these words.
const namingOrder = "A note is shown by its title, else by its filename."

// instructions is what an agent is told once, before it calls anything.
func instructions(core Core) string {
	var b strings.Builder
	opening(&b, core)

	b.WriteString("What is worth knowing before changing anything:\n")
	b.WriteString("- " + namingOrder + " `note_rename` brings whichever of the three names it ")
	b.WriteString("into line and files the note under the new name.\n")
	b.WriteString("- Name a note you speak about as a link, so the person can go to it: ")
	b.WriteString("`[[Harmonic oscillator]]`, the title they know it by. It reaches the note ")
	b.WriteString("the way every link in this vault does, and a name no note answers to is ")
	b.WriteString("drawn as reaching nothing. Write it where you speak about the note, not ")
	b.WriteString("in a list at the end. A title several notes share is written as ")
	b.WriteString("`[[note://<identifier>]]`, which names one of them.\n")
	b.WriteString("- A link written as a name finds its note wherever it moves to, so moving ")
	b.WriteString("notes between folders is safe and does not need links rewritten.\n")
	b.WriteString("- Two notes filed under one name make every link written by that name ")
	b.WriteString("ambiguous. `vault_problems` lists them.\n")
	b.WriteString("- Folders are for the person's convenience. The hierarchy the product ")
	b.WriteString("draws is the parent and child links, not the folder tree.\n")
	b.WriteString("- Prose is yours to write; the frontmatter is the person's. Change it with ")
	b.WriteString("the link and rename tools rather than by writing the file yourself.\n")
	b.WriteString("- A removed note goes to the vault's trash rather than being destroyed.\n")
	b.WriteString("- Any file the vault holds is read by its path with `file_read`, a run at ")
	b.WriteString("a time. The tools here write notes, so a file of another kind is read and ")
	b.WriteString("not written.\n\n")

	b.WriteString("A vault holds books and papers beside its notes, and asking them is not ")
	b.WriteString("like asking a note:\n")
	booksAreNotNotes(&b)
	b.WriteString("- A search answers with several places of one book. Show every place you ")
	b.WriteString("speak about: `source_focus` takes the rest under `also`, and the person is ")
	b.WriteString("taken to the first.\n")
	b.WriteString("- Every passage you speak about is written as a link, and one written ")
	b.WriteString("about without a link is one the person cannot go to. The search hands ")
	b.WriteString("you the address of each passage under `at`: put it in a markdown link ")
	b.WriteString("where you speak about the passage, not in a list at the end.\n")
	b.WriteString("  `[the Remuna episode](numen:library%2FA%20Book.pdf?start=62690&length=1246)`\n\n")

	b.WriteString("Changes appear immediately in the window the person has open, so work in ")
	b.WriteString("small steps they can follow.\n")
	return b.String()
}

// opening is what every agent is told first: what a vault is, where this one
// is, and how a note is addressed. A note has one address, and an agent that
// joins it to the root pays for the root once.
func opening(b *strings.Builder, core Core) {
	b.WriteString("These tools work on one numen vault: markdown notes in an ordinary folder, ")
	b.WriteString("joined by typed links into a graph.\n\n")

	shown := core.getShownVault()
	fmt.Fprintf(b, "The vault %q is at %s on this machine.\n", shown.Vault.Name, shown.Root)
	b.WriteString("Every note is addressed by its path relative to that folder, with forward ")
	b.WriteString("slashes — `notes/entropy.md`. That path is what every tool takes and returns. ")
	b.WriteString("To open a note as a file, join it to the folder above; if you cannot read ")
	b.WriteString("files, `note_read` gives you the same text.\n\n")

	if core.Attending != nil {
		b.WriteString("What the person has open is `window_tab_list`: every tab of their window, ")
		b.WriteString("and which of them they are looking at. Ask it before saying anything ")
		b.WriteString("about what is in front of them, and ask again when it matters — they ")
		b.WriteString("move between tabs while you work.\n\n")
	}
}

// readingInstructions is what an agent served the reading tools is told once,
// before it calls anything.
func readingInstructions(core Core) string {
	var b strings.Builder
	opening(&b, core)

	b.WriteString("Everything you can do here reads. Nothing you can call writes a note, a ")
	b.WriteString("card or a document, and nothing you call moves the person's window.\n\n")

	sayWhereFrom(&b)

	b.WriteString("Asking a book is not like asking a note:\n")
	booksAreNotNotes(&b)
	return b.String()
}

// getReviewingInstructions is what the window a person runs their cards in tells
// its agent.
func getReviewingInstructions(core Core) string {
	var b strings.Builder
	opening(&b, core)

	b.WriteString("The person is running their cards. Everything else you can do here ")
	b.WriteString("reads: you write the cards of a deck, and nothing you can call writes a ")
	b.WriteString("note, a link or a document, makes a deck or a stencil, or moves the ")
	b.WriteString("person's window.\n\n")

	if core.Reviewing != nil {
		b.WriteString("Which card they are on is `card_showing`: the deck, the card's mark ")
		b.WriteString("and the face it is shown through. A question that says \"this card\" ")
		b.WriteString("means the one it names, and nothing else here says which that is.\n\n")
	}

	b.WriteString("A card is written into a deck that is already there, cut by a stencil ")
	b.WriteString("that already exists. Read the deck with `card_read` before changing it, ")
	b.WriteString("and hand its fingerprint back so a write over an edit you did not see is ")
	b.WriteString("refused.\n\n")

	sayWhereFrom(&b)

	b.WriteString("Asking a book is not like asking a note:\n")
	booksAreNotNotes(&b)
	return b.String()
}

// sayWhereFrom is what an agent is told about naming where an answer came
// from, so the person can go to it again.
func sayWhereFrom(b *strings.Builder) {
	b.WriteString("Say where each part of an answer came from, in words the person can find ")
	b.WriteString("it by: the note's path and the heading it stands under, the book's name ")
	b.WriteString("and its chapter or page. Write it where you say the thing, not in a list ")
	b.WriteString("at the end. There is nothing here to open a `numen:` link with, so a link ")
	b.WriteString("is not a place a person can go.\n\n")
}

// booksAreNotNotes is what an agent is told about asking a book: search with
// the person's words first, and read on before saying a book does not say
// something. The tools that put a person in front of a passage say the rest.
func booksAreNotNotes(b *strings.Builder) {
	b.WriteString("- Search with the person's own words before searching with your own. A ")
	b.WriteString("book's sections are searched by name, and a section named what was asked ")
	b.WriteString("for is what the search answers with.\n")
	b.WriteString("- A passage is a window cut to a size and it ends where it was cut, which ")
	b.WriteString("is mid-sentence as often as not. Read on with `source_read` before saying ")
	b.WriteString("a book does not say something.\n")
}
