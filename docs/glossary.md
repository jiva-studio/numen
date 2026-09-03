# Glossary

The words this product is written in: the domain, the storage, the wire between the core and a client, and the strings a person reads. Only a word that crosses a boundary or has been contested is here — a word used in one place, meaning the obvious thing, needs no entry. A new term is added here in the change that introduces it.

The page is grouped by the context a word is spoken in, and a word spoken in two contexts has an entry in each.

A concept takes the name its field already gives it. Where a word exists for a thing — transcription, OCR, chunking, highlight, review — that word is the name, and nothing is coined beside it.

A word means one thing inside its context, and the same word in two contexts is not a collision. Nothing is renamed to keep clear of a name another context already uses.

## The vault and what is written in it

| Term | What it is | Never called |
| --- | --- | --- |
| vault | A folder the person added, carrying its own identity. | |
| note | A markdown file in a vault. Its shape is [Note format](note-format.md). | |
| file | A path in a vault and the bytes at it, whichever kind the vault holds it as. A note and an asset are both files. | |
| asset | A file in a vault that is not a note. | |
| entry | One file or folder, as a listing of a folder reports it. | item |
| title | The name a note is shown by, and one of the keys the application owns — [Note format](note-format.md). A title names a note; a label names a relationship. | label |
| naming | Which of the two carries a note's name: `frontmatter` or `filename`. It is what a rename brings into line, and it is not the setting that says how far a rename reaches. | |
| sync_title_and_filename | Whether renaming either a note's title or the name of its file brings the other into line — [Settings](settings.md). On. | |
| link | One relationship, as written in a file — [Links](links.md). | connection |
| role | What kind of relationship a link is, from a closed list of five — [Links](links.md). | seat |
| type | What a link is for, as a feature reads it. Open vocabulary — [Links](links.md). | |
| type (of a note) | Which of four a note is: `note`, `deck`, `stencil` or `preset`. Closed list, and absent means `note` — [Cards](cards.md). The word is spent twice: a link's `type` is what that link is for, and a note's `type` is what the file is. | |
| label | The few words a person writes for what a relationship is called — [Links](links.md). | title |
| note (on a link) | Why the link exists, in the person's words. The word is spent twice on purpose: a note is a file, and a link's `note` is why the link is there. | |
| address | Scheme and value; the only thing that says where a link goes — [Links](links.md). | |
| identifier | The ULID a note or a vault carries in the world — [Note format](note-format.md). The number a row has inside the index is not one and does not leave the storage. | |
| backlink | A link that resolves here, whichever end wrote it — [Links](links.md). | |
| trash | Where a removed note is kept: `.trash/` inside the vault. | |
| changed | What a write answers when the note on disk is no longer the one the caller read. | conflict |
| watch | Following a vault for changes the application did not make. | |
| hold | How long events are kept before they are acted on. | window |
| reload | What a client is told when the vault is to be read again whole: more changed at once than could be followed, or a listener that fell behind. | |
| check | One thing that can be wrong with a vault, and what notices it. A person asks for a check by name and is answered with problems. | lint |
| problem | Something that could not be acted on and was not guessed at. | |

## The index and search

