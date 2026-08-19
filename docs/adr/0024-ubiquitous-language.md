# ADR-0024: Ubiquitous language

- **Status:** Accepted
- **Date:** 2026-08-16
- **Extended:** 2026-08-17 — `source`, `chunk`, `location`, `passage` (ADR-0006)
- **Extended:** 2026-08-17 — `reload`, `hash`, `mark`, `stuck`; `focus` settled a
  third time (ADR-0032, ADR-0034)
- **Extended:** 2026-08-18 — `overtaken`, `keep`, `take`; `mark` takes a third
  value and an order (ADR-0032)
- **Extended:** 2026-08-18 — `conversation`, and `session` settled apart from it
  (ADR-0031)
- **Extended:** 2026-08-18 — `finish`, settled apart from `stop` (ADR-0031)
- **Applies to:** the product as a whole
- **Related:** ADR-0003, ADR-0006, ADR-0011, ADR-0014, ADR-0020, ADR-0031,
  ADR-0032, ADR-0034

## Context

The same system is written down five times over: in the prose of these
decisions, in the domain, in the storage, on the wire between the core and a
window, and in what the person using it reads. Each was written at a different
time, and nothing has required them to agree.

They have not. One thing goes by several names, and one name means several
things. Both cost the same thing: a reader crossing a boundary translates in
their head, and a translation nobody wrote down is a translation nobody can
check.

The practice has a name — Evans's *ubiquitous language*: one language spoken by
the code, the documents and the people talking about them, so that nothing has
to be translated in order to be understood.

## Decision

### One word for one thing, and one thing for one word

Two halves, and the second is the one that bites.

**A concept has one name.** If the domain, the storage, the wire and the
interface are talking about the same thing, they use the same word for it. A
field renamed on the way across a boundary is a defect unless a rule requires
the rename, and the rule is written here.

**A word means one thing.** A word already spent is spent. Reusing it for a
second concept is worse than inventing a synonym, because a synonym announces
itself and a homonym does not: the reader carries the wrong meaning across and
nothing tells them.

### The language is one language

The words below hold in every body of text — the decisions, the domain, the
storage, the wire, the interface, and the strings a person reads. A word that
cannot survive a boundary is the wrong word, not evidence that the boundary
needs a dialect.

There is one exception, and it is narrow: the interface may not know the vault
(ADR-0020). Where a domain word would teach it something it must not know, it
takes a word of its own — and that word is in this vocabulary too, meaning that
one thing and nothing else.

### The vocabulary

Only words that cross a boundary or have been contested are listed. A word used
in one place, meaning the obvious thing, needs no entry.

**What the person writes**

| Word | Means |
|---|---|
| vault | A folder the person added, carrying its own identity |
| note | A markdown file in a vault |
| asset | A file in a vault that is not a note |
| title | The name a note is shown by |
| link | One relationship, as written in a file |
| role | What kind of relationship a link is: `parent`, `child`, `jump`, `ref`, `attachment`. Written by the person, closed list (ADR-0003) |
| type | What a link is for, as a feature reads it. Open vocabulary |
| label | The few words a person writes for what a relationship is called |
| note (on a link) | Why the link exists, in the person's words. The one deliberate homonym; see below |
| address | Scheme and value; the only thing that says where a link goes |
| identifier | The ULID a note or a vault carries in the world |
| anchor | An identifier marking a place in a note rather than a thing |
| stretch | A run of a note's prose, named by the text standing in it rather than by where it stands (ADR-0034) |

**What the application keeps**

| Word | Means |
|---|---|
| artifact | Data that cannot be reproduced locally, deterministically and for free. Lives in the vault |
| cache | Data that can. Lives outside the vault |
| bought | Data a model made. Lives outside the vault and is kept: addressed by the text it was made from and the recipe it was made under (ADR-0000) |
| index | The cache. Never a SQL index; that word belongs to SQL and stays in SQL |
| source | A thing the index holds text for. A note and a book are kinds of source (ADR-0006) |
| chunk | One window of a source's text, as a row. Both of ADR-0007's sizes are chunks; the large one is the chunk with no parent (ADR-0006) |
| location | Where a chunk sits, in the terms its own format uses. Nullable, and never a key. `start` and `length` are where a chunk is and are not a location either (ADR-0006) |
| hash | Over a file's bytes, which file it is; over a window's text, which chunk it is. Two things, settled below (ADR-0006, ADR-0034) |
| passage | What a search returns: the text around a hit, and where it came from (ADR-0006, ADR-0007) |
| registry | The list of vaults the installation knows |
| scan | One walk of a whole vault |
| refresh | Bringing named notes up to date. Never *reindex*, never *incremental* |
| group | What a scan writes in: one transaction's worth |
| fingerprint | Path, size and modification time — what says a note need not be read again |
| vector | What a model made of one chunk's text. Kept by its text and its recipe; never *embedding* |
| recipe | Everything that decides what a thing made from text is: for a cut, the reader and the sizes; for a vector, where it was made, which model, how wide, where the text was cut off and how it is stored |
| changed | What a write answers when the note on disk is no longer the one the caller read (ADR-0027). Never a *conflict* |
| reload | What a client is told when the vault is to be read again whole: more changed at once than could be followed, or a listener that fell behind (ADR-0023) |
| backlink | A link that resolves here, whichever end wrote it |
| problem | Something that could not be acted on and was not guessed at |
| watch | Following a vault for changes the application did not make |
| hold | How long events are kept before they are acted on |
| trash | Where a removed note is kept: `.trash/` inside the vault (ADR-0027) |

