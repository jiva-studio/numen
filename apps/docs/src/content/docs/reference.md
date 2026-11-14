---
title: Settings reference
description: Every key numen.json can hold, section by section, generated from the application itself.
---

Every key `numen.json` can hold. The pages under Settings say which of these are worth touching and why; this one lists every key.

The first level of the file is the version `v`, the owner `app` for the settings of the application, and one object for each plugin, named by the plugin's id. Each heading below is either a section of `app` or the id of a plugin. The keys under a section of `app` are written the way they are written inside that section — `textScale` under `app.appearance` is `app.appearance.textScale` if you write it out in full. The keys under a plugin id are the keys of that plugin's object.

A key you leave out keeps its default, so a file naming one setting is a complete file. A key numen does not know, a value of the wrong type or a number outside its range is ignored with a notice, and the file is not rewritten because of it. Under `app.agent`, each program an agent plugin lists has a section named after it, `app.agent.<program>`, holding `command` and `model`; what a program's plugin reads for itself, such as `readsHooksAndSkills` for Claude Code (see [the agent page](/agent/)), is a setting of that plugin in its own object, named by its id. A row with no kind beside it is a group holding the keys under it.

Two things are not settings and are not in here: which files count as notes, and what a vault's own folder is called. Both are set on the command line, by [`numen-cli`](/cli/#where-it-puts-things).

<!-- BEGIN AUTOGEN -->
| | | |
| --- | --- | --- |
| `v` | a number | the format version of the file. Nothing reads it yet. |
| `app` |  | holds the settings of the application. |
| `<plugin id>` |  | holds the settings of one plugin, next to its `enabled`. |

### `app.appearance`

| | | |
| --- | --- | --- |
| `theme` | text | the stylesheet the window is drawn with. The name of an installed theme. |
| `mode` | text | which half of a color pair the theme is read as. A theme that declares light and dark itself leaves this nothing to choose. One of `system`, `light`, `dark`. |
| `interfaceScale` | a number | scale of controls and interface text. 1 is the default. From 0.8 to 2. |
| `textScale` | a number | scale of note and book text. 1 is the default. From 0.8 to 1.75. |

### `app.naming`

| | | |
| --- | --- | --- |
| `syncTitleAndFilename` | yes or no | renaming one also renames the other. |

### `app.performance`

| | | |
| --- | --- | --- |
| `profile` | text | how much of the hardware indexing and models may use. One of `eco`, `balanced`, `maximum`. |

### `md.numen.agents.acp`

The settings of the plugin ACP Agents.

| | | |
| --- | --- | --- |
| `requestTimeout` | a number | seconds to wait for an answer or a new session; empty waits 120. From 1 to 3600. |

### `md.numen.agents.antigravity`

The settings of the plugin Antigravity Agent.

| | | |
| --- | --- | --- |
| `executablePath` | text | path to the Antigravity CLI; empty finds it automatically. |
| `requestTimeout` | a number | seconds to wait for the program to answer a request from a connection. Without a value, a request waits 120 seconds. From 1 to 3600. |

### `md.numen.agents.claude`

The settings of the plugin Claude Code Agent.

| | | |
| --- | --- | --- |
| `executablePath` | text | path to the Claude Code CLI; empty finds it automatically. |
| `readsHooksAndSkills` | yes or no | lets Claude Code read the hooks, skills, standing instructions in CLAUDE.md and plugins configured for this person. Off by default. A hook is a shell command Claude Code runs itself. What a vault contains is never read. |
| `requestTimeout` | a number | seconds to wait for the program to answer a request from a connection. Without a value, a request waits 120 seconds. From 1 to 3600. |

### `md.numen.agents.codex`

The settings of the plugin OpenAI Codex Agent.

| | | |
| --- | --- | --- |
| `executablePath` | text | path to the OpenAI Codex CLI; empty finds it automatically. |
| `requestTimeout` | a number | seconds to wait for the program to answer a request from a connection. Without a value, a request waits 120 seconds. From 1 to 3600. |

