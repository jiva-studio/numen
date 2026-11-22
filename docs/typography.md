# Typography

The window has one type scale and three roles that draw from it. This page names the tokens, says which role each kind of text takes, and lists where the code still departs from that. It answers issue #2500.

## What exists

| Token                                    | Value                            | Follows                              |
| ---------------------------------------- | -------------------------------- | ------------------------------------ |
| `--numen-interface-scale`                | `1`                              | the multiplier of the root font size |
| `--numen-text-scale`                     | `1`                              | the multiplier of reading text only  |
| `--numen-text-1`                         | `0.625rem`                       | interface scale                      |
| `--numen-text-2`                         | `0.8125rem`                      | interface scale                      |
| `--numen-text-3`                         | `1.125rem`                       | interface scale                      |
| `--numen-text-4`                         | `1.375rem`                       | interface scale                      |
| `--numen-font-size`                      | `--numen-text-2`                 | interface scale                      |
| `--numen-title-size`                     | `--numen-text-3`                 | interface scale                      |
| `--numen-display-size`                   | `--numen-text-4`                 | interface scale                      |
| `--numen-reading-size`                   | `0.8125rem` times the text scale | interface scale, then text scale     |
| `--numen-prose-size`                     | `0.875rem` times the text scale  | interface scale, then text scale     |
| `--numen-editor-font-size`               | `14px`                           | neither; the Editor settings         |
| `--numen-line-height`                    | `1.35`                           | unitless                             |
| `--numen-font-sans`, `--numen-font-mono` | the two faces                    |                                      |

The Tailwind names `text-base` and `text-small` are `--numen-font-size` and `--numen-text-1`, both with `--numen-line-height`.

## Roles

| Role    | What it is                                                        | Size                              | Face                |
| ------- | ----------------------------------------------------------------- | --------------------------------- | ------------------- |
| Control | a button, a field's own label, a pill, a chip, a menu row, a tab  | `--numen-font-size` (`text-base`) | sans                |
| Caption | a hint, a count, a status line said quietly beside something else | `--numen-text-1` (`text-small`)   | sans                |
| Title   | the name of a screen, a dialog or a list                          | `--numen-title-size`              | sans                |
| Display | the one line a screen is built around                             | `--numen-display-size`            | sans                |
| Reading | what a person reads in a book or a document                       | `--numen-reading-size`            | as the content sets |
| Prose   | marked-up text, an answer, a card, the message being composed     | `--numen-prose-size`              | sans                |
| Code    | an identifier, a path, a command                                  | the size of its surroundings      | mono                |
| Editor  | the text of a note                                                | `--numen-editor-font-size`        | the Editor settings |

Reading and prose are the only roles that answer to `--numen-text-scale`; everything else answers to the interface scale alone. A control beside a piece of prose therefore stays the same size when the text multiplier is raised, and the two are told apart by role and not by accident. Different roles differ in size on purpose. Two things in the same role have one size.

Weight: 400 for text, 500 for a control label, 600 for a title or a heading, 700 only for a figure. A line height is the token; a bare number is allowed only for a single line of a fixed-height control (`1`).

## What is guarded

Stylelint already rejects `px` outside a 1px border and any non-`rem`/`em` length in the libraries, the desktop windows and the plugin frontends (`numen/no-bare-length`). It does not look at the number: `font-size: 0.9375rem` passes. A guard against raw `font-size` lengths would be a small stylelint rule beside it; it is left as a follow-up because the 37 stylesheets of the marketing sites and the manual would need an override first.

## Audit

Counted at the base of this branch over `*.css`, `*.vue` and `*.ts` outside tests and `tokens/`.

| Place                    | Raw `font-size` or `*-font-size`/`*-text` length        | State                                |
| ------------------------ | ------------------------------------------------------- | ------------------------------------ |
| `libs/ui`, `libs/plugin` | 0 in CSS                                                | clean                                |
| `apps/desktop`           | 0                                                       | clean                                |
| `plugins/*/frontend`     | 20 declarations in 8 plugins before this change; 1 left | see below                            |
| `apps/landing`           | 37                                                      | follow-up (separate site, own scale) |
| `apps/docs` (the manual) | 7                                                       | follow-up (separate site, own scale) |
| `tests/acceptance`       | 4                                                       | not interface                        |

The tokens are used 196 times in 104 files; `text-base` and `text-small` are in 31 and 32 files.

### The composer (the issue's example)

The typed message, its hidden mirror and its placeholder inherit `--numen-prose-size` from `.composer__grow`, which follows the text multiplier. The model and reasoning pills were `text-small`, `--numen-text-1`, which is a caption size. A pill is a control, and the pills are the row under the input, so the input was 14px times the text scale beside 10px. The pills are now `text-base`, the control size. The input stays prose; the two roles remain distinct and the input follows the text scale while the pills do not.