**What reaches a vault from outside**

| Word | Means |
|---|---|
| agent | A program acting on a vault on a person's behalf, through tools (ADR-0026) |
| tool | One operation an agent can call. Never a synonym for a use case |
| client | A consumer of the schema that draws a vault (ADR-0025). Never an agent |
| conversation | One thread of talk with an agent, named by the client and carried in every question of it. Never a *thread*; see below (ADR-0031) |
| session | What the agent's own program calls a conversation it is keeping, named by that program. Never leaves the adapter that started it (ADR-0031) |
| finish | Saying a conversation is over: nothing is asked under its name again, and what the agent kept of it is let go of. Never a *close*; see below (ADR-0031) |
| step | One thing an agent said, did, or stopped for, as the panel is told about it |
| call | What an agent named one use of a tool, so every step reporting it is known to be one |
| kind | What a call does to the vault: `read`, `edit`, `remove`, `move`, `search`. A call that says no more than that it is one is `calling` |
| place | Where a call is working: a note, and the line in it where one is named |
| settings | What a person configures about an installation, and the one file it is in |

**What is drawn**

| Word | Means |
|---|---|
| plex | The focused neighbourhood the product is named for |
| neighbourhood | One note and everything joined to it, seen from that note |
| focus | A neighbourhood's: the note it is seen from. A workspace's: the pane a tab opens into. Never the keyboard's position; settled below |
| node | What is drawn in place of a note |
| edge | A line drawn between two nodes. Two links can be one edge (ADR-0003) |
| seat | Where a node sits relative to the focus: `parent`, `child`, `jump`, `sibling` |
| viewport | The area the plex is drawn into |
| window | The application's window on screen, and nothing else |
| panel | The column beside the plex where a person asks an agent something |
| turn | One thing shown in the panel's conversation: what was asked, what was answered, what is being done |
| voice | Whose turn it is, and so how it is drawn |
| workspace | Everything the window holds open, and how it is split |
| branch | A split of the workspace, drawn as two parts side by side |
| pane | One part of a branch, holding tabs and showing one of them |
| tab | One thing a pane holds open, shown by its title |
| menu | A list of things that can be done, opened on what they are done to |
| unsaved | A tab whose text is not the text in its file |
| stuck | A tab whose file can be neither read nor written: not a note, not text, over the ceiling, or frontmatter that will not parse (ADR-0032) |
| gone | A tab whose name has no file behind it, so its save stopped (ADR-0032) |
| overtaken | A tab whose file no longer holds the prose the tab read, so its save stopped (ADR-0032) |
| keep | Writing an overtaken tab's prose over its file, when the person says so (ADR-0032) |
| take | Replacing an overtaken tab's prose with its file's, when the person says so (ADR-0032) |
| mark | The one word a tab carries beside its title for the state it is in, and what a screen reader reads out: `stuck`, `gone`, `overtaken` or `unsaved`, the first of those that holds |

### Words that were spent twice, and how they are settled

**`role` and `seat`.** A role is written in a file by a person and is one of
five. A seat is where a node sits in the picture and is one of four —
`sibling` among them, which no file ever carries, and without `ref` or
`attachment`, which are not drawn. They are near enough to be confused and
different enough to matter, so they keep different words. A seat is not a
value that can be written to a note.

**`link` and `edge`.** A link is a record in a file; an edge is a line in a
picture, and one edge can be two links (`parent: B` in A and `child: A` in B).
The fold from one to the other is the whole reason both words exist.
*Connection* is neither: it is a connection to the database.

**`identifier`.** The ULID a note or vault carries in the world. The number a
row happens to have inside the index is not an identifier and is not called
one; it does not leave the storage it belongs to.

**`window`.** The application's window. How long the watcher holds events
before acting on them is a *hold*; the area the plex draws into is a
*viewport*; a span of a source's text is a *chunk*. Where ADR-0007 says window
it means how large a chunk is cut.

**`answered`.** Two things, and they are settled apart. A *step* named `answered`
is a tool that has finished, which is what the agent's own stream reports. A
*voice* named `answered` is the agent replying to the person. The first crosses
the wire and belongs to an agent's work; the second never leaves the interface and
belongs to a conversation. Where both could be read, the step is *the tool answered*
and the voice is *the agent's reply*.

**`calling` and `doing`.** A step is `calling` in the core and `doing` on the
wire, and this is the rename a boundary requires: the core says what the agent is
doing, and the wire is read by something drawing a line about it. Every other
step keeps its word across the boundary, and a third name for either of these is
a defect.