### `md.numen.agents.mcp.server`

The settings of the plugin Model Context Protocol Server.

| | | |
| --- | --- | --- |
| `port` | a number | port on 127.0.0.1 MCP clients connect to, at /mcp. From 1024 to 65535. |
| `token` | text | the bearer token MCP clients send with each request. The plugin makes one on its first start. An empty token rejects every request. |

### `md.numen.agents.numen`

The settings of the plugin Numen Agent.

| | | |
| --- | --- | --- |
| `temperature` | a number | lower is steadier, higher is more varied. From 0 to 1. |
| `maxTokens` | a number | most tokens one answer may have. From 256 to 128000. |
| `maxSteps` | a number | most model requests one task may make. From 1 to 1000. |

### `md.numen.connections.llm`

The settings of the plugin Language model services.

| | | |
| --- | --- | --- |
| `requestTimeout` | a number | seconds to wait; empty waits 120, or 60 for embeddings. From 1 to 3600. |

### `md.numen.engines.embeddings`

The settings of the plugin Vector Embeddings.

| | | |
| --- | --- | --- |
| `source` | text | where texts are turned into vectors. One of `local`, `connection`. |
| `model` | text | local embedding model from Hugging Face; it needs a profile. |
| `connections` | text | a new connection or model reindexes the vault. |
| `minChunkLength` | a number | the fewest characters a piece of text needs to get a vector. Shorter pieces are found by their words only. From 0 to 1000. |
| `chunkTokens` | a number | largest piece of a note, in tokens, that gets one vector. From 16 to 8192. |

### `md.numen.engines.ocr`

The settings of the plugin Document OCR.

| | | |
| --- | --- | --- |
| `model` | text | text recognition model. One of `pp-ocrv6-tiny`, `pp-ocrv6-small`, `pp-ocrv5-latin`, `pp-ocrv5-eslav`, `pp-ocrv5-cyrillic`, `pp-ocrv5-devanagari`, `pp-ocrv5-greek`, `custom`. |
| `customModelPath` | text | paddleOCR text recognition model in ONNX format. |
| `customDictPath` | text | a UTF-8 text file with one character per line. Leave it empty when the model carries its own dictionary. |
| `dpi` | text | resolution a page is rendered at before recognition. One of `150`, `200`, `300`, `400`, `600`. |
| `recognizeAutomatically` | yes or no | recognize every PDF and image in the vault that has no text. |

### `md.numen.engines.transcription`

The settings of the plugin Audio/Video Transcription.

| | | |
| --- | --- | --- |
| `model` | text | downloaded from Hugging Face. Parakeet v3 recognizes 25 European languages, Parakeet v2 English only. One of `parakeet-tdt-0.6b-v3-int8`, `parakeet-tdt-0.6b-v2-int8`. |
| `transcribeAutomatically` | yes or no | transcribe every recording in the vault that has no transcript. |

### `md.numen.notes`

The settings of the plugin Notes.

| | | |
| --- | --- | --- |
| `softWrap` | yes or no | wrap long lines to fit the editor viewport. |
| `lineNumbers` | yes or no | show line numbers in the editor margin. |
| `tabSize` | a number | spaces a Tab inserts. From 1 to 8. |
| `fontFamily` | text | a comma-separated list of families is allowed. A name with spaces needs no quotes. |
| `lineHeight` | a number | space between lines, as a multiple of the font size. From 1.2 to 2.4. |
| `highlightActiveLine` | yes or no | mark the line the caret is on. |

### `md.numen.readers.pdf`

The settings of the plugin PDF Reader.

| | | |
| --- | --- | --- |
| `defaultZoom` | text | zoom a PDF opens at. One of `page`, `width`, `100%`. |

### `md.numen.tools.media.importer`

The settings of the plugin Media Importer.

