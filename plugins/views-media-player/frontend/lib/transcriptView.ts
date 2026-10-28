/**
 * The transcript as plain text in a CodeMirror view: lines, undo history, `Mod-s` to save, and
 * the extensions the caller adds, such as the time gutter. Transcript text is not Markdown, so the
 * view draws no syntax.
 */
import { Compartment, EditorState, type Extension } from '@codemirror/state'
import { EditorView, drawSelection, keymap } from '@codemirror/view'
import { defaultKeymap, history, historyKeymap } from '@codemirror/commands'

export interface TranscriptViewOptions {
  readonly parent: HTMLElement
  readonly text: string
  readonly isEditable: boolean
  readonly label: string
  readonly extensions: Extension
  /** Called with the whole text after each edit by the person. */
  readonly onChange: (text: string) => void
  readonly onSave: () => void
}

export interface TranscriptView {
  setText(text: string): void
  setEditable(isEditable: boolean): void
  setExtensions(extensions: Extension): void
  destroy(): void
}

/* `--editor-measure`, `--editor-lead` and `--editor-margin` are set by the element the view is
   drawn in. */
const theme = EditorView.theme({
  '&': {
    color: 'var(--numen-ink)',
    backgroundColor: 'transparent',
    fontFamily: 'var(--numen-font-sans)',
    fontSize: 'var(--numen-reading-size)',
    lineHeight: '1.6',
    height: '100%',
  },
  '&.cm-focused': { outline: 'none' },
  '.cm-scroller': {
    fontFamily: 'inherit',
    lineHeight: 'inherit',
    justifyContent: 'center',
    paddingInline: 'var(--editor-margin, 0)',
  },
  '.cm-content': {
    caretColor: 'var(--numen-ink)',
    paddingTop: 'var(--editor-lead, 0.5rem)',
    paddingBottom: 'var(--numen-gutter)',
    maxWidth: 'var(--editor-measure, none)',
  },
  '.cm-gutters': { maxWidth: 'var(--editor-measure, none)' },
  '.cm-cursor, .cm-dropCursor': { borderLeftColor: 'var(--numen-ink)' },
  '.cm-line': { padding: '0 var(--numen-gutter)' },
  '&.cm-focused > .cm-scroller > .cm-selectionLayer .cm-selectionBackground': {
    backgroundColor: 'var(--numen-selection)',
  },
  '&:not(.cm-focused) > .cm-scroller > .cm-selectionLayer .cm-selectionBackground': {
    backgroundColor: 'var(--numen-selection-away)',
  },
  '.cm-selectionBackground, .cm-content ::selection': {
    backgroundColor: 'var(--numen-selection)',
  },
})

const createEditable = (isEditable: boolean): Extension => [
  EditorView.editable.of(isEditable),
  EditorState.readOnly.of(!isEditable),
]

export function createTranscriptView(options: TranscriptViewOptions): TranscriptView {
  const editable = new Compartment()
  const added = new Compartment()
  // Text set from outside is not reported back as an edit.
  let isSetting = false

  const rootNode = options.parent.getRootNode()
  const root =
    rootNode instanceof Document ||
    (typeof ShadowRoot !== 'undefined' && rootNode instanceof ShadowRoot)
      ? rootNode
      : document

  const view = new EditorView({
    parent: options.parent,
    root,
    state: EditorState.create({
      doc: options.text,
      extensions: [
        history(),
        drawSelection(),
        EditorView.lineWrapping,
        keymap.of([
          {
            key: 'Mod-s',
            run: () => {
              options.onSave()
              return true
            },
          },
          ...defaultKeymap,
          ...historyKeymap,
        ]),
        theme,
        EditorView.contentAttributes.of({ 'aria-label': options.label }),
        editable.of(createEditable(options.isEditable)),
        added.of(options.extensions),
        EditorView.updateListener.of((update) => {
          if (update.docChanged && !isSetting) options.onChange(update.state.doc.toString())
        }),
      ],
    }),
  })

  const setText = (text: string) => {
    if (view.state.doc.toString() === text) return
    const anchor = Math.min(view.state.selection.main.anchor, text.length)
    isSetting = true
    try {
      view.dispatch({
        changes: { from: 0, to: view.state.doc.length, insert: text },
        selection: { anchor },
      })
    } finally {
      isSetting = false
    }
  }

  return {
    setText,
    setEditable: (isEditable) =>
      view.dispatch({ effects: editable.reconfigure(createEditable(isEditable)) }),
    setExtensions: (extensions) => view.dispatch({ effects: added.reconfigure(extensions) }),
    destroy: () => view.destroy(),
  }
}
