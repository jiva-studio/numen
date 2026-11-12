# What the panel's agent can reach

The user types a question into the panel beside their notes, and the panel runs it as one `acp.task` of the agent its connection rows name (see [Settings](settings.md#which-agent-answers)). The plugin that provides the type of the connection starts the program, chooses its tools, its working directory and its environment. This page describes what that agent is given and what is restricted.

## What the agent is given

The tools of this vault: the provider lists the tools of the host its own permissions allow and serves them to the program on an MCP endpoint of its own, on loopback, behind a bearer token, for the length of the session. The program's permission requests are granted for those tools alone; every other request is refused, and the panel shows no prompts. Each call carries the context the asking view gave and the vault it named.

Claude Code is started through its ACP adapter with its own tools off except the web search, and reads no configuration source of this machine or of the vault. A fetch goes to an address given in the text, and a note may have been written by anybody, so the vault's content would decide where the fetch goes; the web fetch is therefore off. Codex and Antigravity run their own tools as their adapters do, and each of their requests for a permission is refused unless it is for a tool of this vault.

The Numen Agent calls the tools of the host itself, with no endpoint between.

## What the agent is told

A task receives the instruction sentences of the tools it is given before the prompt, in listing order, each sentence once and joined with one space, the same sentences `agents-mcp-server` gives. The sentence of `window_tab_list` says that a task saying "this note" means the note in the active tab, which `window_tab_list` returns. The path of that note is not written into the instructions: a note name in a synced vault is set by the sync source, and a tool result is read as data.

`agents-mcp-server` also gives every client that connects to it instructions: one line of its own on what a vault is, then, after an empty line, the sentences each tool it offers lists with it, in listing order, each sentence once and joined with one space. The host lists the tools grouped by plugin in the order of the plugin ids, so the sentences of one plugin are adjacent. A sentence of a tool that is not offered is left out, so a plugin that is disabled, or none of whose tools is offered, adds nothing. The vault tool sentences describe how to link a note and a passage, which tool searches and reads what (`note_search` the notes, `library_search` and `library_read` all indexed files), and that a passage is a truncated excerpt, so the agent reads more before it concludes that a source does not cover a topic. The window tool sentences describe when to read the tabs and when to show a place.

## What an answer links to

A note an answer mentions is named as `[[Harmonic oscillator]]` — the same brackets a note is written with, resolved the same way, and pressing one opens the note in a tab next to the active tab. A name that matches no note is rendered as an unresolved link, and pressing it opens nothing. `[[note://<identifier>]]` identifies one note where a title is shared. See [Links](links.md).

A passage of a book is named as `[the Remuna episode](numen:library%2FA%20Book.pdf?start=62690&length=1246)`: a document's path, percent-encoded, and the text range the search returned. Pressing one opens the document there, with the other passages of that answer highlighted.

The flashcard review window opens neither link type: an answer there cites its sources in plain text, because a link cannot be opened there.

## The tools

Six plugins register the tools an agent can call, each for its own subject. The host lists them for a plugin whose permissions allow their effect, the agent providers serve them to their programs, and `agents-mcp-server` serves them to programs outside Numen. The effect column says whether a tool only reads or also writes; a tool that writes is listed only for a plugin that holds `vault:write`.

`agents-tools-vault`, the notes and files of the vault:

| Tool                 | What it does                                                                                                                                                                                                                                                                                                   | Effect  |
| -------------------- | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | ------- |
| `note_read`          | the whole `text`, the body prose, title, frontmatter and fingerprint of one note, by its path                                                                                                                                                                                                                  | reads   |
| `note_write`         | write or create one note from its `text`; with the fingerprint `note_read` returned, a write over a newer version is rejected                                                                                                                                                                                  | writes  |
| `note_heading_list`  | the headings of one note: level, text, `line` counted from 1 at the first line of the body, and `start`, the byte offset in the body                                                                                                                                                                           | reads   |
| `note_search`        | the notes whose title, path or text match a query, at most 100, 20 by default, with the `line` of the match counted from 1 at the first line of the body                                                                                                                                                       | reads   |
| `note_list`          | the paths and titles of the notes of the vault or of one folder, ordered by path; at most 500, 100 by default, paged with `offset` and `nextOffset`, with the total count                                                                                                                                      | reads   |
| `note_title_list`    | the titles of up to 50 notes by their paths, with the identifier of each note that has one, and the paths that name no note                                                                                                                                                                                    | reads   |
| `note_resolve`       | the paths of every note with one file name, as a wikilink names it; more than one path means the name is ambiguous                                                                                                                                                                                             | reads   |
| `library_search`     | passages of all indexed files (notes, books, PDFs, scans, recordings) by the words and by the meaning, with the path, the title, the `place` (page, CFI and chapter, or seconds and end seconds), the `line` counted from 1, the text and the `uri` of each; narrowed by `kinds` and `folder`                  | reads   |
| `library_read`       | the text of one file from its text version, by page or by byte range, with the `ranges` of the text and the `place` of each (page, CFI and chapter, or seconds and end seconds), the size and the `nextStart` of the next range; a file with no text version is an error until it is recognised or transcribed | reads   |
| `link_list`          | the links written in one note and the notes that link to it                                                                                                                                                                                                                                                    | reads   |
| `link_add`           | write up to 50 links, each to a target `address`, into the frontmatter `links:` of notes, with the fingerprint `note_read` returned; the links for one note are written together, and a stale fingerprint rejects all links for that note                                                                      | writes  |
| `link_update`        | change the role, type, label and note of the links one note has to a target `address`; fields left out are kept; a stale fingerprint is rejected                                                                                                                                                               | writes  |
| `link_remove`        | remove the links one note has to a target `address`, or only the one of a role; a stale fingerprint is rejected                                                                                                                                                                                                | writes  |
| `vault_get`          | the identifier, name, root path and number of notes of the vault of the call, or of the `vault` named by its name, path or identifier                                                                                                                                                                          | reads   |
| `vault_list`         | every vault of the installation: identifier, name and path, with the current vault marked `isCurrent` and a vault whose folder is missing marked `isMissing`                                                                                                                                                   | reads   |
| `vault_problem_list` | the problems the checks find in the vault: `parse`, `frontmatter` and `ambiguous` by default, `dangling` when named; each with the path, a description, and the link target and its candidates where the check is about a link                                                                                 | reads   |
| `file_read`          | a byte range of a text file in the vault, by its path from the vault root; a binary file is rejected with an error that names `library_read`                                                                                                                                                                   | reads   |
| `file_create`        | a new file in the vault, with the given `text`                                                                                                                                                                                                                                                                 | writes  |
| `note_rename`        | rename a note, written in its title key or its file name, and move its file to the title, returning its `path`; the host rewrites the links to the old name                                                                                                                                                    | writes  |
| `note_move`          | move up to 50 notes to a folder, `"/"` for the vault root, one result per path, a path that fails not stopping the others                                                                                                                                                                                      | writes  |
| `note_remove`        | move up to 50 notes to the vault's trash, from which they can be restored, one result per path; a folder is rejected                                                                                                                                                                                           | writes  |
| `vault_add`          | add a folder of this machine to the list of vaults by its absolute path, returning its identifier, name and `path`                                                                                                                                                                                             | writes  |
| `vault_rename`       | change the name of a vault, named by its name, path or identifier                                                                                                                                                                                                                                              | writes  |
| `vault_remove`       | remove a vault from the list without deleting its folder; the current vault and the last vault are rejected                                                                                                                                                                                                    | writes  |
| `vault_open`         | switch the user's window to another vault and return when it shows it, which ends the session's access to the current vault; the current vault is rejected, and a window with unsaved changes keeps its vault, prompts the user and the call returns an error                                                  | writes  |

`agents-tools-window`, the window the user has open:

| Tool                  | What it does                                                                                                                                                          | Effect  |
| --------------------- | --------------------------------------------------------------------------------------------------------------------------------------------------------------------- | ------- |
| `window_tab_list`     | the tabs open in the window with `tabId`, title, path, kind and the `place` the reader reports (page, page count, seconds, CFI, chapter), and the `activeTabId`       | reads   |
| `window_tab_activate` | make the tab with a `tabId` from `window_tab_list` the active tab, and return it as the window reports it; an unknown id is an error                                  | writes  |
| `note_focus`          | make a note the current one of the window, centring the neighbourhood view on it                                                                                      | writes  |
| `source_focus`        | open a file in the window at the passages `library_search` returned, scrolling to the first and highlighting all of them; a note opens whole with nothing highlighted | writes  |
| `note_edit_preview`   | propose a change to a note as a diff in the editor, leaving the file as it is                                                                                         | reads   |

`views-flashcards-studio`, the decks and card models of the vault:

| Tool                                                             | What it does                                                                    | Effect  |
| ---------------------------------------------------------------- | ------------------------------------------------------------------------------- | ------- |
| `card_current_get`                                               | the card shown in the flashcard review window                                   | reads   |
| `card_model_list`                                                | the card models of the vault, at most 50, and the fields each declares          | reads   |
| `card_model_read`                                                | the fields and templates of one card model                                      | reads   |
| `card_read`                                                      | the cards of one deck in deck order, at most 50 a call, or one card by its id   | reads   |
| `card_deck_create`                                               | a deck with no cards                                                            | writes  |
| `card_model_create`                                              | a card model: the fields a card requires, and its templates                     | writes  |
| `card_model_edit`                                                | change a card model's fields and templates in place                             | writes  |
| `card_field_rename`                                              | rename one of a card model's fields everywhere it is used                       | writes  |
| `card_add`                                                       | write one card at the end of a deck or of one of its sections                   | writes  |
| `card_edit`                                                      | write values into one card, leaving the fields the call does not name unchanged | writes  |
| `card_value_remove`                                              | remove one field from one card                                                  | writes  |
| `card_remove`                                                    | remove one card from a deck                                                     | writes  |
| `card_section_add`, `card_section_rename`, `card_section_remove` | write, rename and remove the sections a deck is divided into                    | writes  |

`engines-ocr`, `engines-transcription` and `tools-media-importer`, the text versions of scans, recordings and web addresses:

| Tool                 | What it does                                                                                                                                                                                                | Effect  |
| -------------------- | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | ------- |
| `recognition_create` | start the recognition of a scanned PDF or image in the background and return its task id; a second call while it runs reports `running`, a call on a recognised file reports `done` and the version to read | writes  |
| `transcript_create`  | start the transcription of a recording in the background and return its task id; a second call while it runs reports `running`, a call on a transcribed recording reports `done` and the version to read    | writes  |
| `url_import`         | create a link note for an http or https address, fetch its text before returning, and with `downloadVideo` start a download of the video in the background                                                  | writes  |

A card is named by its card id and never by the place it is in, so the cards stay addressable when a deck is reordered during a run. Every write to a deck takes the fingerprint `card_read` returned, and is applied only to the deck that fingerprint identifies.

`file_read` is how a text file that is not a note is read — a transcript somebody typed, an export, any other text file — and how a file too long to return whole is read one range at a time. A binary file, such as a PDF, is read with `library_read`.

### How a tool is named

A tool is named for the subject it acts on, then what it does: `note_read`, `card_add`, `window_tab_activate`.

## External MCP clients

The tools are served to an external MCP client by the plugin `md.numen.agents.mcp.server`. Its manifest declares the listener `mcp` on the plugin's setting `port` (3333 by default): while the plugin is enabled, the host listens on `127.0.0.1` at that port, binds the new port when the setting changes, starts the plugin when a connection arrives and forwards the connection to the port the plugin reports for `mcp` in its ready line. The plugin stops when it is idle and no connection is open; the port stays bound. The application writes no token file.

A client authenticates with the header `Authorization: Bearer <token>`, compared in constant time with the plugin's setting `token` (a string with no default and `generate: "token"`: the host stores 32 random bytes in hex in it when it loads the enabled plugin and the setting holds no value; the user copies it from the plugin's page in the settings into the client configuration, and may replace it; the host passes it to the plugin on Configure). A token in the query parameter `token` is not read. While the setting is empty the server returns 401 for every request, with `WWW-Authenticate: Bearer realm="numen"`; a changed token takes effect at the next request. A request whose `Origin` header is present and is not an `http` or `https` URL on `localhost` or a loopback IP address gets 403 before the token is checked. The server sends no CORS headers, and an `OPTIONS` request is checked like any other. A client is configured with the header:

```json
{
  "mcpServers": {
    "numen": {
      "type": "http",
      "url": "http://127.0.0.1:3333/mcp",
      "headers": { "Authorization": "Bearer <token>" }
    }
  }
}
```

## What is not covered

**An agent whose tools did not load answers anyway.** A program that takes no MCP server over HTTP gets no tools, and the turn fails; a server that fails while the program runs is reported by the program, if at all.

**A file the agent writes in the vault is read by the user's own agent.** A note it writes stays in the folder, and the user's own agent, started in that folder, reads what is there. Where a note may be written is limited to places a note may live and this application's own folders, so another tool's dot-folder is blocked; the vault's root is not, and a file with the configuration filename of another tool can be written there.

**The options that turn Claude Code's tools off are the adapter's.** They are passed in `_meta.claudeCode.options` of `session/new`, and the adapter can change what it reads in any release without notice. Each release must be checked, and nothing checks it automatically.
