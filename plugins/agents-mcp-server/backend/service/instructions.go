package service

// instructions is what the server tells an agent once, before it calls anything.
//
// A tool is named only here when the plugin that offers it is always present, so that no sentence
// sends the agent to a tool that is not listed.
const instructions = `These tools work on one Numén vault: markdown notes in an ordinary folder, joined by links, with books, PDFs and recordings filed beside them.

Naming what you speak about:
- A note is addressed by its path relative to the vault folder, and referred to by its title. Name a note you speak about as a link, ` + "`[[Harmonic oscillator]]`" + `, the title the person knows it by. Write it where you speak about the note, not in a list at the end. A title several notes share is written as ` + "`[[note://<identifier>]]`" + `, which names one of them.
- Every passage of a book, a PDF or a recording that you speak about is written as a markdown link, and a passage written about without a link is one the person cannot go to. ` + "`library_search`" + ` gives the address of each passage under ` + "`at`" + `: put it in a markdown link where you speak about the passage, not in a list at the end.
  ` + "`[the Remuna episode](numen:library%2FA%20Book.pdf?start=62690&length=1246)`" + `
- Never use file:// links for vault notes or books. A link to vault content is ` + "`numen:<path>?start=...&length=...`" + ` for a passage of a document and ` + "`[[Note]]`" + ` for a note.

Searching and reading:
- ` + "`note_search`" + ` finds notes only. ` + "`library_search`" + ` finds passages in everything the vault holds: notes, books, PDFs, scans and recordings. ` + "`library_read`" + ` reads the text of one source from its text version, a page or a range at a time. ` + "`file_read`" + ` reads text files and refuses a binary.
- Search with the person's own words before searching with your own.
- A passage is a window cut to a size and it ends where it was cut, which is mid-sentence as often as not. Read on with ` + "`library_read`" + ` before saying that a source does not say something.
- When the person asks about a book, pass ` + "`kinds: [\"file\"]`" + ` to ` + "`library_search`" + `: a vault holds far more notes than books, and a search told to look everywhere answers with notes.
`
