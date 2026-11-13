# Glossary of the product

This file lists the things that exist in Numen, the word a person reads for each, and the similar words that are wrong. A scenario in `features/` uses these words and no others. The canonical term is the word in the interface (`words.ts`, manifests, command titles), not the name in the code. Code names are in backticks. Where the interface uses two words for one thing, both are listed under the entry.

## The words that look alike

- file, note, text: a file is any file in a vault; a note is a markdown file that Numen reads as a note; a text is what is written inside a note, or a text version made from a file.
- vault, folder, workspace, window: a vault is a folder that is added to Numen; a folder is a directory inside a vault; the workspace is the tab layout inside a window; a window shows one vault.
- tab, pane, panel, banner, popover: a tab is one page in the tab bar; a pane is one stack of tabs; a panel is a side area in the flashcards review (and the palette action panel); a banner is the bar inside a note tab; a popover is a small box opened from the title bar or from a link.
- task, notice, message, notification: a task is a row in the task list for work or for a failure; a notice is a message from the core or a plugin that shows as a failed task; a message is a line in the agent conversation; "notification" is not a word of the interface.
- link, wikilink, connection: a link joins two notes and is written in a note; a wikilink is `[[Note Title]]`; a connection is the pair of link records of two notes.
- chunk, vector, passage: a chunk is a piece of text of the index; a vector is the numbers made for a chunk; a passage is a piece of text a search shows.
- transcript, caption, recognition, proofreading: text made from speech, text from a video site, text read from a scan, and a cleaned text.
- card, review card, card model, template, field: a card is written in a deck; a review card is a card with one template; a card model gives fields and templates.
- plugin, provider, engine, tool: a plugin is an add-on; "Provider" is a setting name; an engine is a plugin that does heavy work (speech, OCR, vectors); a tool is an action the agent calls.
- remove, trash, delete, erase: remove moves a file or folder to the trash (and removes a vault from the list); "Move to trash" is reversible; "Empty trash" deletes what is in the trash for good; "Erase vault" moves the vault folder to the trash.

## Vault and files

### vault
- Is: A folder of files that you add to Numen. A window shows one vault. The installation keeps one list of vaults for all windows.
- States: `isMissing` (shown as "Missing"): nothing is at the path now; the vault stays on the list.
- Actions: `app:vault.open` "Open vault", `vault.create` "Create vault", `vault.rename` "Rename vault", `vault.remove` "Remove vault from list", `vault.erase` "Erase vault".
- Texts: "The folder is not deleted"; "The folder moves to the trash"; "Type the vault name to confirm"; "a vault with this name exists"; "the only vault cannot be removed"; "No vault open" (apps/desktop/editor/src/shared/words.ts, entities/vault/words.ts).
- Not: not a "window" and not a "workspace". A folder cannot be added if it is inside a vault of the list or holds one.

### vault switcher
- Is: The list that opens from the vault name in the title bar, to change the vault of this window.
- Texts: "Switch vault", "Search vaults", "No vault found", "Vaults", "Create vault", "Choose a folder" (editor/src/shared/words.ts).

### folder
- Is: A directory inside a vault. Listings show folders first, then files, each by name.
- Actions: "New folder" in the file tree menu (new folder first gets the name "New folder"); `folder.create`.
- Not: "Choose a folder" picks a folder on the computer to be a vault; it is not a folder of a vault.

### file
- Is: Any regular file in a vault: a note, a PDF, a recording. A note is a file; `SourceKind` is `note` or `file` (every other file).
- Actions: `file.open`, `file.move`; "Not moved: a file with this name exists".
- Not: the core says "source" for a file the index holds. A file that is not markdown opens in the tab of the plugin that handles its content type.

### content type
- Is: The kind of a file, like `text/markdown` or `audio/mpeg`. A plugin handles it as `default` (opens without asking) or `alternate` (on request).
- Not: not a note type and not a tab kind.

### trash
- Is: Where a removed file or folder goes so that it can come back: the folder `.trash` of the vault, under the same path.
- Actions: `app:file.remove` "Move to trash"; `trash.empty` "Empty trash"; a vault folder goes to the trash of the computer.
- Texts: "no trash on this system" (entities/vault/words.ts).
- Not: "delete" is not the default action; the default is trash.

