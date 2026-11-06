package service

// The instruction sentences of the vault tools. A sentence about several tools is attached to each
// of them, and the host lists it once.
const (
	noteLinkSentence    = "Name a note you write about as a link with its title, for example [[Tidal pools]], where you write about it, not in a list at the end; a title several notes share is written as [[note://<identifier>]], which names one of them."
	passageLinkSentence = "Write every passage of a book, a PDF or a recording that you write about as a markdown link to the address library_search returns for it under `uri`, in place, not in a list at the end, for example [the harbour chapter](numen:library%2FA%20Sea%20Atlas.pdf?start=40120&length=980)."
	noFileLinkSentence  = "Never use file:// links for notes or books."
	searchScopeSentence = "note_search finds notes only; library_search finds passages in everything the vault holds: notes, books, PDFs, scans and recordings."
	userWordsSentence   = "Search with the user's own words before your own."
	truncatedSentence   = "A passage is truncated, often mid-sentence: read on with library_read before you say that a source does not say something."
	bookKindSentence    = "For a question about a book, pass kinds [\"file\"] to library_search, because a vault holds far more notes than books."
	libraryReadSentence = "library_read reads the text of one source a page or a range at a time."
	fileReadSentence    = "file_read reads text files and refuses a binary."

	noteListSentence      = "Use note_list to list the notes of the vault or of one folder by path and title, and page through them with limit and offset."
	noteTitlesSentence    = "Use note_titles to get the titles and identifiers of notes whose paths you already have."
	noteResolveSentence   = "Use note_resolve to get the paths of the notes a wikilink name points to; more than one path means the name is ambiguous."
	vaultListSentence     = "vault_list lists every vault of the installation, marks the vault the call works on with isCurrent and a vault whose folder is gone with isMissing."
	vaultProblemsSentence = "vault_problems runs the parse, frontmatter and ambiguous checks over the vault, and the dangling check only when you name it."
	noteRenameSentence    = "note_rename gives a note another title and moves its file to the title's file name."
	noteMoveSentence      = "note_move moves notes to a folder."
	noteRemoveSentence    = "note_remove moves notes to the vault's trash, from where they can be put back."
	linkRewriteSentence   = "After note_rename or note_move the host rewrites the links that named the note by its old name."
	linkEditSentence      = "link_add, link_update and link_remove write the links in the frontmatter of a note and need the fingerprint note_read gave for that note; a stale fingerprint is refused, so read the note again."
	vaultAddSentence      = "vault_add adds a folder as a vault."
	vaultRenameSentence   = "vault_rename changes the name of a vault."
	vaultRemoveSentence   = "vault_remove takes a vault off the list and leaves its folder on the disk."
	vaultOpenSentence     = "vault_open switches the user's window to another vault and ends this session's access to the vault it works on, so call it only when the user asks for it."
	vaultOpenAskSentence  = "When the window has unsaved changes, vault_open switches nothing, the window asks the user about them, and the error says to call it again after the user answers."
)
