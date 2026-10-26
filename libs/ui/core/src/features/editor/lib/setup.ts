/**
 * Everything the editor is, as one extension.
 *
 * Live preview is a facet of the configuration, so it can be turned off
 * without the editor being built again; with it off what is left is markdown
 * with its syntax coloured.
 */
import { history, historyKeymap, defaultKeymap, indentLess, indentMore } from '@codemirror/commands'
import { markdown } from '@codemirror/lang-markdown'
import { bracketMatching, syntaxHighlighting } from '@codemirror/language'
import {
  type Extension,
  Compartment,
  EditorState,
  StateEffect,
  StateField,
} from '@codemirror/state'
import {
  type KeyBinding,
  EditorView,
  drawSelection,
  keymap,
  placeholder,
  rectangularSelection,
} from '@codemirror/view'
import { follow } from './extensions/address'
import { pendingChange, changeMarks, createPacePlugin, type EditorChange } from './change/change'
import { highlighting } from '../config/highlight'
import { LANGUAGES } from '../config/languages'
import { live, wholeLines } from './live/live'
import { GFM } from '@lezer/markdown'
import { saveHandler } from './outside'
import { monospaced, theme } from '../config/theme'

/** What can be changed without the editor being built again. */
export const previewCompartment = new Compartment()
export const editableCompartment = new Compartment()
export const changeCompartment = new Compartment()
export const extensionsCompartment = new Compartment()
export const languageCompartment = new Compartment()

export interface Settings {
  /** Marks are drawn as what they mean, away from the caret. */
  readonly isLivePreview?: boolean | undefined
  readonly readonly?: boolean | undefined
  readonly placeholder?: string | undefined
  /**
   * What the editor is announced as. A box typed into is not named by what has
   * been typed in it, so the placeholder is no name and this is the only one.
   */
  readonly name?: string | undefined
  /** What the editor is described by: where the words are written is the caller's. */
  readonly describedBy?: string | undefined
  /** A change being made to the text by something other than the reader. */
  readonly change?: EditorChange | null | undefined
  /** What whoever put the editor on the screen draws into it. */
  readonly extensions?: Extension | undefined
}

export const preview = (isOn: boolean): Extension => (isOn ? [wholeLines, live, follow] : [])

export const showChange = (change: EditorChange | null): Extension => pendingChange.of(change)

export const createEditable = (isOn: boolean): Extension => [
  EditorView.editable.of(isOn),
  EditorState.readOnly.of(!isOn),
]

/**
 * The chord that asks for the text to be kept now, answered by whoever put the
 * editor on the screen. It is handled here, so the letter is not inserted and
 * the page's own answer to the chord does not run.
 */
export const saveKeyBinding: KeyBinding = {
  key: 'Mod-s',
  run: (view) => {
    view.state.facet(saveHandler)()
    return true
  },
}

/**
 * Whether Tab belongs to the editor or to the page. It begins with the editor,
 * Escape hands it to the page, and the keyboard arriving takes it back, so
 * every visit begins the same way.
 */
const setTabCapture = StateEffect.define<boolean>()

const tabCapture = StateField.define<boolean>({
  create: () => true,
  update: (target, change) =>
    change.effects.reduce((now, effect) => (effect.is(setTabCapture) ? effect.value : now), target),
})

const restoreTabCapture = EditorView.domEventHandlers({
  focus: (_press, view) => {
    if (!view.state.field(tabCapture)) view.dispatch({ effects: setTabCapture.of(true) })
    return false
  },
})

/** Tab indents while the editor holds it, and walks on once it does not. */
export const tabKeyBinding: KeyBinding = {
  key: 'Tab',
  run: (view) => view.state.field(tabCapture) && indentMore(view),
  shift: (view) => view.state.field(tabCapture) && indentLess(view),
}

/**
 * Escape hands Tab to the page and is passed on, so what else answers Escape
 * around the editor still answers it.
 */
export const escapeKeyBinding: KeyBinding = {
  key: 'Escape',
  run: (view) => {
    if (view.state.field(tabCapture)) view.dispatch({ effects: setTabCapture.of(false) })
    return false
  },
}

/** Markdown, which is what a document naming no language is written in. */
export const prose = (): Extension => markdown({ extensions: GFM, codeLanguages: LANGUAGES })

/** One whole document of code: how it reads, and the face it is set in. */
export const code = (support: Extension): Extension => [support, monospaced]

export const setup = (settings: Settings = {}): Extension => [
  history(),
  drawSelection(),
  rectangularSelection(),
  bracketMatching(),
  EditorView.lineWrapping,
  tabCapture,
  restoreTabCapture,
  keymap.of([saveKeyBinding, escapeKeyBinding, ...defaultKeymap, ...historyKeymap, tabKeyBinding]),
  languageCompartment.of(prose()),
  syntaxHighlighting(highlighting),
  theme,
  placeholder(settings.placeholder ?? ''),
  EditorView.contentAttributes.of({
    'aria-label': settings.name ?? 'Editor',
    ...(settings.describedBy ? { 'aria-describedby': settings.describedBy } : {}),
  }),
  changeMarks,
  createPacePlugin(),
  previewCompartment.of(preview(settings.isLivePreview ?? true)),
  editableCompartment.of(createEditable(!settings.readonly)),
  changeCompartment.of(showChange(settings.change ?? null)),
  extensionsCompartment.of(settings.extensions ?? []),
]