### file tree
- Is: A side tab that lists all folders and files of the vault. Tab kind `files`, tab title "Files". Plugin "File Tree Explorer".
- Actions: `tree.open` "Open file tree"; `tree.revealNote` "Show in file tree".
- Texts: "Files", "Vault files" (screen reader), "No files in this vault" (plugins/views-file-tree/frontend/words.ts).
- Not: a row is one folder or one file of the tree (code `entry`); the tab is not the vault.

### file tree menu
- Is: The menu on a row of the tree (right click). The tab adds its entries, the window adds the rest.
- Actions: "New" with "New note" and "New folder"; "Rename"; "Move to trash".
- Texts: "N files" (drag label); "Not moved, names already exist:" (plugins/views-file-tree/frontend/words.ts).

## Notes and tabs

### note
- Is: A markdown file of a vault that Numen opens in a note tab. It has a path, a title, an id, frontmatter, headings, links and a body.
- Actions: `note.open` "Open note", `note.openBeside` "Open note beside", `note.goTo` "Go to note", `note.create` "Create note", `note.rename` "Rename note", `note.copyPath` "Copy note path".
- Texts: "No active note"; "Broken links in:" (editor/src/shared/words.ts).
- Not: the conflict texts say "file" in one place and "note" in another. The core says "source" (index side). A "document" is the PDF reader word; a "passage" is a piece of text found by search.

### note title and filename
- Is: The title is the name a note is shown by (the `title` key of the frontmatter, else the filename without `.md`). The filename is the name of the file.
- Texts: setting "Sync title and filename": "Renaming one also renames the other." (libs/core/adapter/settings/app.json).

### note type and frontmatter
- Is: The frontmatter is the YAML block at the top of a note. Its `type` key tells the kind of note: none is an ordinary note; plugins add `deck`, `card-model`, `preset`, `link`.
- States: unreadable frontmatter: `unreadable`, text "frontmatter unreadable"; "text cannot start with frontmatter" (`bodyUnwritable`).
- Not: not a content type and not a tab kind.

### note tab
- Is: A tab of kind `note` (tab title "Note"; editor "Markdown Note Editor") that shows and edits one note. It saves by itself after a quiet time.
- States: `loading`, `clean`, `unsaved` (also while `saving`), `stale` (file changed on disk; saving paused), `gone` (file removed from the vault; saving paused, text kept), `stuck` (an error blocks read or write).
- Actions: `keepLocal` "Keep my version"; `takeFile` "Use disk version"; "Recreate it" (for `gone`).
- Texts: "not a note", "not a text file", "note too large for the editor", "vault unreachable, note not saved" (editor/src/entities/note/model/buffer/words.ts).
- Not: the editor is drawn by the `notes` plugin; the state machine is in the window.

### file conflict
- Is: The case where the file of an open note changed or vanished on disk. It is not an error and not a notice.
- States: `stale`, `gone`.
- Texts: "Saving paused: the file changed on disk."; "Saving paused: the note was removed from the vault. Your text is kept."; "Keep my version"; "Use disk version"; "Recreate it" (plugins/notes/frontend/words.ts).
- Not: older copies in editor/src/entities/note/words.ts and buffer/words.ts differ in case ("recreate it", "saving paused: the note changed on disk").

### conflict banner
- Is: The bar at the top of a note tab (class `caution`, `role="alert"`) that shows the file conflict text and its buttons.
- Actions: the three buttons above; "Dismiss banner" (aria label).
- Not: not a notice, not a task, not a toast.

### unsaved changes prompt
- Is: A prompt at the bottom of the window when you quit and notes have a file conflict. One row per note.
- Actions: `keep` "Keep my version", `take` "Use disk version", `later` "Later".
- Texts: "Saving paused: the files of these notes changed on disk." (editor/src/shared/words.ts).

### tab
- Is: One page in a pane, with a title in the tab bar (the "strip"). It has a kind. Kinds: `note`, `settings`, and the kinds plugins give (Files, Plex, Agent, Deck, Card Model, Preset, Flashcards, EPUB Reader, PDF Reader, Audio Player, Record Audio, Link).
- States: a mark beside the title: `unsaved`, `stale`, `gone`, `stuck`.
- Actions: `app:tab.close` "Close tab"; open; move to another pane.
- Not: a tab is not a note and not a file. The core saves the open tabs and the active one.

