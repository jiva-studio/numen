/**
 * The code editor of the query in an element: the search language highlighted, the names of the catalog
 * completed, the problems underlined with their message on hover, and the query sent once typing
 * pauses. It is made when the component mounts and destroyed when it unmounts; a query still waiting
 * then is sent at once.
 */
import { onBeforeUnmount, onMounted, watch, type Ref } from 'vue'
import { autocompletion, completionKeymap } from '@codemirror/autocomplete'
import { defaultKeymap, history, historyKeymap, indentLess, indentMore } from '@codemirror/commands'
import { getIndentUnit, indentString, syntaxHighlighting } from '@codemirror/language'
import type { StateCommand } from '@codemirror/state'
import { setDiagnostics } from '@codemirror/lint'
import { EditorView, highlightActiveLine, keymap, tooltips } from '@codemirror/view'
import { WORDS as words } from '../../words'
import {
  createQueryCompletion,
  getQueryDiagnostics,
  queryHighlighting,
  queryLanguage,
  type QueryFunction,
  type QueryProblem,
} from '../lib/queryLanguage'
import { formatQuery } from '../lib/formatQuery'
import { queryTheme } from '../lib/queryTheme'
import { createQueryOptionDetails } from './queryOption'
import { QUERY_PAUSE_MS } from './queryPause'

const insertIndent: StateCommand = ({ state, dispatch }) => {
  if (state.selection.ranges.some((range) => !range.empty)) return indentMore({ state, dispatch })
  const unit = indentString(state, getIndentUnit(state))
  dispatch(state.update(state.replaceSelection(unit), { scrollIntoView: true, userEvent: 'input' }))
  return true
}

const formatDocument: StateCommand = ({ state, dispatch }) => {
  const query = state.doc.toString()
  const formatted = formatQuery(query)
  if (formatted !== query) {
    dispatch(
      state.update({
        changes: { from: 0, to: state.doc.length, insert: formatted },
        scrollIntoView: true,
        userEvent: 'input.format',
      }),
    )
  }
  return true
}

/** Where the editor is drawn, what it starts with, what it reads while it is open, and where a query goes. */
export interface QueryEditorOptions {
  readonly host: Readonly<Ref<HTMLElement | null>>
  readonly query: string
  readonly getProblems: () => readonly QueryProblem[]
  readonly getFunctions: () => readonly QueryFunction[]
  readonly onChange: (query: string) => void
}

/**
 * The editor of the query: focus puts the cursor in it, moveTo puts it at a line and column counted from
 * 1, format lays the query out as one change that undo takes back.
 */
export interface QueryEditor {
  focus(): void
  moveTo(line: number, column: number): void
  format(): void
}

/** The editor of the query, drawn into the host when the component mounts. */
export function useQueryEditor(options: QueryEditorOptions): QueryEditor {
  let view: EditorView | null = null
  let pause: ReturnType<typeof setTimeout> | null = null

  const flush = (): void => {
    if (pause === null) return
    clearTimeout(pause)
    pause = null
    if (view) options.onChange(view.state.doc.toString())
  }

  const typingListener = EditorView.updateListener.of((update) => {
    if (!update.docChanged) return
    if (pause !== null) clearTimeout(pause)
    pause = setTimeout(flush, QUERY_PAUSE_MS)
  })

  onMounted(() => {
    const parent = options.host.value
    const optionDetails = createQueryOptionDetails()
    if (!parent) return
    view = new EditorView({
      parent,
      doc: options.query,
      extensions: [
        history(),
        highlightActiveLine(),
        queryLanguage,
        syntaxHighlighting(queryHighlighting),
        queryTheme,
        autocompletion({
          override: [createQueryCompletion(options.getFunctions)],
          addToOptions: [{ render: optionDetails.render, position: 90 }],
          icons: false,
        }),
        optionDetails.extension,
        tooltips({ parent: parent.ownerDocument.body }),
        keymap.of([
          ...completionKeymap,
          ...historyKeymap,
          ...defaultKeymap,
          { key: 'Tab', run: insertIndent, shift: indentLess },
          { key: 'Shift-Alt-f', run: formatDocument },
        ]),
        EditorView.lineWrapping,
        EditorView.contentAttributes.of({ 'aria-label': words.queryField }),
        typingListener,
      ],
    })
  })

  onBeforeUnmount(() => {
    flush()
    view?.destroy()
    view = null
  })

  const showProblems = (problems: readonly QueryProblem[]): void => {
    if (!view) return
    view.dispatch(setDiagnostics(view.state, getQueryDiagnostics(view.state.doc, problems)))
  }

  onMounted(() => showProblems(options.getProblems()))

  watch(options.getProblems, showProblems)

  const moveTo = (line: number, column: number): void => {
    if (!view) return
    const doc = view.state.doc
    const target = doc.line(Math.min(Math.max(line, 1), doc.lines))
    const anchor = Math.min(target.from + Math.max(column - 1, 0), target.to)
    view.dispatch({ selection: { anchor }, scrollIntoView: true })
    view.focus()
  }

  return {
    focus: () => view?.focus(),
    moveTo,
    format: () => {
      if (!view) return
      formatDocument(view)
      view.focus()
    },
  }
}