| Term | What it is | Never called |
| --- | --- | --- |
| artifact | Data that cannot be reproduced locally, deterministically and for free. Lives in the vault. | |
| cache | Data that can. Lives outside the vault. | |
| bought | Data a model made. Lives outside the vault and is kept, addressed by the text it was made from and the recipe it was made under. | |
| index | The cache. | |
| source | A thing the index holds text for. A note and a book are kinds of source. | |
| chunk | One cut of a source's text, as a row. Both sizes are chunks; the large one is the chunk with no parent. | |
| cutting | How a source's text is cut into chunks: the sizes, taken from the settings and from one place, so a vault cut in a terminal and one cut in a window are cut alike. | |
| part | A named division of a source: the heading that names it, and where in the source's text the division begins. A note's headings and a book's outline are both parts; where a call is working is a place. A node hangs the parts of the note it stands for under its box, and choosing one opens that note where the part begins. | place |
| page | One page of a document, at the offset where its text begins. Called by where it stands in the file, and by nothing else. | sheet |
| sheet | One page as it was read off a scan: how big it is, and what was found on it. | page |
| location | Where a chunk sits, in the terms its own format uses. Nullable, and never a key. | |
| hash | Over a file's bytes, which file it is; over a cut's text, which chunk it is. Two columns, and nothing joins one to the other. | |
| fingerprint | Path, size and modification time — what says a note need not be read again. | |
| stretch | A run of a source's text by where it stands: `Start` and `Length`, in bytes over the text the source is read as. A client counts the same run as a span. | |
| passage | What a search returns: the text around a hit, and where it came from. | |
| scan | One walk of a whole vault. | |
| refresh | Bringing named notes up to date. | reindex, incremental |
| group | What a scan writes in: one transaction's worth. | pane |
| searchable | A vault whose notes are read, whose books are read, and whose chunks have their vectors. The three are one pass in one order. | |
| way | How a search is asked: `words`, `meaning`, `part`, or every way fused into one ranking. | half |
| vector | What a model made of one chunk's text. Kept by its text and its recipe. | |
| kept | Held past the run that made it, and claimed again by what it was made from. | stored |
| recipe | Everything that decides what a thing made from text is: for a cut, the reader and the sizes; for a vector, where it was made, which model, how wide, where the text was cut off and how it is kept. | |
| station | Where a vector is made: on this machine, or by a service. | placement |
| presence | What a model's files are on this machine: `present` where a fetch put them, `not fetched` for a model this machine runs whose files are not here, and `nothing to fetch` for a model reached over the network. It says where files stand and nothing else, and a model is never preferred for it. | installed, missing, available, downloaded |
| arriving | A model that is not on this machine yet. What it is is known from the settings, so the index is fitted and vectors are claimed under its recipe while the weights come down. | |
| landed | The model turning up, or the reason it never will. The first of the two counts, and one turning up after the wait is over is let go of. | |
| disown | The model turning out not to be the one whose vectors are kept: it is let go of, and nothing is asked of it again. | |
| fill | Giving the index the vectors it owes. A pass that fills waits for the model; a question does not. | |

## Cards and review

| Term | What it is | Never called |
| --- | --- | --- |
| stencil | A note declaring the fields a card has and the faces it is shown by — [Cards](cards.md). | note type, model, template |
| field | One named slot of a stencil, and the heading a card writes its value under. | |
| face | One way a stencil shows a card: a front and a back, written with `{{Field}}` where a value goes. A stencil has as many as the person writes. | template, side |
| card | One filled-in set of a stencil's fields: a second-level heading in a deck, and the values under it. Its heading is the first line of its first field, read back, and it carries a mark of its own. | |
| mark (of a card) | The ten characters after `^` at the end of a card's heading, which is what that card is wherever it goes — [Cards](cards.md). A page's mark and the marks an interface is drawn with are other things. | |
| section (of a deck) | A first-level heading in a deck, and the cards standing under it until the next one. It is a name and nothing else. | |
| deck | A note whose body is cards, in sections where a person made them — [Cards](cards.md). | collection |
| preset | A note saying how the decks pointing at it are scheduled — [Cards](cards.md). A deck points at one with a link carrying `type: preset`, and a deck pointing at none is scheduled by the defaults. The word is spent twice: a theme that ships inside the application is named `preset:numen`. | profile, options |
| goal | Which of three a preset's one control steers: minutes a day, a retention target, or a day the material is to be in the head by. | target |
| learned | What a preset counts as a card the person has learned: under `interval` one sent away for the preset's interval or longer, under `retention` one whose chance of being recalled today is at or above the preset's target — [Cards](cards.md). Which of the two is the person's to choose, and a goal of a date aims at it on the day it names. It says nothing about how the scheduler is treating the card. | mature, known, graduated, spaced |
| load (of a day of the week) | How much of a day's load one day of the week carries under a preset, in per cent. A day the preset does not name carries all of it, and a day at nothing schedules nothing. | light day, rest day |
| even load | A preset moving a card off the day it fell on, onto a day of the tolerance around it carrying less. | spreading, levelling |
| spaced | A card face the scheduler sends days away. One it is still putting into memory comes round in minutes. Retention is measured over the answers given to spaced card faces and no others. It is the scheduler's own reckoning, and no setting reaches it. The word is for the code: on screen these are **cards you are reviewing**, which is how the review history says what a day's share of recall is a share of. | learned, mature, in review |

