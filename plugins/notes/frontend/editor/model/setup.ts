/**
 * Everything the editor is, as one extension.
 *
 * Live preview is a facet of the configuration, so it can be turned off
 * without the editor being built again; with it off what is left is markdown
 * with its syntax coloured.
 */
import { browserClock, doc } from '@numen/ui'
import { history, historyKeymap, defaultKeymap, indentLess, indentMore } from '@codemirror/commands'
import { markdown } from '@codemirror/lang-markdown'
import { bracketMatching, indentUnit, syntaxHighlighting, syntaxTree } from '@codemirror/language'
import {
  type Extension,
  type Range,
  Compartment,
  EditorState,
  RangeSet,
  StateEffect,
  StateField,
} from '@codemirror/state'
import {
  type KeyBinding,
  EditorView,
  GutterMarker,
  drawSelection,
  gutterLineClass,
  highlightActiveLine,
  keymap,
  lineNumberWidgetMarker,
  lineNumbers,
  placeholder,
  rectangularSelection,
} from '@codemirror/view'
import { follow } from '../lib/extensions/address'
import {
  pendingChange,
  changeMarks,
  createPacePlugin,
  type EditorChange,
} from '../lib/change/change'
import { highlighting } from '../lib/config/highlight'
import { LANGUAGES } from '../lib/config/languages'
import { live, sourceWikilinks, wholeLines } from './live/live'
import { renderBracketMatch } from '../lib/live/wikilinks'
import { HEADING } from '../lib/live/styles'
import { GFM } from '@lezer/markdown'
import { saveHandler } from '../lib/outside'
import { getFontFamilyValue } from '../../model/fontFamily'
import { monospaced, theme } from '../lib/config/theme'

/** What can be changed without the editor being built again. */
export const previewCompartment = new Compartment()
export const editableCompartment = new Compartment()
export const changeCompartment = new Compartment()
export const extensionsCompartment = new Compartment()
export const languageCompartment = new Compartment()
export const wrapCompartment = new Compartment()
export const lineNumbersCompartment = new Compartment()
export const tabSizeCompartment = new Compartment()
export const fontCompartment = new Compartment()
export const lineHeightCompartment = new Compartment()
export const activeLineCompartment = new Compartment()

export interface Settings {
  readonly isLivePreview?: boolean | undefined
  readonly readonly?: boolean | undefined
  readonly placeholder?: string | undefined
  readonly name?: string | undefined
  readonly describedBy?: string | undefined
  readonly change?: EditorChange | null | undefined
  readonly extensions?: Extension | undefined
  readonly softWrap?: boolean | undefined
  readonly lineNumbers?: boolean | undefined
  readonly tabSize?: number | undefined
  readonly fontFamily?: string | undefined
  readonly lineHeight?: number | undefined
  readonly highlightActiveLine?: boolean | undefined
}

export const preview = (isOn: boolean): Extension =>
  isOn ? [wholeLines, live, follow] : [sourceWikilinks]

export const showChange = (change: EditorChange | null): Extension => pendingChange.of(change)

export const createEditableExtension = (isOn: boolean): Extension => [
  EditorView.editable.of(isOn),
  EditorState.readOnly.of(!isOn),
]

export const getWrapExtension = (softWrap: boolean): Extension =>
  softWrap ? EditorView.lineWrapping : []

class BlockLineNumberMarker extends GutterMarker {
  constructor(readonly label: string) {
    super()
  }

  override eq(other: BlockLineNumberMarker): boolean {
    return this.label === other.label
  }

  override toDOM(): Node {
    return doc.createTextNode(this.label)
  }
}

class LineClassMarker extends GutterMarker {
  constructor(override readonly elementClass: string) {
    super()
  }

  override eq(other: LineClassMarker): boolean {
    return this.elementClass === other.elementClass
  }
}