### Fixed in this change

| File                                                                                          | Was                                                                | Now                                       |
| --------------------------------------------------------------------------------------------- | ------------------------------------------------------------------ | ----------------------------------------- |
| `plugins/views-agent-chat/.../AgentModelSelector.vue`, `ReasoningSelector.vue`                | `text-small`                                                       | `text-base`                               |
| `plugins/views-media-viewer/.../MediaRecordingActions.vue`, `MediaRecordingPreview.vue`       | buttons 0.9375rem, save 0.8125rem, time 0.8125rem, title 0.9375rem | `--numen-font-size`                       |
| `plugins/views-media-viewer/.../MediaRecordingTab.vue`                                        | title 1.25rem, status 0.75rem, error 0.875rem                      | `--numen-title-size`, `--numen-font-size` |
| `plugins/views-plex-related-names/.../RelatedNames.vue`                                       | 0.7rem                                                             | `--numen-text-1`                          |
| `plugins/tools-index-stats`, `tools-plugin-stats`, `tools-sync`, `tools-updater` `styles.css` | chip and close 0.75rem                                             | `--numen-chrome-text-2` (see the chrome)  |
| `plugins/readers-pdf/.../DocumentTab.vue`                                                     | message 0.875rem                                                   | `--numen-font-size`                       |
| `plugins/notes/.../MarkdownNoteEditor.vue`                                                    | dismiss glyph 1.125rem                                             | `--numen-title-size`                      |

The title-bar chips moved from 12px to 13px and the pills from 10px to 13px; these are the visible changes. The sizes were picked to the nearest step of the scale.

### Follow-ups

| Priority | Item                                                                                                                                                    |
| -------- | ------------------------------------------------------------------------------------------------------------------------------------------------------- |
| 1        | `plugins/tools-saved-search/frontend/styles.css:163` 0.875rem; another branch moves it to `--numen-reading-size`.                                       |
| 1        | `MediaRecordingMeter.vue` timer 2.5rem: no step above `--numen-display-size`; decide whether a timer is a display.                                      |
| 2        | `text-xs` in `libs/ui/components/.../MenuRow.vue` and `libs/plugin/loader/.../pluginView.ts`: the Tailwind default, not a token. Use `text-small`.      |
| 2        | A stylelint rule that rejects a `rem` literal on `font-size` and on a `--*-font-size`/`--*-text` custom property outside `tokens/`.                     |
| 2        | 60 `line-height` literals with 9 distinct values (1 to 1.6) against `--numen-line-height` (1.35). Roles: single-line control 1, running text the token. |
| 3        | 66 `font-weight` declarations: 400 ×3, 500 ×27, 600 ×34, 700 ×2. Settle on the weights above and move the 600/500 mix.                                  |
| 3        | `apps/landing` (37) and `apps/docs` (7): give each site its own scale tokens.                                                                           |
| 3        | `--numen-reading-size` (0.8125rem) and `--numen-prose-size` (0.875rem) differ by a shade; decide whether reading and prose are one role.                |

## The window chrome

The title bar, the status bar and every widget a plugin draws in one of their slots (`titleBar.*`, `statusBar.*`), with the popovers those widgets open, keep one size at every interface scale and every text scale, like the native window buttons beside them. They are set in the chrome tokens, which hold `px`, and never in `rem` or a token that follows the root font size.

| Token                         | Value  |
| ----------------------------- | ------ |
| `--numen-chrome-unit`         | `16px` |
| `--numen-chrome-text-1`       | `10px` |
| `--numen-chrome-text-2`       | `13px` |
| `--numen-chrome-inset`        | `8px`  |
| `--numen-chrome-radius`       | `6px`  |
| `--numen-chrome-radius-tight` | `3px`  |

A length is a multiple of the unit: `calc(1.25 * var(--numen-chrome-unit))`. The stylelint rule `numen/chrome-fixed-size` rejects `rem`, `--numen-interface-scale`, `--numen-text-scale` and every token whose value follows them in the frontends of `views-title-bar`, `views-status-bar` and each plugin whose manifest draws a view in a chrome slot. The shared `Tooltip` and `Menu` of `@numen/ui` are not chrome and follow the interface scale.

## Acceptance

- Where components diverge from the roles and the scale: the audit above.
- Reading and interface stay distinct: reading and prose carry the text multiplier on top of the interface scale; control, caption and title carry the interface scale alone; the window chrome carries neither.
- Control text is covered: the chat input is prose, the pills are controls, and each is a named role.
- Not every element is one size: eight roles, five sizes.