## Recordings and transcription

| Term | What it is | Never called |
| --- | --- | --- |
| proofread | Asking a second model to put right the text a first one produced, and writing its corrections beside the artifact — [Proofreading](proofreading.md). It is the word for the run wherever it is named: the `proofread` facet of an asset, the command, and **Proofread transcript** on screen. Writing a transcript back as a person edited it in the window is not one. | put right, correct |
| drop (a transcript) | Taking away everything listening to a recording produced: what a model heard, what a person put right, the record of what listened, the answer and the chunks cut from any of them — [Transcribing](transcribing.md). The word is for the code: on screen it is **Delete transcript**. | forget, clear |

## The agent

| Term | What it is | Never called |
| --- | --- | --- |
| agent | A program acting on a vault on a person's behalf, through tools. | |
| tool | One operation an agent can call. | a use case |
| MCP | The protocol this vault's tools are served over, so another program can reach them — [The agent](agents.md), [Settings](settings.md). The setting is `agent.serve_tools`, and the address is `-mcp-addr`. | tools on a port |
| client | A consumer of the schema that draws a vault. | an agent |
| conversation | One thread of talk with an agent, named by the client and carried in every question of it. Where the English word is wanted the phrase is *thread of talk*, and the field is still `conversation`. | |
| session | What the agent's own program calls a conversation it is keeping, named by that program. It never leaves the adapter that started it. | |
| finish | Saying a conversation is over: nothing is asked under its name again, and what the agent kept of it is let go of. One answer given up on is a stop, and the conversation stays open. | close |
| step | One thing an agent said, did, or stopped for, as the panel is told about it. | |
| call | What an agent named one use of a tool, so every step reporting it is known to be one. | |
| kind | What a call does to the vault: `read`, `edit`, `remove`, `move`, `search`. A call that says no more than that it is one is a *tool call*. A note is created and a link is added; both are removed. | deleted |
| place | Where a call is working: a source, by the path the vault files it under, and the stretch of that source's text the call names. A length of zero names the source and nothing inside it. | placement |
| stood | A run named by the text standing in it, quoted. What an agent names a stretch by when it has read prose and not measured it. | |

## The installation

| Term | What it is | Never called |
| --- | --- | --- |
| registry | The list of vaults the installation knows. | |
| settings | What a person configures about an installation, and the one file it is in — [Settings](settings.md). | |
| setting | One field of `numen.json` and what stands there: a path through the file, and a value written as JSON — [Settings](settings.md). | option, preference |
| model | What a setting that runs against a model names: the model the vault is indexed by, the one a scanned page is read by, and the one the agent answers with — [Settings](settings.md). | engine |

## The window

