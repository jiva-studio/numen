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

### `app.agent`

Selects the agent the panel uses and configures how each program runs. A program's section is kept whether or not it is selected, so switching agent preserves the settings of the previous one. The programs are the ones listed by the plugins that provide agents.

| | | |
| --- | --- | --- |
| `use` | text | names the program the panel uses. When absent, the program that the providers mark as default is used. |

### `app.naming`

| | | |
| --- | --- | --- |
| `syncTitleAndFilename` | yes or no | renaming one also renames the other. |

### `app.performance`

| | | |
| --- | --- | --- |
| `profile` | text | how much of the hardware indexing and models may use. One of `eco`, `balanced`, `maximum`. |

### `app.ai.default`

The agent and the model that handle the questions of a plugin whose own setting names no agent, and how many questions one agent handles at once.

| | | |
| --- | --- | --- |
| `agent` | text | the id of the agent plugin to use. Empty selects the provider with the lowest id. |
| `model` | text | the model the agent uses. Empty leaves the choice to the agent. |
| `concurrency` | a number | how many questions one agent handles at once. |

### `md.numen.agents.acp`

The settings of the plugin ACP Agents.

| | | |
| --- | --- | --- |
| `executablePath` | text | installed ACP agents are discovered automatically. Choose one or more custom ACP agent executables to add them. |

### `md.numen.agents.antigravity`

The settings of the plugin Antigravity Agent.

| | | |
| --- | --- | --- |
| `executablePath` | text | path to the Antigravity CLI executable. Empty finds it automatically. |

### `md.numen.agents.claude`

The settings of the plugin Claude Code Agent.

| | | |
| --- | --- | --- |
| `executablePath` | text | path to the Claude Code CLI executable. Empty finds it automatically. |
| `readsHooksAndSkills` | yes or no | let Claude Code read the hooks, skills, standing instructions in CLAUDE.md and plugins configured for this person. Off. A hook is a shell command Claude Code runs itself. What a vault contains is never read. |

### `md.numen.agents.codex`

The settings of the plugin OpenAI Codex Agent.

| | | |
| --- | --- | --- |
| `executablePath` | text | path to the OpenAI Codex CLI executable. Empty finds it automatically. |

### `md.numen.agents.mcp.server`

The settings of the plugin Model Context Protocol Server.

| | | |
| --- | --- | --- |
| `port` | a number | local TCP port on 127.0.0.1 for external MCP clients (Streamable HTTP at /mcp). From 1024 to 65535. |
| `token` | text | token the host generates on first load; an MCP client sends it in the header Authorization: Bearer <token>, and while it is empty the server rejects every request. |

### `md.numen.agents.numen`

The settings of the plugin Numen Agent.

| | | |
| --- | --- | --- |
| `baseUrl` | text | base URL of an OpenAI-compatible chat API, such as https://openrouter.ai/api/v1 or http://localhost:11434/v1. Available models are fetched from <Chat API>/models. |
| `model` | text | model used when a task sets none. |
| `temperature` | a number | sampling temperature for model responses (0.0 to 1.0). From 0 to 1. |
| `maxTokens` | a number | maximum number of tokens to generate per completion. From 256 to 128000. |
| `maxSteps` | a number | maximum number of requests to the model one task may make before it is stopped. From 1 to 1000. |

### `md.numen.engines.embeddings`

The settings of the plugin Vector Embeddings.

| | | |
| --- | --- | --- |
| `provider` | text | where texts are turned into vectors. One of `local`, `openrouter`, `openai`, `custom`. |
| `baseUrl` | text | base URL of the embeddings API, for example https://example.org/v1. |
| `model` | text | model identifier. A local model needs a profile: the listed ones ship with the plugin, and a profile file of your own in the profiles folder of the plugin data directory adds more (see the plugin README). Local models are downloaded from Hugging Face; remote providers use the model ID listed by the service. |
| `minChunkLength` | a number | pieces of text shorter than this many characters get no vector for the search by meaning, so page numbers and running heads do not crowd out real answers. They stay searchable by their words. A note shorter than this is found by its words too. 0 gives every piece a vector. From 0 to 1000. |
| `chunkTokens` | a number | a note is cut into chunks of at most this many tokens, and each chunk gets a vector. It never exceeds the input limit of the model. A smaller value gives more precise passages and more vectors. A change cuts the notes again; vectors for texts that are unchanged are kept. From 16 to 8192. |

### `md.numen.engines.ocr`

The settings of the plugin Document OCR.

| | | |
| --- | --- | --- |
| `model` | text | text recognition model. One of `pp-ocrv6-tiny`, `pp-ocrv6-small`, `pp-ocrv5-latin`, `pp-ocrv5-eslav`, `pp-ocrv5-cyrillic`, `pp-ocrv5-devanagari`, `pp-ocrv5-greek`, `custom`. |
| `customModelPath` | text | paddleOCR text recognition model in ONNX format. |
| `customDictPath` | text | UTF-8 text, one character per line. Empty for a model that includes its own. |
| `dpi` | text | resolution a page is rendered at before recognition. One of `150`, `200`, `300`, `400`, `600`. |
| `recognizeAutomatically` | yes or no | recognize the text of every PDF and image in the vault that has none. A text removed with Delete text is recognized again. |

### `md.numen.engines.transcription`

The settings of the plugin Audio/Video Transcription.