**`conversation`, and not `thread`.** One thread of talk with an agent. The
client mints the name, the schema carries it, the port takes it and the adapter
keys what it remembers by it, and all four say `conversation`. *Thread* is the
same thing under a second name, so a client that says thread renames a field on
the way across a boundary. Where the English word is wanted, the phrase is
*thread of talk* and the field is still `conversation`.

**`finish` and `stop`.** A `stop` is one answer given up on: the person presses
stop, and the conversation stays open to be asked again. A `finish` is the
conversation itself being over — the tab closed, or the window went — and
nothing is asked under that name again. Finishing stops whatever is still being
answered in the conversation; stopping an answer finishes nothing. The client
says it, the schema carries it, the port takes it and the adapter drops the
session on it, and all four say `finish`.

*Close* is neither, and is the near-synonym to refuse: the window closes, a tab
closes, and an agent's `Close` ends every child it started. A conversation is
finished.

**`session`.** What the agent's own program calls a conversation it is keeping.
A conversation is the person's and is named by the client; a session is that
program's and is named by that program, and the adapter's whole job here is to
hold one against the other. The word therefore stays on the identifier and on
nothing else: the environment variables that describe somebody else's session
are the ones this application drops, and are named for that.

**`focus`.** Two things, and what carries them settles them apart. A
*neighbourhood*'s focus is the note it is seen from. A *workspace*'s focus is the
pane a tab opens into. A neighbourhood knows nothing of panes and a workspace
knows nothing of notes (ADR-0020), so the two are never read in one sentence.

The keyboard's position is neither, and has its own word: a reader who sees `focus`
in a stylesheet will guess wrong exactly once and be wrong everywhere after.

**`mark`.** A tab's mark is one word about the state of what the tab holds, drawn
beside the title and read out as the tab's own label. Markdown's syntax characters
are markup, and where the editor says mark it is the library's `Decoration.mark`,
which does not leave the module that calls it.

**`changed` and `overtaken`.** `changed` says a file is no longer what somebody
read: a write answers it, and a watcher's event carries the paths it happened to.
`overtaken` is the tab whose save stopped for that reason, and it is a mark, which
is a word about a tab.

**`reload` and `take`.** A client told to `reload` reads the vault again whole. A
`take` is one tab reading one file, and the person asks for it.

**`hash`.** Two, and what each is taken over settles them. A source's `hash` is
over the bytes of a file and says which file it is (ADR-0006). A chunk's `hash` is
over the text of one window and says which chunk it is (ADR-0034). Each is a column
of the table it belongs to, nothing joins one to the other, and a sentence that
says `hash` alone has said nothing.

**`label` and `title`.** A title names a note; a label names a relationship.
Both are drawn, a few pixels apart, which is precisely why they cannot share a
word.

**`group` and `pane`.** A group is what a scan writes in: one transaction's worth
of notes (ADR-0022). A pane is one part of a split workspace. The workspace was
written with the storage word and is renamed, because the two are read side by
side in the same session and one of them is about the database.

**`unsaved` and `dirty`.** A tab whose text is not the text in its file is
*unsaved*. `dirty` names the fraction of a chunk's words that are rubbish, and
that reading is the one a person is shown.

**`position`.** An ordinal — which link, which heading. A line number is a
*line*.

**`remove`.** One act over different things keeps one verb: a note is removed
and a link is removed, never deleted in one place and removed in the other. The
constructive side is allowed two, because the acts differ and the noun already
says which is meant: a note is *created* — a file appears — and a link is
*added* — a record joins a list that was already there.

**`note`, twice.** A note is a file. A link's `note` is why it exists. This one
is kept: the key was chosen for the file format, where it is read by people and
reads naturally, and moving it now would break every vault for a word nobody
outside the file ever says. It is the exception that is written down rather
than the exception that is discovered.

## Consequences

**Positive**

- A name is no longer a matter of taste at each new file. It is looked up.
- A word that has to be translated at a boundary is now visible as a decision
  with a reason, or as a defect.
- A new concept that has no word is a signal: something is being built that was
  never decided.

**Negative**

- Renaming reaches into the wire, which is a contract, and into storage, which
  is a migration. Nothing is deployed, so the cost is paid now rather than
  compounded.
- Vocabulary drifts by default. Nothing enforces this but reading, and the
  first thing to rot will be a word invented in a hurry and never brought back
  here.
- A decision written before its word existed will read oddly against this list
  until it is amended.

## Alternatives considered

**Let each layer keep its own vocabulary and translate at the boundary.**
Rejected: this is what happened, and it is why the same field is `Seat`, `seat`
and `role` across three files that sit in one call chain. A translation is only
safe when a rule forces it, and there was no rule.

**A glossary file rather than a decision.** Rejected: a glossary describes, and
a description that disagrees with the code loses. This binds, and the code is
what changes.

**Record the collisions and rename nothing.** Rejected: the point is not that
the collisions are documented, it is that a reader never meets them. Two of
them — a seat where a role is expected, an identifier where a row number is —
are the shape of a real defect, not an inconvenience.
