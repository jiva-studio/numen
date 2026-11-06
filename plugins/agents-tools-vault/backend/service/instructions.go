package service

// ArrangeInstructions is the instruction paragraph for note_list, note_titles, note_resolve,
// vault_list and vault_problems.
const ArrangeInstructions = "Use note_list to list the notes of the vault or of one folder by path and title, and page through them with limit and offset. Use note_titles to get the titles of notes whose paths you already have, and note_resolve to get the paths of the notes a wikilink name points to; more than one path means the name is ambiguous. vault_list lists every vault of the installation and marks the vault the call works on as showing. vault_problems reports the files of the vault that could not be parsed."

// citeInstructions is the instruction paragraph for naming notes and passages, and for note_search,
// library_search, library_read and file_read.
const citeInstructions = "Name a note you write about as a link with the title the person knows it by, for example [[Tidal pools]], where you write about it, not in a list at the end; a title several notes share is written as [[note://<identifier>]], which names one of them. " +
	"Write every passage of a book, a PDF or a recording that you write about as a markdown link to the address library_search returns for it under `at`, in place, not in a list at the end, for example [the harbour chapter](numen:library%2FA%20Sea%20Atlas.pdf?start=40120&length=980). " +
	"Never use file:// links for notes or books. " +
	"note_search finds notes only; library_search finds passages in everything the vault holds: notes, books, PDFs, scans and recordings. library_read reads the text of one source a page or a range at a time, and file_read reads text files and refuses a binary. " +
	"Search with the person's own words before your own. " +
	"A passage ends where it was cut, often mid-sentence: read on with library_read before you say that a source does not say something. " +
	"For a question about a book, pass kinds [\"file\"] to library_search, because a vault holds far more notes than books."

// instructions is the paragraph the vault tools list with them.
const instructions = citeInstructions + " " + ArrangeInstructions