| | | |
| --- | --- | --- |
| `model` | text | speech recognition model downloaded from Hugging Face. Parakeet v3 supports 25 European languages with automatic language detection; Parakeet v2 is English-only. One of `parakeet-tdt-0.6b-v3-int8`, `parakeet-tdt-0.6b-v2-int8`. |
| `transcribeAutomatically` | yes or no | transcribe every recording in the vault that has no transcript. A transcript removed with Delete text is made again. |

### `md.numen.notes`

The settings of the plugin Notes.

| | | |
| --- | --- | --- |
| `softWrap` | yes or no | wrap long lines to fit the editor viewport. |
| `lineNumbers` | yes or no | show line numbers in the editor margin. |

### `md.numen.readers.pdf`

The settings of the plugin PDF Reader.

| | | |
| --- | --- | --- |
| `defaultZoom` | text | initial zoom level when opening a PDF document. One of `page`, `width`, `100%`. |

### `md.numen.tools.media.importer`

The settings of the plugin Media Importer.

| | | |
| --- | --- | --- |
| `ytDlp` | text | path to the yt-dlp program. |
| `ffmpeg` | text | path to the ffmpeg program. |
| `captions` | text | caption languages in order of preference, comma-separated. |
| `automaticCaptions` | yes or no | use auto-generated captions when a site has no others. |
| `downloadCaptionsAutomatically` | yes or no | download the captions or the article of every link note that has none. A text removed with Delete text is downloaded again. |
| `copyMaxSizeMb` | a number | maximum video download size in megabytes; 0 for no limit. From 0 to 100000. |

### `md.numen.tools.proofreading`

The settings of the plugin LLM Proofreading.

| | | |
| --- | --- | --- |
| `provider` | text | how proofreading requests are executed: through the default Numen agent or directly against an OpenAI-compatible batch/chat API. One of `agent`, `openrouter`, `openai`, `custom`. |
| `baseUrl` | text | base URL of the OpenAI-compatible API, for example http://localhost:11434/v1. |
| `model` | text | model ID used for batch proofreading requests. |
| `batchSize` | a number | lines per model request. From 1 to 500. |
| `overlap` | a number | lines shared by neighbouring batches. From 0 to 100. |
| `inFlight` | a number | parallel model requests per call. From 1 to 32. |
| `maxEditDistance` | a number | maximum edit distance of a correction from the original line. From 0 to 1. |

### `md.numen.tools.updater`

The settings of the plugin Update Notifier.

| | | |
| --- | --- | --- |
| `channel` | text | which builds are announced. One of `stable`, `beta`, `custom`. |
| `endpoint` | text | required. Address of the latest.json file to ask for the newest build of the Custom channel. The Stable and Beta channels ask their own published feeds. |
| `checkIntervalHours` | a number | hours between two checks for a newer build. From 1 to 168. |

### `md.numen.views.agent.chat`

The settings of the plugin Agent Chat.

| | | |
| --- | --- | --- |
| `reasoning` | text | the depth of reasoning a new question starts with, by the name of one of the levels the model lists. Empty uses the default of the model; a level the model does not list is ignored. |

### `md.numen.views.flashcards.studio`

The settings of the plugin Flashcards Studio.

| | | |
| --- | --- | --- |
| `dayStartTime` | text | time a review day starts. Answers before it count toward the day before. HH:MM from 00:00 to 12:00; other values use 04:00. |

### `md.numen.views.plex.graph`

The settings of the plugin Plex Graph.

| | | |
| --- | --- | --- |
| `nestHeadings` | yes or no | show the headings of a note under its node. |
| `headingLimit` | a number | maximum number of headings shown under a node. The rest scroll. From 1 to 12. |
| `showNames` | yes or no | under each related note, show the names of the notes it is related to. |
| `namesPerNote` | a number | most names shown under and over a related note. From 1 to 12. |
| `showPills` | yes or no | draw a pill behind each title. Off: the title alone, with a dot in the color of its relation. |
| `pillPaddingSides` | a number | space between a title and the left and right edge of its pill. From 0 to 24. |
| `pillPaddingEnds` | a number | space between a title and the top and bottom edge of its pill. From 0 to 16. |
| `focusSize` | a number | how much larger the pill of the note in the middle is. Its title keeps its size. From 1 to 2.5. |
| `animationLength` | a number | how long moving to another note takes, in milliseconds. 0 moves at once. From 0 to 2000. |
| `lineThickness` | a number | width of the lines between notes, in pixels. From 0.5 to 4. |
| `outlineThickness` | a number | width of the outline of a pill, in pixels. From 0.5 to 4. |
| `lineCurvature` | a number | how far a line between two notes bends. 0 is a straight line. From 0 to 1. |
| `noteRoom` | a number | extra empty space kept around every note. Larger is sparser. From 0 to 40. |
| `references` | text | show the notes linked in the text: the ones this note links to, the ones that link to it, both, or none. One of `off`, `outgoing`, `both`, `incoming`. |

### `md.numen.views.titleBar`

The settings of the plugin Title bar.

| | | |
| --- | --- | --- |
| `buttons` | text | where the bar draws the minimize, maximize and close buttons: at the right like Windows, at the left like macOS, or not at all. macOS always keeps the system buttons at the left. On Windows, the buttons at the left lose the Snap Layouts menu. One of `mac`, `windows`, `none`. |
| `hideInFullscreen` | yes or no | draw no bar while the window is in full screen, so the tabs start at the top of the window. The bar is drawn again when the window leaves full screen. |
<!-- END AUTOGEN -->
