/**
 * Everything the editor is, as one extension.
 *
 * Live preview is a facet of the configuration, so it can be turned off
 * without the editor being built again; with it off what is left is markdown
 * with its syntax coloured.
 */
import { history, historyKeymap, defaultKeymap, indentWithTab } from '@codemirror/commands'
import { markdown } from '@codemirror/lang-markdown'
import { bracketMatching, syntaxHighlighting } from '@codemirror/language'
import { type Extension, Compartment, EditorState } from '@codemirror/state'
import {
  type KeyBinding,
  EditorView,
  drawSelection,
  keymap,
  placeholder,
  rectangularSelection,
} from '@codemirror/view'
import { changing, marked, pacing, type EditorChange } from './change'
import { highlighting } from './highlight'
import { LANGUAGES } from './languages'
import { following, live, wholeLines } from './live'
import { GFM } from '@lezer/markdown'
import { saving } from './outside'
import { theme } from './theme'

/** What can be changed without the editor being built again. */
export const drawing = new Compartment()
export const editing = new Compartment()
export const showing = new Compartment()

export interface Settings {
  /** Marks are drawn as what they mean, away from the caret. */
  readonly live?: boolean
  readonly readonly?: boolean
  readonly placeholder?: string
  /** A change being made to the text by something other than the reader. */
  readonly change?: EditorChange | null
}

export const preview = (on: boolean): Extension => (on ? [wholeLines, live, following] : [])

export const shown = (change: EditorChange | null): Extension => changing.of(change)

export const editable = (on: boolean): Extension => [
  EditorView.editable.of(on),
  EditorState.readOnly.of(!on),
]

/**
 * The chord that asks for the text to be kept now, answered by whoever put the
 * editor on the screen. It is handled here, so the letter is not inserted and
 * the page's own answer to the chord does not run.
 */
export const keeping: KeyBinding = {
  key: 'Mod-s',
  run: (view) => {
    view.state.facet(saving)()
    return true
  },
}

export const setup = (settings: Settings = {}): Extension => [
  history(),
  drawSelection(),
  rectangularSelection(),
  bracketMatching(),
  EditorView.lineWrapping,
  keymap.of([keeping, ...defaultKeymap, ...historyKeymap, indentWithTab]),
  markdown({ extensions: GFM, codeLanguages: LANGUAGES }),
  syntaxHighlighting(highlighting),
  theme,
  placeholder(settings.placeholder ?? ''),
  marked,
  pacing(),
  drawing.of(preview(settings.live ?? true)),
  editing.of(editable(!settings.readonly)),
  showing.of(shown(settings.change ?? null)),
]