| Term | What it is | Never called |
| --- | --- | --- |
| window | The application's window on screen, and nothing else. | |
| plex | The focused neighbourhood the product is named for. | |
| neighbourhood | One note and everything joined to it, seen from that note. | |
| neighbour | One of the notes a neighbourhood holds: a note joined to the one it is seen from, either way round. | |
| focus | A neighbourhood's: the note it is seen from. A workspace's: the pane a tab opens into. | the keyboard's position |
| node | What is drawn in place of a note. | |
| ticket | What the picture calls a note. The application mints one the first time a note is drawn, and the note holds it while its file moves. Every gesture the plex reports names a node by its ticket, and the application translates it back to a path. | |
| edge | A line drawn between two nodes. Two links can be one edge. | connection |
| seat | Where a node sits relative to the focus: `parent`, `child`, `jump`, `sibling`. It is not a value that can be written to a note. | role |
| hang_parts_under_a_node | Whether a node hangs the parts of the note it stands for under its box — [Settings](settings.md). On. | |
| parts_under_a_node | How many of those parts stand under a node at once, the rest being wound to — [Settings](settings.md). 6. | |
| viewport | The area the plex is drawn into. | window |
| span | A run of text as a client counts it: `from` and `to`, in UTF-16 code units. The one form a run takes on the wire; in the core the same run is a stretch. | |
| highlight | Where a stretch of a document's text falls on the pages it was read from: the pages, and the rectangles covering it on each. | |
| standing on nothing | A window showing no vault: an installation that holds none, or one whose vault was taken down and nothing came up in its place. Every question that would reach into a vault is refused there, and the welcome screen offers the list and the way to add one. | empty vault, no workspace |
| panel | The column beside the plex where a person asks an agent something. | |
| turn | One thing shown in the panel's conversation: what was asked, what was answered, what is being done. | |
| voice | Whose turn it is, and so how it is drawn. | |
| workspace | Everything the window holds open, and how it is split. | |
| branch | A split of the workspace, drawn as two parts side by side. | |
| pane | One part of a branch, holding tabs and showing one of them. | group |
| tab | One thing a pane holds open, shown by its title. | |
| welcome | What the window draws while it holds no tab: the mark, the ways into the vault, and the vaults this installation holds. | splash, start page |
| tree | A hierarchy of rows drawn as an indented list, some of them holding others. The vault's folders and files are shown in one. | |
| row | One line of a tree: an entry, at the depth it sits. | node |
| selection | The rows of a tree chosen together. A gesture made on one of them is made on all of them. | |
| carry | A gesture that lifts something in one part of the window and lets it go in another. The tree says what it has lifted; the plex says which seat letting go over it comes to, and never what is being carried. | |
| anchor | The row a selection is reached from, which is where a plain or joining press last landed. | |
| unsaved | A tab whose text is not the text in its file. | |
| stuck | A tab whose file can be neither read nor written: not a note, not text, over the ceiling, or frontmatter that will not parse. | |
| gone | What a name points to is not on disk: a tab whose name has no file behind it, so its save stopped, and a vault with nothing at its path. | |
| overtaken | A tab whose file no longer holds the prose the tab read, so its save stopped. | |
| keep | Writing an overtaken tab's prose over its file, when the person says so. | |
| take | Replacing an overtaken tab's prose with its file's, when the person says so. | |
| mark | The one word a tab carries beside its title for the state it is in, and what a screen reader reads out: `stuck`, `gone`, `overtaken` or `unsaved`, the first of those that holds. The form feed between two pages of a reading's artifact is a page break. | |

## The interface

| Term | What it is | Never called |
| --- | --- | --- |
| menu | A list of things that can be done, opened on what they are done to. | |
| band | A stretch of one list of things to choose. The palette gives each a title; a menu draws a rule where one band gives way to the next. | |
| shelf | The heading a run of rows stands under in a list: the two a theme comes off, and the ones the agent's models stand on. | |
| current | The row a list opens on, which is the one in force: the theme shelf, light and dark, the two size ladders, the vaults. | worn now, the size now |
| theme | A CSS file redeclaring tokens under `:root`, applied whole. Exactly one is applied — [Themes](themes.md). | |
| preset | A theme shipped inside the application, named `preset:`. A theme in the person's own folder is named `mine:` — [Themes](themes.md). | |
| mode | Which half of a token's pair is taken: `system`, `light`, `dark` — [Settings](settings.md). | |
| interface_scale | How large the interface is drawn — [Settings](settings.md). A multiplier, 1 being as designed. | zoom |
| text_scale | How large the text a person reads is set — [Settings](settings.md). A multiplier, 1 being as designed. | zoom |