### pane
- Is: One stack of tabs of which one shows. The pane in focus is the focused pane; "beside" opens in a pane next to it.
- Not: not "panel". The code says pane.

### workspace
- Is: The area of the window that holds all panes: a tree of splits. It opens with a main pane and, when asked, one aside pane.
- Actions: drag a tab to a side of a pane (`left`, `right`, `top`, `bottom`, `center`) to split or move.
- Not: not the vault and not the window.

### tab link
- Is: A group of tabs that follow each other (for example a text and its recording), also across windows (`TabLinkGroup`).
- Not: no interface text found.

### plugin unavailable pane
- Is: The page in a tab whose plugin is off, stopped, missing or incompatible.
- Texts: "Plugin unavailable", "Plugin incompatible", "Plugin disabled", "Plugin not installed", "Plugin not running", "Starting the plugin for this tab…"; buttons "Enable <name>", "Install <name>", "Open Settings"; "Could not restore this tab" is another pane (editor/src/entities/tab/model/unavailableView.ts).

### live preview
- Is: A mode of the note editor that shows markdown as formatted text, except on the line you edit.
- Texts: settings "Soft Wrap" (default on), "Line Numbers" (default off) (plugins/notes/manifest.json).

## Window and status

### window
- Is: One app window with its own vault, tabs and tasks. Two kinds: the editor and the flashcards window. It holds a title bar and a workspace, and a status bar where the plugin "Status bar" is enabled.
- Texts: `appName` "numen" (editor/src/shared/words.ts).
- Not: not a vault; two windows can show two vaults.

### title bar
- Is: The bar at the top, drawn by the plugin "Title bar": the vault name (opens the vault switcher), buttons for the commands that plugins place in the slot `titleBar.commands`, the status items that plugins place in `titleBar.status` (tasks, index stats, update), window controls.
- Texts: "Title bar", "More commands", "Minimize", "Maximize", "Restore", "Close", "Switch vault" (plugins/views-title-bar/frontend/lib/words.ts).

### welcome screen
- Is: The screen of a window that has no tab. It lists entry points and the vaults.
- Texts: "Search the vault", "Show commands", "Create note", "Settings", "Vaults", "Current", "Missing", "Create vault", "Choose a folder", "No start note".

### command palette
- Is: The pop-up box where you type to find notes or run commands. Two entries: search the vault and show commands (Ctrl or Cmd plus P).
- Texts: "Search the vault", "Show commands", "Type a command", "Search notes", "Actions", "Search actions", "No actions found", "Text search only: no embedding model set" (editor/src/shared/words.ts).
- Not: not a "menu"; menus hold the same commands.

### palette step
- Is: One screen of the palette. A command may ask for several values, one step each: `commands`, `name`, `note`, `vault`, `choice`, `confirmation`, `retypedName`, `arguments`.
- Texts: "Type the note name to confirm", "This cannot be undone", "Empty the trash of", "Keep trash", "Erase", "Keep vault", "Remove", "No changes".