| | | |
| --- | --- | --- |
| `ytDlp` | text | path to the yt-dlp program. |
| `ffmpeg` | text | path to the ffmpeg program. |
| `captions` | text | preferred caption languages, comma-separated. |
| `automaticCaptions` | yes or no | use auto-generated captions when a site has no others. |
| `downloadCaptionsAutomatically` | yes or no | download captions or the article of link notes without them. |
| `copyMaxSizeMb` | a number | largest video to download, in MB; 0 for no limit. From 0 to 100000. |

### `md.numen.tools.proofreading`

The settings of the plugin LLM Proofreading.

| | | |
| --- | --- | --- |
| `connections` | text | services and agents, tried in order until one answers. |
| `batchSize` | a number | lines per model request. From 1 to 500. |
| `overlap` | a number | lines shared by neighboring batches. From 0 to 100. |
| `inFlight` | a number | parallel model requests per call. From 1 to 32. |
| `maxEditDistance` | a number | largest edit distance of a correction from its line. From 0 to 1. |

### `md.numen.tools.updater`

The settings of the plugin Update Notifier.

| | | |
| --- | --- | --- |
| `channel` | text | which builds are announced. One of `stable`, `beta`, `custom`. |
| `endpoint` | text | required. Address of latest.json for the Custom channel. |
| `checkIntervalHours` | a number | hours between two update checks. From 1 to 168. |

### `md.numen.views.agent.chat`

The settings of the plugin Agent Chat.

| | | |
| --- | --- | --- |
| `connections` | text | agents. |

### `md.numen.views.file.tree`

The settings of the plugin File Tree Explorer.

| | | |
| --- | --- | --- |
| `sortBy` | text | how files and folders are ordered in the tree. One of `name`, `modified`. |

### `md.numen.views.flashcards.studio`

The settings of the plugin Flashcards Studio.

| | | |
| --- | --- | --- |
| `dayStartTime` | text | the time a review day starts, as HH:MM up to 12:00. Any other value uses 04:00. |
| `agentConnections` | text | agents. |

### `md.numen.views.plex.graph`

The settings of the plugin Plex Graph.

| | | |
| --- | --- | --- |
| `references` | text | show the notes linked in the text: the ones this note links to, the ones that link to it, both, or none. One of `off`, `outgoing`, `incoming`, `both`. |
| `nestHeadings` | yes or no | show the headings of a note under its node. |
| `headingLimit` | a number | most headings shown under a node. From 1 to 12. |
| `showNames` | yes or no | under each related note, the names of the notes related to it. |
| `namesPerNote` | a number | most names shown under and over a related note. From 1 to 12. |
| `namesOpacity` | a number | how visible the names under and over a note are while the note is not in focus. 0 hides them. From 0 to 1. |
| `focusedNamesOpacity` | a number | how visible the names under and over a note are while the pointer is over the note or one of them. From 0.05 to 1. |
| `showPills` | yes or no | draw a pill behind each title. Off: the title alone. |
| `pillPaddingSides` | a number | space beside the title inside its pill. From 0 to 24. |
| `pillPaddingEnds` | a number | space above and below the title inside its pill. From 0 to 16. |
| `focusSize` | a number | how much larger the note in the middle is drawn: its pill, title and icon. From 1 to 2.5. |
| `lineThickness` | a number | width of the lines between notes, in pixels. From 0.5 to 4. |
| `outlineThickness` | a number | width of the outline of a pill, in pixels. From 0.5 to 4. |
| `lineCurvature` | a number | how far lines bend; 0 is straight. From 0 to 1. |
| `highlightLinks` | yes or no | highlight every line that joins a note while the pointer is over it. Off: only a line the pointer is over is highlighted. |
| `animationLength` | a number | time to move to another note, in ms; 0 is instant. From 0 to 2000. |
| `noteRoom` | a number | empty space around every note. From 0 to 40. |

### `md.numen.views.titleBar`

The settings of the plugin Title bar.

| | | |
| --- | --- | --- |
| `buttons` | text | where the bar draws the window buttons. macOS always draws them on the left. One of `mac`, `windows`, `none`. |
| `hideInFullscreen` | yes or no | hide the bar while the window is full screen. |
<!-- END AUTOGEN -->
