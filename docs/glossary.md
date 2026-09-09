# Glossary

The words this product is written in: the domain, the storage, the wire between the core and a client, and the strings a person reads. Only a word that crosses a boundary or has been contested is here — a word used in one place, meaning the obvious thing, needs no entry. A new term is added here in the change that introduces it.

The page is grouped by the context a word is spoken in, and a word spoken in two contexts has an entry in each.

A concept takes the name its field already gives it. Where a word exists for a thing — transcription, OCR, chunking, highlight, review — that word is the name, and nothing is coined beside it.

A word means one thing inside its context, and the same word in two contexts is not a collision. Nothing is renamed to keep clear of a name another context already uses. A word is listed as never called something only where the two would genuinely be taken for the same thing.

## The vault and what is written in it

| Term | What it is | Never called |
| --- | --- | --- |
| vault | A folder the person added, carrying its own identity. | |
| note | A markdown file in a vault. Its shape is [Note format](note-format.md). | |
| file | A path in a vault and the bytes at it, whichever kind the vault holds it as. A note and an asset are both files. | |
| asset | A file in a vault that is not a note. It is also the scheme a link to one is written under, and the word the window addresses one by over HTTP. | |
| entry | One file or folder, as a listing of a folder reports it. | |
| title | The name a note is shown by, and one of the keys the application owns — [Note format](note-format.md). A title names a note; a label names a relationship. | label |
| naming | Which of the two carries a note's name: `frontmatter` or `filename`. It is what a rename brings into line, and it is not the setting that says how far a rename reaches. On the wire it is `NamedBy` and the field is `by`; the word `naming` on its own is the settings section `sync_title_and_filename` stands in. | |
| sync_title_and_filename | Whether renaming either a note's title or the name of its file brings the other into line — [Settings](settings.md). On. | |
| link | One relationship, as written in a file — [Links](links.md). A link is what a person wrote; an edge is what the picture draws. | edge |
| role | What kind of relationship a link is, from a closed list of five: `parent`, `child`, `jump`, `ref`, `attachment` — [Links](links.md). A role is written in a note; a seat is worked out from where a node stands. | seat |
| type | What a link is for, as a feature reads it. Open vocabulary, and a value is only introduced with the code that reads it — [Links](links.md). | |
| type (of a note) | Which of four a note is: `note`, `deck`, `stencil` or `preset`. Closed list, and absent means `note` — [Cards](cards.md). The word is spent twice: a link's `type` is what that link is for, and a note's `type` is what the file is. | |
| label | The few words a person writes for what a relationship is called — [Links](links.md). No field carries the target's name, because a name kept beside an address is a cache in the file the address is written in. | title |
| note (on a link) | Why the link exists, in the person's words. The word is spent twice on purpose: a note is a file, and a link's `note` is why the link is there. It is the file's key; in the core the field is `Why`. | |
| address | Scheme and value; the only thing that says where a link goes. Three schemes: `name`, `note` and `asset` — [Links](links.md). | |
| identifier | The ULID a note or a vault carries in the world — [Note format](note-format.md). The number a row has inside the index is not one and does not leave the storage. | |
| backlink | A link that resolves here, whichever end wrote it — [Links](links.md). | |
| trash | Where a removed note is kept: `.trash/` inside the vault. The machine's own trash, which a whole vault folder is moved to, is the other thing the word is spent on. | |
| stale | Why a write was refused when the file on disk is no longer the one the caller read. It is one refusal among the others and not a field of its own; what a tab in that state is called to a person is `overtaken`. In the core it is `port.ErrStale`. | changed |
| refusal | Why one call was told no, from a closed list the protocol carries: missing, too large, occupied, unnameable, stale, not a stencil, and the rest. It is written once, in the core, and every window and every tool names the reason the same way. A refusal answers one call; a problem is something wrong with the vault, and a check is what notices one. A refusal is an answer: a call that could not be answered at all carries a status code instead. | problem |
| watch | Following a vault for changes the application did not make. | |
| hold | How long events are kept before they are acted on. The word is also the vault's write lock — one write to a vault happens at a time, and `Hold` is what takes it. | |
| reload | What a client is told when the vault is to be read again whole: more changed at once than could be followed, or a listener that fell behind. | |
| check | One thing that can be wrong with a vault, and what notices it. A person asks for a check by name and is answered with problems. Four: parse, frontmatter, ambiguous, dangling. | |
| problem | Something that could not be acted on and was not guessed at. | |