### command
- Is: A named action with an id (`owner:object.action`, the window's own owner is `app`) and a title. It shows in the palette, menus and title bar. Plugins declare commands in the manifest.
- Texts: groups "Note", "File", "General", "Settings", "Vault"; "<title>: done"; "Not saved: resolve the file conflict in its tab first".
- Not: a tool of the agent is not a command.

### status bar
- Is: The bar at the bottom, drawn by the plugin "Status bar", which is disabled by default. It holds the commands, submenus and views that plugins place in `statusBar.left` and `statusBar.right`.
- Texts: "Status bar" (plugins/views-status-bar/frontend/lib/entries.ts).
- Not: the summary of tasks and the index stats are in the title bar.

### task
- Is: One row in the task list: work the app does behind the window, or a failure to tell about. Work has progress (`completedCount` of `total`, counted in `items`, `bytes` or `seconds`).
- States: `running`, `failed` (has an error), `cancelling`, `completed`, `notice`.
- Actions: "Cancel task" (then "Cancelling"), "Retry task", "Dismiss", "Configure in settings".
- Texts: titles "Indexing the vault", "Indexing", "Update index", "Read embedding model", "Import files", "Plugin stopped", "Downloading <dependency>"; "<Name> stopped: <reason>" (editor/src/shared/words.ts).
- Not: three definitions in the code.

### task list
- Is: The popover from the task status in the title bar that lists the tasks, newest first. The task status is a chip that says "Idle" or "<n> of <m>".
- Texts: "Background tasks"; "No active background tasks"; "Close" (plugins/views-tasks/frontend/lib/words.ts).

### notice
- Is: A message of the window, the core or a plugin with a severity (`info`, `warning`, `error`). It has no screen of its own: a warning or an error shows as a failed task; `info` is dropped. Example: a bad value in `numen.json` is ignored with a notice titled "Read settings".
- Texts: "Vault changes not watched", "Vault unreadable", "Themes unavailable", "Theme unreadable".
- Not: not a toast, not a banner, not a "notification". The e2e spec says "the window shows no corner notices".

### fatal start report
- Is: The message shown when Numen cannot start.
- Texts: "Numen can't start", "Numen can't read its settings", "Numen can't open its index", "The settings file is not valid JSON.", "This file is not a Numen index." (apps/desktop/app/fatal_report.go).

## Search and index

### search
- Is: A search of the active vault. Two questions: names (titles of notes and headings) and passages (text). Modes `words`, `meaning`, `names`, `hybrid`.
- Texts: result groups "Names", "Text", "Meaning"; "No results found"; "Search failed"; "Semantic index not built yet"; row actions "Open note", "Open note beside", "Open at this heading"; when nothing matches: "Create note", "Create as child", "Create as parent", "Create as jump" (editor/src/features/command-palette/words.ts).
- Not: "Meaning" is the search by vectors; with no embedding model only text search runs.

### passage
- Is: A piece of text a search found, with the place it came from (path, page, chapter, line). In a reader it is the place a link or a highlight points to.
- Not: a chunk is inside the index; a passage is what the person sees. A "quote" is the copied text.

### index
- Is: The cache Numen builds from a vault so it can search. It is rebuilt from the vaults if lost.
- States: scan `is_ready`; `is_embedding` is false when there is no model (words only).
- Texts: "Numen can't open its index"; "Update index".

### chunk and vector
- Is: A chunk is a piece of text the index cuts a note or file into. A vector is the numbers a model makes for a chunk, used by search by meaning.
- Texts: "Pieces of text the index cuts the notes and files into" (plugins/tools-index-stats/frontend/words.ts).
- Not: "embedding" is the act; a vector is the result.

### index stats
- Is: A title bar item that shows how many vectors the index holds, with a popover of numbers. Plugin "Index stats".
- Texts: "Index stats", "vectors", "indexing"; rows "Vectors", "Chunks", "Waiting for a vector", "Notes and files", "Model", "Recipe", "Vector size", "Performance profile" ("Eco", "Balanced", "Maximum"), "Index file size".
- Not: not a notification.

### search relevance
- Is: A setting for how close a note must be to your search to appear. Plugin "Search Relevance", settings view `floor`.
- States: "Auto" or "Manual"; source "Calibrated on your vault", "Model profile", "Default", "Set by you"; "Not enough notes yet (needs ≥ 20)".
- Actions: "Recalculate", "Check" a query.
- Texts: "Relevance threshold", "Manual threshold", "Check a query" (plugins/tools-search-relevance/frontend/floor/words.ts).
- Not: the code says `floor`; the interface says threshold and relevance.

## Links and the graph

### link (between notes)
- Is: A connection from one note to another, written in a note. It has a role (`parent`, `child`, `jump`, `ref`, `attachment`), a description and an arrow. In the plex it is drawn as an edge.
- States: `dangling` (matches no note), `ambiguous` (matches more than one note).
- Actions: "Add link", "Remove link", "Add reverse direction"; "Done", "Cancel".
- Texts: "Link description"; "No arrow"; "Arrow to <title>"; "The link between these notes will be removed."; "Links not created:" (plugins/views-plex-graph/frontend/words.ts).
- Not: not the imported "link". The pair of records of two notes is a connection (`parent: B` in A and `child: A` in B).

### wikilink
- Is: A link written in the text of a note as `[[Note Title]]`.
- Not: only the README of the notes plugin uses this word.

### completion
- Is: The list of suggestions the editor of a note shows while a link `[[…` is typed.

### link inspector
- Is: The small popover of the plex that edits the link between two notes. It opens on a click on the line of the pair. The header names the two notes. It has one row for each record of the pair (up to two, one for each direction); a row has an arrow button ("No arrow", "Arrow to <title>"), a field "Link description" and a trash button "Remove link". It has no field for the role: the arrow button sets it (an arrow to the second note is a child, an arrow to the first is a parent, no arrow is a jump). The button is read as "<arrow now>. Click for: <next arrow>".
- Actions: "Add link" (when the pair has no record), "Add reverse direction" (when it has one), "Remove link" (the trash of a row), "Done", "Cancel". Removing the last row shows "The link between these notes will be removed."; "Done" then removes the link.
- Not: not the quick link.

### reverse link
- Is: The second record of a pair of notes: the one the other note writes, added with "Add reverse direction" in the link inspector.
- Not: not a new connection; both records are one connection.

### quick link
- Is: The small popover of the plex that opens when you drag from a node to empty space, to pick or create the note to link.
- Texts: "Note to link"; "Search or type note title…"; `Create "<title>"`; "No matching notes".

### plex
- Is: A picture of one note in the middle (the focus) and the notes joined to it, as boxes (nodes) and lines (edges). Tab kind `plex`, tab title "Plex", plugin "Plex Graph".
- States: "Loading…".
- Actions: `graph.open` "Open plex graph"; `graph.showNote` "Show in plex"; click a node to move the focus; drag a line to link.
- Texts: settings "Nest headings under notes", "Headings per note" (plugins/views-plex-graph/frontend/words.ts).
- Not: "graph" and "map". A node is a note in the plex.

### reference note
- Is: A note in the plex that the text of the focus note links to with a wikilink, or that links to the focus note in its text. Its node is read as "<title>, reference"; its line has an arrow at the end the link points to, and the line is read as "Line from <from> to <to>, arrow to <title>". The setting "References" chooses which are shown: "Off", "Outgoing", "Outgoing and incoming", "Incoming only".
- Not: not a parent, child or jump; those come from the `links` of the frontmatter.

### name under a note
- Is: The title of a note that a related note is related to, written in small text under (or over) that related note. A name is a button, read as "<name>, name under <title>"; a click on it moves the focus to that note. It is mouse-only: no key reaches it. The setting "Show notes related to related notes" shows or hides them; "Names per note" limits them.
- Not: not a node of the plex.

### relation
- Is: The place of a note compared to the focus note: `parent` (above), `child` (below), `jump` (left and a little below), `reference` (left and a little above), `sibling` (right, derived, never written). The plex reads a line dragged up as a parent, down as a child and left as a jump.
- Actions: "Create child note", "Create parent note", "Create jump note". No "Create sibling note".
- Texts: drop labels "as child", "as parent", "as jump".
- Not: a role is what a link is for; a relation is where a note stands.

### node menu
- Is: The menu on a node of the plex.
- Texts: "Open note", "Copy note path", "Create child note", "Create parent note", "Create jump note", "Rename note", "Move to trash" (plugins/views-plex-graph/frontend/words.ts).

## Reading

### book
- Is: An EPUB file read in a tab. Tab kind `reader:reader-epub`, tab title "EPUB Reader". It holds chapters, contents, pages, passages.
- Actions: "Previous page", "Next page", "Zoom in", "Zoom out", "Show contents", "Hide contents".
- Texts: "Pages", "Page", "N of M", "N pages left in chapter", "Loading…" (plugins/readers-epub/frontend/words.ts).
- Not: for a PDF the code says "document".

### chapter and contents
- Is: A chapter is a part of a book named in the contents. The contents is the list of chapters that you click to go there.
- Texts: "Contents"; "Find in contents"; "No table of contents."; "N pages left in chapter" (plugins/readers-epub/frontend/words.ts).
- Not: "section" and "table of contents".

### page
- Is: One screen of a book or one sheet of a PDF. A PDF page has a fixed number; the pages of a book depend on the text size.
- Texts: "Page", "Pages" (the thumbnail list).

### pdf document
- Is: A PDF file read in a tab. Tab kind `reader:reader-pdf`, tab title "PDF Reader". May have a recognised text and a proofread text.
- Actions: "Previous page", "Next page", "Zoom in", "Zoom out"; setting "Default Zoom" (`page`, `width`, `100%`).
- Not: the user says "PDF"; the code says "document".

### highlight
- Is: A coloured mark on a passage of a book or PDF. A passage linked from a note is also marked.
- Not: no highlight label in `words.ts`.

### recording
- Is: An audio or video file played in a tab with its transcript. Tab kind `reader:media-viewer`, tab title "Audio Player".
- States: "Media playback not supported here."; "Recording failed to play."; "Loading stopped."; "Recording unreadable."; "Recording format not supported."; "Loading…".
- Texts: plugins/views-media-viewer/frontend/words.ts.
- Not: the recorder is another tab.

### recorder
- Is: A tab that records voice from the microphone and saves it into the vault. Tab kind `recording`, tab title "Record Audio". Command `recording.create` "Record audio".
- States: `idle`, `recording`, `paused`, `stopped`.
- Actions: "Start recording", "Pause recording", "Resume recording", "Stop recording", "Save audio recording", "New recording".

### transcript
- Is: The written words of a recording, with times. The current line follows the audio.
- States: "No transcript yet."; "Transcribing…".
- Actions: `transcript.create` "Transcribe recording" (button "Transcribe"); "Proofread"; `transcript.delete` "Delete transcript"; "Follow".
- Texts: "Transcript" (plugins/views-media-viewer/frontend/words.ts); plugin "Audio/Video Transcription" (plugins/engines-transcription/manifest.json).
- Not: not "subtitles"; the agent chat key `noTranscript` means "No messages yet".

### text version
- Is: A text made from a file by a plugin: a transcript, a recognition, an article, a proofread text. Shown and indexed when it is the live one.
- Actions: "Transcribe recording", "Recognise text", "Proofread text", "Delete transcript", "Delete recognition", "Delete proofreading".
- Not: the person reads the text, not the word "version".

## Agent

### agent
- Is: The AI helper that answers questions about the vault and can edit notes. A provider plugin supplies it.
- States: `reachable` or unreachable; no agent set up reads "No agent".
- Texts: "Agent", "Agent unreachable.", "No agent", "No agent is set up.", "Open settings", "The agent returned no answer." (plugins/views-agent-chat/frontend/words.ts).
- Not: manifests say "AI assistant"; the interface says "agent".

### agent provider
- Is: A plugin that connects one kind of AI program as an agent (capability `ai.agent.provider`): "Numen Agent", "Claude Code Agent", "OpenAI Codex Agent", "Antigravity Agent", "ACP Agents".
- Texts: settings "Chat API", "Model", "Temperature", "Max Tokens", "Max Steps", "Executable Path", "Hooks & skills" (plugins/agents-*/manifest.json).
- Not: not the setting "Provider".

### agent tab
- Is: The tab where a person chats with the agent. Tab kind `agent`, tab title "Agent", plugin Agent Chat.
- Actions: `chat.open` "Open agent chat"; `note.ask` "Ask agent about note".
- Texts: "No messages yet", "Thinking", "Stopped", "Not sent" (plugins/views-agent-chat/frontend/words.ts).

### conversation, turn and tool call
- Is: The conversation is the list of messages in an agent tab. A turn is one entry: `prompt` (the person), `response` (the agent), `doing` (a tool line). A tool call is a quiet line that shows a tool the agent runs, with its name (`note_search`).
- States of a turn: `settled`, `arriving`, `failed`.
- Not: the code also says thread, exchange, work line; the person reads "messages".

### composer
- Is: The field where the person writes to the agent, with the model and reasoning selectors and one round button.
- Actions: "Send", "Stop the agent", "Open settings".
- Texts: "Ask about the vault", "Write a message", "Model", "Search models", "Reasoning" (plugins/views-agent-chat/frontend/words.ts).
- Not: "Thinking" is the wait line; "Reasoning" is the level.

### agent tool and mcp server
- Is: An agent tool is an action the agent calls (`note_read`, `note_write`, `library_search`, `link_add`). Two plugins give them: "Vault Knowledge Tools" and "Window Interaction Tools". The "Model Context Protocol Server" lets other AI apps use the open vault (settings "Port" 3333, "Token").
- Not: a tool is not a command.

## Cards

### flashcards window
- Is: The separate window where a person reviews cards. Title "numen" and "flashcards". It lists vaults, then decks and presets, then a session.
- Texts: "Vaults", "Loading vaults…", "No plugin views.", "Appearance", "Choose a theme or mode" (apps/desktop/flashcards/src/shared/words.ts); "Counting the cards" for a vault without a count.
- Not: the tab kind `flashcards.review` has the title "Flashcards".

### deck
- Is: A note of type `deck` that holds cards, grouped under sections. It names a preset ("Scheduled by"). Tab kind `deck`, tab title "Deck".
- Actions: `deck.create` "Create deck" (argument "Title"); the conflict buttons of a note tab.
- Texts: "Scheduled by", "Not a deck.", "This vault holds no deck.", "No preset set. Scheduled by the defaults." (plugins/views-flashcards-studio/frontend/cards/words.ts).
- Not: a deck is a note, not a folder. Its conflict text says "file".

### card
- Is: One question and answer written in a deck. It has a card id, field values and a card model.
- Actions: "Add a card", "Remove", "Reorder".
- Texts: "Card", "Card model", "Nothing in it"; "No card model called X".
- Not: what the review schedules is a review card.

### review card
- Is: One card with one template. A card model with two templates gives each card two review cards.
- Not: the interface says "card"; "review card" is a code and docs name.

### card model
- Is: A note of type `card-model` that says which fields a card has and how it looks. Tab kind `card-model`.
- Actions: `cardModel.create` "Create card model"; "Add a field", "Add a template".
- Texts: "Fields", "Templates", "No fields yet", "A field needs a name", "That name is taken" (.../ui/cards/lib/card-model.ts).
- Not: old name in code and stories: "cut". Not the AI "model".

### template, field and section
- Is: A template is one front and back of a card model (fields go in braces). A field is a named slot of a card model. A section is a named group of cards in a deck, written as a heading.
- Texts: "Front", "Back", "Preview", "Insert"; "Add a section", "A section needs a name".
- Not: in a preset, `Field` means a setting row.

### preset
- Is: A note of type `preset` that sets how many cards a day a deck gives and the goal. Tab kind `preset`.
- Actions: `preset.create` "Create preset"; `preset.open` "Open preset".
- Texts: "Goal" with "Minutes per day", "Retention", "Target date"; "New per day", "Reviews per day", "Budget unit", "Learned when", "Overdue share", "Load by weekday" (plugins/views-flashcards-studio/frontend/preset/words.ts).
- Not: "load" is the share of the daily budget on a weekday.

### stop reason
- Is: The short reason a preset gives no cards today.
- States: "no time budget", "no cards per day", "no target date", "target date <day> passed", "no load on <weekday>", "nothing due today", "daily limit reached", "no new cards".
- Texts: "No cards scheduled: <reason>."

### review session
- Is: The screen where a person answers due cards one after another, with side panels "Source notes" and "Ask about this card".
- Actions: "Show the answer" (space), answer 1 to 4, take back (u), leave (esc), "Back to the decks".
- Texts: "N left", "N answered.", "Nothing left today.", "Done today" (.../review/pages/session/ui/Session.vue).

### rating
- Is: How well a card was recalled: "Again" (1), "Hard" (2), "Good" (3), "Easy" (4). Each button shows the time the card would wait.
- Not: the frontend says `Grade`, the backend `Rating`, the person reads "answer".

### schedule, due and heatmap
- Is: The schedule is the next day a card is due, computed by FSRS from the answers. Due is a card ready today, and the count of such cards. The heatmap is a grid of days with the cards answered and the cards ahead.
- Texts: setting "Day starts at" (default 04:00); "Counting the cards".
- Not: "left" is what remains in a running session; "overdue" is due on an earlier day.

## Importing and engines

### imported link
- Is: A note of type `link` that holds a web address and the text fetched from it. Made by plugin "Media Importer". Tab kind `link`, tab title "Link".
- States: "Fetching the page…", "Copying the video…", "No text found at this link.", "The text of this link was deleted.", "Link unreadable".
- Actions: `link.import` "Import from URL" (argument "The address"), "Download text again", "Download video", "Delete text", "Delete downloaded video".
- Texts: "This note has no web address" (plugins/tools-media-importer/frontend/words.ts).
- Not: not a link between notes and not Numen Link (sync).

### link import
- Is: A page gives article text; a video gives its captions as a transcript, plus an optional copy of the video. Needs yt-dlp and ffmpeg.
- Texts: settings "Caption languages", "Automatic captions", "Maximum download size" (plugins/tools-media-importer/manifest.json).
- Not: "captions" are the site's subtitles; a "transcript" is made from speech.

### recognition
- Is: Text read from a scanned PDF or an image. Plugin "Document OCR".
- Actions: `recognition.create` "Recognise text"; `recognition.delete` "Delete recognition".
- Not: "Recognise" is British.

### proofreading
- Is: Cleaning of a transcript or scanned text with a language model, in batches of lines. Plugin "LLM Proofreading".
- Actions: `proofreading.create` "Proofread text"; `proofreading.delete` "Delete proofreading".
- Texts: settings "Provider", "Service URL", "Model", "Batch size", "Overlap", "In-flight requests", "Max edit distance" (plugins/tools-proofreading/manifest.json).

### embedding provider
- Is: Where texts are turned into vectors. Plugin "Vector Embeddings" (capability `embedding`).
- Texts: setting "Provider": Local, OpenRouter, OpenAI, Custom service; "Model": "E5 small (default, CPU, 118 MB)", "Harrier 270M (better, 344 MB)", "BGE-M3 (large, 569 MB)" (plugins/engines-embeddings/manifest.json).

## Settings and plugins

### settings
- Is: Values kept in the file `numen.json`, shown in a tab of kind `settings`, one per window. Pages: "System", "Secrets", a list of plugins.
- Actions: `settings.open` "Open settings".
- Texts: "System": "Theme, scale and filename sync."; "Theme", "Light or dark", "Interface size", "Reading size", "Window buttons", "Performance profile"; "Setting not saved:" (editor/src/pages/settings/words.ts, libs/core/adapter/settings/app.json).
- Not: an invalid value is ignored with a notice; an invalid file stops the start.

### secret
- Is: A private value, such as an API key, kept on this machine. It is in groups; a plugin sees only the secrets it is allowed.
- States: "Value is set", "No value yet", "Not saved. Press Enter to try again."
- Actions: "Add secret", "Add group", "Rename group", "Delete group", "Delete secret", "Move up", "Move down".
- Texts: columns "Name", "Value", "Used by" (editor/src/pages/settings/words.ts).
- Not: a secret is not a setting.

### plugin
- Is: An add-on with a manifest. It gives commands, tab kinds, settings, status views, themes, note types or models.
- States: badge "Enabled", "Disabled", "Stopped", "Not installed".
- Actions: "Enable", "Disable", "Retry", "Install", "Configure", "Delete data", "Details", "All plugins".
- Texts: "Search plugins…", "No plugins installed.", "No plugins found." (libs/plugin-manager/src/words.ts).
- Not: a dependency is a program a plugin needs, not a plugin.

### theme
- Is: A set of colours for the windows. Plugin "Preset Themes" has twelve: Amber, Ayu, Catppuccin, Cobalt2, Dracula, GitHub, Gruvbox, Monokai, Nord, One Dark, Solarized, Tokyo Night.
- Texts: "Built-in", "Custom", "Mode" with "System", "Light", "Dark" (apps/desktop/flashcards/src/shared/words.ts).
- Not: the mode (light or dark) is not the theme.

### update and release channel
- Is: An update is a newer build of Numen. Plugin "Update Notifier" only tells; it does not install. The channel is which builds are announced: "Stable", "Beta", "Custom".
- Actions: "Download", "Release notes", "Dismiss".
- Texts: "Update available", "Released on" (plugins/tools-updater/frontend/words.ts); the chip in the title bar shows the version and its tooltip begins "Update available: v" (plugins/tools-updater/frontend/model/tooltip.ts).
