---
title: Settings reference
description: Every key numen.json can hold, section by section, generated from the application itself.
---

Every key `numen.json` can hold. The pages under Settings say which of these are worth touching and why; this one lists every key.

The first level of the file is the version `v`, the owner `app` for the settings of the application, and one object for each plugin, named by the plugin's id. Each heading below is a section of `app`, and the keys under it are written the way they are written inside that section — `textScale` under `app.appearance` is `app.appearance.textScale` if you write it out in full.

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

Selects the agent the panel uses and configures how each program runs.  A program's section is kept whether or not the program is selected, so switching to another agent preserves the settings of the previous one. The programs are the ones listed by the plugins that provide agents.

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

### `app.window`

| | | |
| --- | --- | --- |
| `buttons` | text | where the window draws its minimise, maximise and close buttons on Linux: at the right like Windows, or at the left like macOS. One of `windows`, `mac`. |
<!-- END AUTOGEN -->