function buildGutterLineClasses(state: EditorState): RangeSet<GutterMarker> {
  const markers: Range<GutterMarker>[] = []
  syntaxTree(state).iterate({
    enter: (node) => {
      const heading = HEADING.exec(node.name)
      if (heading) {
        const line = state.doc.lineAt(node.from)
        const gap = line.from > 0 ? ' cm-gutter-gap' : ''
        markers.push(
          new LineClassMarker(`cm-gutter-heading cm-gutter-heading-${heading[1]}${gap}`).range(
            line.from,
          ),
        )
        return false
      }
      if (node.name === 'FencedCode') {
        const line = state.doc.lineAt(node.from)
        markers.push(new LineClassMarker('cm-gutter-code-first').range(line.from))
        return false
      }
      return true
    },
  })
  return RangeSet.of(markers, true)
}

const gutterLineClasses = StateField.define<RangeSet<GutterMarker>>({
  create: buildGutterLineClasses,
  update: (current, tr) =>
    tr.docChanged || syntaxTree(tr.startState) !== syntaxTree(tr.state)
      ? buildGutterLineClasses(tr.state)
      : current,
  provide: (field) => gutterLineClass.from(field),
})

export const getLineNumbersExtension = (show: boolean): Extension =>
  show
    ? [
        lineNumbers(),
        gutterLineClasses,
        lineNumberWidgetMarker.of(
          (view, _widget, block) =>
            new BlockLineNumberMarker(String(view.state.doc.lineAt(block.from).number)),
        ),
      ]
    : []

export const getTabSizeExtension = (tabSize: number): Extension => {
  const size = Math.max(1, Math.trunc(tabSize))
  return [EditorState.tabSize.of(size), indentUnit.of(' '.repeat(size))]
}

/** The font of the text, set as a variable the theme reads; no variable where the family is empty or invalid. */
export const getFontExtension = (family: string): Extension => {
  const value = getFontFamilyValue(family)
  return value ? EditorView.editorAttributes.of({ style: `--editor-font: ${value}` }) : []
}

export const getLineHeightExtension = (height: number): Extension =>
  EditorView.editorAttributes.of({ style: `--editor-line-height: ${height}` })

export const getActiveLineExtension = (isOn: boolean): Extension =>
  isOn ? highlightActiveLine() : []

/**
 * The chord that requests an immediate save, handled by the code that hosts the
 * editor. It is handled here, so the letter is not inserted and
 * the page's own handling of the chord does not run.
 */
const saveKeyBinding: KeyBinding = {
  key: 'Mod-s',
  run: (view) => {
    view.state.facet(saveHandler)()
    return true
  },
}

/**
 * Whether Tab belongs to the editor or to the page. It starts with the editor,
 * Escape releases it to the page, and keyboard input into the editor restores it, so
 * every visit starts the same way.
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

const getLayout = (settings: Settings): Extension[] => [
  wrapCompartment.of(getWrapExtension(settings.softWrap ?? true)),
  lineNumbersCompartment.of(getLineNumbersExtension(settings.lineNumbers ?? false)),
  tabSizeCompartment.of(getTabSizeExtension(settings.tabSize ?? 2)),
  fontCompartment.of(getFontExtension(settings.fontFamily ?? '')),
  lineHeightCompartment.of(getLineHeightExtension(settings.lineHeight ?? 1.6)),
  activeLineCompartment.of(getActiveLineExtension(settings.highlightActiveLine ?? false)),
]

const getAccessibility = (settings: Settings): Extension =>
  EditorView.contentAttributes.of({
    'aria-label': settings.name ?? 'Editor',
    ...(settings.describedBy ? { 'aria-describedby': settings.describedBy } : {}),
  })

export const setup = (settings: Settings = {}): Extension => [
  history(),
  drawSelection(),
  rectangularSelection(),
  bracketMatching({ renderMatch: renderBracketMatch }),
  ...getLayout(settings),
  tabCapture,
  restoreTabCapture,
  keymap.of([saveKeyBinding, escapeKeyBinding, ...defaultKeymap, ...historyKeymap, tabKeyBinding]),
  languageCompartment.of(prose()),
  syntaxHighlighting(highlighting),
  theme,
  placeholder(settings.placeholder ?? ''),
  getAccessibility(settings),
  changeMarks,
  createPacePlugin(browserClock),
  previewCompartment.of(preview(settings.isLivePreview ?? true)),
  editableCompartment.of(createEditableExtension(!settings.readonly)),
  changeCompartment.of(showChange(settings.change ?? null)),
  extensionsCompartment.of(settings.extensions ?? []),
]
