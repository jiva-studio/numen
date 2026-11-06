package service

// ArrangeInstructions is the instruction paragraph for the tools that list, title, resolve, rename,
// move and remove notes, edit the links of notes, and list, check and manage vaults.
const ArrangeInstructions = "Use note_list to list the notes of the vault or of one folder by path and title, and page through them with limit and offset. " +
	"Use note_titles to get the titles and identifiers of notes whose paths you already have, and note_resolve to get the paths of the notes a wikilink name points to; more than one path means the name is ambiguous. " +
	"vault_list lists every vault of the installation, marks the vault the call works on as showing and a vault whose folder is gone as missing. " +
	"vault_problems runs the parse, frontmatter and ambiguous checks over the vault, and the dangling check only when you name it. " +
	"note_rename gives a note another title and moves its file to the title's file name, note_move moves notes to a folder, and note_remove moves notes to the vault's trash, from where they can be put back; the host writes again the links that named a renamed or moved note by its old name. " +
	"link_add, link_update and link_remove write the links in the frontmatter of a note and need the fingerprint note_read gave for that note; a stale fingerprint is refused, so read the note again. " +
	"vault_add adds a folder as a vault, vault_rename changes the name of a vault, and vault_remove takes a vault off the list and leaves its folder on the disk. " +
	"vault_open moves the person's window to another vault and ends this session's view of the vault it works on, so call it only when the person asks for it."

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