## The index and search

| Term | What it is | Never called |
| --- | --- | --- |
| artifact | Data that cannot be reproduced locally, deterministically and for free. Lives in the vault. Two words name one: what it is — `ocr`, `transcript`, `article`, `copy` — is the folder it stands in, and what made it is the producer, which stands in the file's name where a kind has more than one. A corrected text is a kind of its own, `ocr.corrected` and `transcript.corrected`. | |
| cache | Data that can. Lives outside the vault. | |
| bought | Data a model made. Lives outside the vault and is kept, addressed by the text it was made from and the recipe it was made under. | |
| owing | Work the index knows is wanted and has not done: a source that owes its text, a chunk that owes its vector. It is what a scan counts down and what the window draws progress from. Bought and kept are the other side of it. | |
| index | The cache. | |
| source | A thing the index holds text for. A note, a book, a recording and a url are the four kinds. | |
| producer | What made a source's text, where the source's own bytes are not it: `asr` for a recording transcribed, the recogniser's name for a scan read. It is the one word everywhere — the `producer` column in the storage, and `Producer` on `domain.Source`, `domain.Passage` and `port.SourceText` in the core — it stands in the name of a file whose kind has more than one producer, and it is empty for the ordinary source that is its own text. What a recipe names is the reader, which is the format's own extraction and a different thing under a word that would be taken for this one. | reader, extractor, text_from |
| reading | What a model read off the pages of a scanned document, with the place on the page each word stands at — [Recognising a document](recognising.md). It is the artifact a chunk of that document is a place in, named by the producer and the hash of the bytes read, and it is what a proofreader puts right. The places on pages are what separate it from a transcript, which carries times instead, and from an article, which carries neither. A recognition is the run that makes one. | recognition |
| recognition | One run of a model over a scanned document's pages, asked for by a person and never begun by the application — [Recognising a document](recognising.md). It is also the settings section `indexing.recognition`. What a recognition produces is a reading. | |
| text layer | The text a document carries of its own, which a library takes out deterministically and nothing stores — [A book's text is a cache or an artifact](adr/0015-a-books-text-is-a-cache-or-an-artifact.md). It is the cache to a reading's artifact, it is used until a person asks for the pages to be read instead, and in the core it is `port.TextLayer`. | reading |
| chunk | One cut of a source's text, as a row. Both sizes are chunks; the large one is the chunk with no parent. | |
| chunking | How a source's text is cut into chunks: the sizes, taken from the settings and from one place, so a vault cut in a terminal and one cut in a window are cut alike. | cutting |
| part | A named division of a source: the heading that names it, and where in the source's text the division begins. A note's headings and a book's outline are both parts; where a call is working is a place. A node hangs the parts of the note it stands for under its box, and choosing one opens that note where the part begins. | place |
| page | One page of a document: where its text begins in the reading, how large it was rendered, and what was found on it. It carries no name of its own — what a page is called is where it stands in the file, and a second name for one page is a second thing to be wrong about. | sheet |
| location | Where a chunk sits, in the terms its own format uses. Nullable, and never a key. | |
| spread | What a reflowing book shows at once: one column where the pane is narrow, two where it is wide, turned as one. A page of such a book belongs to the window and changes with it; a spread is what stands in front of a person now. | |
| hash | Over a source's bytes, which source it is; over a cut's text, which chunk it is. Two columns, and nothing joins one to the other. A passage carries both, named `SourceHash` and `ChunkHash`. | |
| fingerprint | Path, size and modification time — what says a note need not be read again. The vectors are keyed by a column of the same name holding a chunk hash, which is the one place the word is spent twice inside this context. | |
| stretch | A run of a source's text by where it stands: `Start` and `Length`, in bytes over the text the source is read as. It rides on the wire under its own name; a client counting a note's own text counts a span instead. | |
| passage | What a search returns: the text around a hit, and where it came from. A read model — a chunk is not rebuilt from one. | |
| highlight | Where a stretch of a document's text falls on the pages it was read from: the pages, and the rectangles covering it on each, in fractions of the page so a page drawn at any size lines up. A model reading a scan and a document's own text layer both produce them, and nothing above asks which. | |
| scan | One walk of a whole vault. The word is spent twice inside this context: a scan is also a document that is a photograph of paper, which is what a recognition reads and what a reading comes out of. | |
| refresh | Bringing named notes up to date. | |
| group | What a scan writes in: one transaction's worth. | |
| searchable | A vault whose notes are read, whose books are read, and whose chunks have their vectors. The three are one pass in one order. | |
| search mode | How a search is asked: `words`, `meaning`, `names`, or all three fused into one ranking, which the retrieval literature calls `hybrid`. In the core the four are `Lexical`, `Dense`, `ByName` and `Hybrid`. On the wire the enum is `SearchMode`, because the theme's `Mode` already holds that name in the package. | |
| vector | What a model made of one chunk's text. Kept by its text and its recipe. | |
| kept | Held past the run that made it, and claimed again by what it was made from. | |
| recipe | Everything that decides what a thing made from text is: for a cut, the reader and the sizes; for a vector, where it was made, which model, how wide, where the text was cut off and how it is kept. | |
| provider | Where a vector is made: on this machine, or by a service. | |
| presence | What a model's files are on this machine: `present` where a fetch put them, `not fetched` for a model this machine runs whose files are not here, and `nothing to fetch` for a model reached over the network. It says where files stand and nothing else, and a model is never preferred for it. | |
| arriving | A model that is not on this machine yet. What it is is known from the settings, so the index is fitted and vectors are claimed under its recipe while the weights come down. | |
| landed | The model turning up, or the reason it never will. The first of the two counts, and one turning up after the wait is over is let go of. | |
| disown | The model turning out not to be the one whose vectors are kept: it is let go of, and nothing is asked of it again. | |
| fill | Giving the index the vectors it owes. A pass that fills waits for the model; a question does not. | |

## Cards and review

| Term | What it is | Never called |
| --- | --- | --- |
| stencil | A note declaring the fields a card has and the faces it is shown by — [Cards](cards.md). Anki's words are not this product's: a card here is a card, and the note that cuts it is a stencil. It is a `Stencil` in the core, on the wire and in the window. | note type, model |
| field | One named slot of a stencil, and the heading a card writes its value under. | |
| face | One way a stencil shows a card: a front and a back, written with `{{Field}}` where a value goes. A stencil has as many as the person writes, and a face with only one of the two lays out nothing. The type holding a face's layout is `FaceTemplate`; front and back are its sides. | side |
| card face | One card shown through one face of its stencil. It is what a schedule belongs to and what a day's work is counted in, keyed by the card's mark and the face's name. The mark travels with the card between decks and between vaults; the face's name does not travel at all, and renaming a face starts its schedule again. On the wire the two halves ride as separate fields and are one identity all the same. | |
| card | One filled-in set of a stencil's fields: a second-level heading in a deck, and the values under it. Its heading is the first line of its first field, read back, and it carries a mark of its own. | note |
| mark (of a card) | The ten characters after `^` at the end of a card's heading, which is what that card is wherever it goes — [Cards](cards.md). In the core it is a `CardID`, minted in one place. A tab's mark, a page's mark and a mark on the preset's curve are other things. | |
| section (of a deck) | A first-level heading in a deck, and the cards standing under it until the next one. It is a name and nothing else: no fields, no stencil, no schedule, no mark. | |
| deck | A note whose body is cards, in sections where a person made them — [Cards](cards.md). | |
| preset | A note saying how the decks pointing at it are scheduled — [Cards](cards.md). A deck points at one with a link carrying `type: preset`, and a deck pointing at none is scheduled by the defaults. The word is spent twice: a theme that ships inside the application is named `preset:numen`. | |
| session | One run of answering cards: what is asked, in the order it is asked, and what was answered — [Flashcards](flashcards.md). A session is on one vault and over one deck, one preset, or the whole of it. Anki, SuperMemo and Mochi all call it this. In the core it is `flashcards.Session`, on the wire `StartSession`, and in the window `Session.vue`. | sitting |
| goal | Which of three a preset's one control steers: minutes a day, a retention target, or a day the material is to be in the head by. | |
| budget | What one day of a preset holds: how many new cards, how many reviews, and how long the day runs. Anki calls the three settings *daily limits*, and this is the thing they add up to, which Anki has no word for: a `BudgetName` is which of them closed a day, and it is the key the preset writes. | daily limit |
| budget unit | What a day's budget is counted in: `cards`, where a card face is charged the first time it is answered in a review day and comes round again in that day for nothing, or `shows`, where every showing is charged. It is `review.BudgetUnit` in the core and `BudgetUnit` in the window. | |
| rating | Which of the four a person answered a card face with, as the wire and the review log carry it: `again`, `hard`, `good` or `easy`, and an unspecified nothing beside them. It is `review.Rating` in the core and `rating` in the log on disk. | grade |
| grade | The four a person may actually answer with, which is `rating` less the nothing. It is the flashcards window's word, for the four buttons and the interval each of them offers, and the split is FSRS's own: `ts-fsrs` names its enum `Rating` and the subset of it a person can press `Grade`. | rating |
| owed | A card face the day holding now has reached the scheduled day of. A card owed today is owed for the whole of it, whatever hour it falls at — [Flashcards](flashcards.md). In the core it is what the `CardsDue` family counts: `CountCardsDue` over a vault, `DeckCardsDue` over a deck, `PresetCardsDue` over a preset. The index owes work of its own, which is a different thing under the same word. | due, pending |
| curve | What a preset's one control comes to over the whole range of its goal: the value of the goal at each place, what the settings come to there, and where the control stands now — [Cards](cards.md). It is a `Curve` in the core and `ComputeCurve` on the wire. The word is for the code: on screen it is **the picture**, which is what the preset tab calls it. | graph, chart |
| point | One place of a curve and what the preset comes to there: what a day of review costs, what comes back, what stands owed, and how much of the material is learned by then. It is `review.Point` in the core and `numen.v1.Point` on the wire. A place on the screen is a position. | position |
| preset counts | What a preset schedules, as the figures over the picture count it: how many decks it schedules, how many card faces stand in them, how many of those have had their day and were not answered on it, and how many nobody has answered at all. No setting moves one of them. It is `PresetCounts` in the window. | |
| make kind | Which of the three a file is made as: `deck`, `stencil` or `preset`. It is what a window asks the vault to make from nothing, and it is `MakeKind` in the window. | |
| learned | What a preset counts as a card face the person has learned: under `interval` one sent away for the preset's interval or longer, under `retention` one whose chance of being recalled today is at or above the preset's target — [Cards](cards.md). Which of the two is the person's to choose, and a goal of a date aims at it on the day it names. It says nothing about how the scheduler is treating the card. | spaced |
| load (of a day of the week) | How much of a day's load one day of the week carries under a preset, in per cent. A day the preset does not name carries all of it, and a day at nothing schedules nothing. The same word counts showings on a projected day, which is an answer and not a setting. | |
| even load | A preset moving a card off the day it fell on, onto a day of the tolerance around it carrying less. | |
| spaced | A card face the scheduler sends days away. One it is still putting into memory comes round in minutes. Retention is measured over the answers given to spaced card faces and no others. It is the scheduler's own reckoning, and no setting reaches it. The word is for the code: on screen these are **cards you are reviewing**, which is how the review history says what a day's share of recall is a share of. | learned |

## Recordings and transcription

| Term | What it is | Never called |
| --- | --- | --- |
| recording | A file in a vault whose text is what somebody said in it, and one of the three kinds of source — [Transcribing](transcribing.md). It is never moved, copied or renamed: it plays where it lies. | |
| url | The file: a web address and nothing else, under `.url` — [Importing an address](importing.md), [A url is a source of its own](adr/0038-a-url-is-a-source-of-its-own.md). It is a source beside a book and a recording, its text is what was downloaded from the address it holds, and it holds no body to write in. It is what a person renames, moves and deletes. | link, link note |
| address | What stands in that file: an `http` or `https` URL in the one form every spelling of it reaches. Everything downloaded for the file is named by it, and its fingerprint is the `hash` of the source. A link's address is the other thing the word is spent on. | |
| asr | A producer: a model here, listening to a recording — [Transcribing](transcribing.md). What it hears is kept as a transcript, and `asr` stands in the file's name to tell it from `captions`. | |
| captions | A producer: a site publishing words with a video — [Importing an address](importing.md). What it published is kept as a transcript, read and put right by what reads and puts right any other, and `captions` stands in the file's name to tell it from `asr`. | subtitles |
| article | The prose a page is written around, and the kind it is kept under — [Importing an address](importing.md). It carries no times and no places on pages, which is what separates it from a transcript and from a reading. | |
| copy | The bytes of a video fetched onto this disk, asked for by hand — [Importing an address](importing.md). It is named by the address like everything else fetched for a url, and the tab plays it instead of framing the site. | |
| transcript | The kind: text with the times each stretch of it was said at, written as WebVTT — [Transcribing](transcribing.md), [A transcript is WebVTT](adr/0031-a-transcript-is-webvtt.md). It is the artifact a chunk of a recording or of a url is a place in, and it is what a proofreader puts right. Two producers write one — `asr` and `captions` — and both write into the folder the kind is named by. | |
| cue | One stretch of speech in a transcript: what was said, when it was said, and where it stands in the text the transcript reads as. It is WebVTT's own word for the thing, so it is the word here, in `transcript.Cue` and on the wire. | caption, segment, subtitle |
| transcription | One run of a model over a recording. It is also the settings section `indexing.transcription` and the adapter that reads it, in a row with `recognition` and `proofreading`. What a transcription produces is a transcript. Unlike a recognition it is not asked for: a recording nobody asked about is listened to where `indexing.transcribe_recordings` says so. | |
| proofread | One run of a second model over the text a first one produced, putting it right and writing the corrections beside the artifact — [Proofreading](proofreading.md). It is the word wherever that run is named: the `proofread` key each consumer names its profile under, the command, and **Proofread transcript** on screen. Writing a transcript back as a person edited it in the window is not one. | |
| proofreading | How an installation proofreads: the profiles a proofreader is reached at, and how far a correction may move a line's letters. It is the settings section `indexing.proofreading` — [Proofreading](proofreading.md). A proofread is one run; proofreading is what every run is asked through. | |
| profile | One place a reading is put right at, by the name a consumer asks for it under: a service speaking the chat-completions shape, or the `claude` command line. An installation naming none proofreads nothing. A preset is the cards' word and is not this. | |
| delete (a transcript) | Taking away everything transcribing a recording produced: what a model wrote down, what a person put right, the record of what transcribed it, the answer and the chunks cut from any of them — [Transcribing](transcribing.md). It is **Delete transcript** on screen. | drop |

## The agent

| Term | What it is | Never called |
| --- | --- | --- |
| agent | A program acting on a vault on a person's behalf, through tools. | |
| tool | One operation an agent can call, by the name it is served under. A tool is not a use case: several tools reach one, and one tool reaches several. | a use case |
| MCP | The protocol this vault's tools are served over, so another program can reach them — [The agent](agents.md), [Settings](settings.md). The setting is `agent.serve_tools`, and the address is `-mcp-addr`. | |
| client | A consumer of the schema that draws a vault. An agent acts on a vault; a client draws one. | an agent |
| conversation | One thread of talk with an agent, named by the client and carried in every question of it. Where the English word is wanted the phrase is *thread of talk*, and the field is still `conversation`. Claude Code calls the conversation it is keeping a `session_id`; that word is the other program's and never leaves the adapter that started it. | a session |
| finish | Saying a conversation is over: nothing is asked under its name again, and what the agent kept of it is let go of. One answer given up on is a stop, and the conversation stays open. | |
| step | One thing an agent said, did, or stopped for, as the panel is told about it. | |
| call | What an agent named one use of a tool, so every step reporting it is known to be one. | |
| kind | What a call does to the vault: `read`, `edit`, `remove`, `move`, `search`. A call that says no more than that it is one is a *tool call*. A note is created and a link is added; both are removed. | deleted |
| place | Where a call is working: a source, by the path the vault files it under, and the stretch of that source's text the call names. A length of zero names the source and nothing inside it. | part |
| shown vault | The vault a call through the tools is answered about: the vault itself, and the folder it stands in. A build naming none answers about no vault at all. In `adapter/mcp` it is `ShownVault`, and the window spends the word too. | |
| stood | A run named by the text standing in it, quoted. What an agent names a stretch by when it has read prose and not measured it. The field a note edit carries that text in is `match`, which is what the tool calls the text as the note has it. | |

## The installation

| Term | What it is | Never called |
| --- | --- | --- |
| registry | The list of vaults the installation knows. It is application state and is not in the settings file. | |
| settings | What a person configures about an installation, and the one file it is in — [Settings](settings.md). | |
| setting | One field of `numen.json` and what stands there: a path through the file, and a value written as JSON — [Settings](settings.md). | |
| model | What a setting that runs against a model names: the model the vault is indexed by, the one a scanned page is read by, the one a transcript is put right at, and the one the agent answers with — [Settings](settings.md). | |

## The window

| Term | What it is | Never called |
| --- | --- | --- |
| window | The application's window on screen, and nothing else. | |
| plex | The focused neighbourhood the product is named for. | |
| neighbourhood | One note and everything joined to it, seen from that note. | |
| neighbour | One of the notes a neighbourhood holds: a note joined to the one it is seen from, either way round. | |
| focus | A neighbourhood's: the note it is seen from. A workspace's: the pane a tab opens into. | |
| node | What is drawn in place of a note. A row of a tree is not one. | row |
| ticket | What the picture calls a note. The application mints one the first time a note is drawn, and the note holds it while its file moves. Every gesture the plex reports names a node by its ticket, and the application translates it back to a path. | |
| edge | A line drawn between two nodes. Two links can be one edge. | link |
| seat | Where a node sits relative to the focus: `parent`, `child`, `jump`, `sibling`. It is not a value that can be written to a note. | role |
| hang_parts_under_a_node | Whether a node hangs the parts of the note it stands for under its box — [Settings](settings.md). On. | |
| parts_under_a_node | How many of those parts stand under a node at once, the rest being wound to — [Settings](settings.md). 6. | |
| viewport | The area the plex is drawn into. | window |
| position | A place on the screen, `{ x, y }`, in whatever coordinates the caller measures in. The plex, the tree, the menu and the workspace all pass one around, and it is what `@vueuse/core` calls the pair. A place on a preset's curve is a point. | point |
| span | A run of text as a client counts it: `from` and `to`, in UTF-16 code units. It is how a note's own text is addressed on the wire; a run of a source's text rides there as a `Stretch`, in bytes, and in the core every run is a stretch. | |
| standing on nothing | A window showing no vault: an installation that holds none, or one whose vault was taken down and nothing came up in its place. Every question that would reach into a vault is refused there, and the welcome screen offers the list and the way to add one. It names nothing in the code: it is the phrase the core answers such a window in, and the state itself is the vault a window shows being none. | |
| shown vault | What the welcome screen is drawn over: the vault the window is showing, and whether it has been read and can be asked to do anything. It is `ShownVault` in the window, and the tools spend the word on the vault a call is answered about. | |
| panel | The column beside the plex where a person asks an agent something. | |
| turn | One thing shown in the panel's conversation: what was asked, what was answered, what is being done. | |
| voice | Whose turn it is, and so how it is drawn: `asked`, `doing`, `answered`. | |
| task | One piece of work the application is doing behind the window, by a name that stays the same so the same work reported again replaces itself: what is being done, what it is on, and how far it has got where there is a total to count against. Every kind of work is one of these, so the next kind is an entry in a list and not another field, another poll and another branch in what draws it. A task is what is happening; nothing here is a record. | |
| workspace | Everything the window holds open, and how it is split. | |
| branch | A split of the workspace, drawn as two parts side by side. | |
| pane | One part of a branch, holding tabs and showing one of them. | |
| tab | One thing a pane holds open, shown by its title. | |
| tab kind | Which of the kinds of tab a window draws, declared to the window once: how a tab of it opens, what it is called, what is drawn in it and what letting go of it comes to. A window is free to open a kind nothing in the application has heard of. It is `TabKind` in the window; the agent's `kind` — what a call does to the vault — is the other thing the word is spent on. | plugin, view |
| welcome | What the window draws while it holds no tab: the mark, the ways into the vault, and the vaults this installation holds. | |
| tree | A hierarchy of rows drawn as an indented list, some of them holding others. The vault's folders and files are shown in one. | |
| row | One line of a tree: an entry, at the depth it sits. | node |
| selection | The rows of a tree chosen together. A gesture made on one of them is made on all of them. | |
| drag | The gesture while it runs: what has been lifted, and where the pointer is. The tree says what it has lifted; the plex draws a line to it and never looks at what it is. In code it is `drag` and `dragged`, and the element on its way is `data-dragged`. The English *carry* is not this: in this repository's prose it means to bear — a note carries an identifier, a line carries a title — and it names nothing in the code. | carry |
| drop | Where a drag lands and what that comes to: the seat, the place in an order, the link that gets written. In code it is `drop` and `dropped`. | |
| anchor | The row a selection is reached from, which is where a plain or joining press last landed. | |
| unsaved | A tab whose text is not the text in its file. | |
| stuck | A tab whose file can be neither read nor written: not a note, not text, over the ceiling, or frontmatter that will not parse. | |
| gone | What a name points to is not on disk: a tab whose name has no file behind it, so its save stopped, and a vault with nothing at its path. | |
| overtaken | A tab whose file no longer holds the prose the tab read, so its save stopped. | |
| keep | Writing an overtaken tab's prose over its file, when the person says so. | |
| take | Replacing an overtaken tab's prose with its file's, when the person says so. | |
| mark (of a tab) | The one word a tab carries beside its title for the state it is in, and what a screen reader reads out: `stuck`, `gone`, `overtaken` or `unsaved`, the first of those that holds. A card's mark is the other thing the word is spent on, and the form feed between two pages of a reading's artifact is a page break. | |

## The interface

| Term | What it is | Never called |
| --- | --- | --- |
| palette | The list of everything that can be done, opened over the window and narrowed by typing. It is drawn in bands, each item offering the actions whoever put it there named. | |
| menu | A list of things that can be done, opened on what they are done to. | |
| invocation | One command as it is carried out: the note and the vault it is over, the tab holding that note, the files it is over, and what was typed for it. It is `CommandInvocation` in the window. | |
| band | A stretch of one list of things to choose. The palette gives each a title; a menu draws a rule where one band gives way to the next. | |
| shelf | The heading a run of rows stands under in a list: the two a theme comes off, and the ones the agent's models stand on. | |
| current | The row a list opens on, which is the one in force: the theme shelf, light and dark, the two size ladders, the vaults. | |
| theme | A CSS file redeclaring tokens under `:root`, applied whole. Exactly one is applied — [Themes](themes.md). | |
| preset | A theme shipped inside the application, named `preset:`. A theme in the person's own folder is named `mine:` — [Themes](themes.md). | |
| mode | Which half of a token's pair is taken: `system`, `light`, `dark` — [Settings](settings.md). The field is `mode` on the wire and in the window; in the core the type is `ColorScheme`. | |
| interface_scale | How large the interface is drawn — [Settings](settings.md). A multiplier, 1 being as designed. | |
| text_scale | How large the text a person reads is set — [Settings](settings.md). A multiplier, 1 being as designed. | |
