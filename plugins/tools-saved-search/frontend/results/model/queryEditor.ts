/**
 * The code editor of the query in an element: the search language highlighted, the names of the catalog
 * completed, the problems underlined with their message on hover and under the cursor, and the query
 * sent once typing pauses. It is made when the component mounts and destroyed when it unmounts; a query
 * still waiting then is sent at once.
 */
import { onBeforeUnmount, onMounted, watch, type Ref } from 'vue'
import { autocompletion, completionKeymap } from '@codemirror/autocomplete'
import { defaultKeymap, history, historyKeymap, indentLess, insertTab } from '@codemirror/commands'
import { syntaxHighlighting } from '@codemirror/language'
import { setDiagnostics } from '@codemirror/lint'
import { EditorView, keymap } from '@codemirror/view'
import { WORDS as words } from '../../words'
import {
  createQueryCompletion,
  getQueryDiagnostics,
  queryHighlighting,
  queryLanguage,
  type QueryFunction,
  type QueryProblem,
} from '../lib/queryLanguage'
import { problemAtCursor } from './problemTooltip'
import { queryTheme } from '../lib/queryTheme'

/** How long typing pauses before the query is sent. */
export const QUERY_PAUSE_MS = 1000

/** Where the editor is drawn, what it starts with, what it reads while it is open, and where a query goes. */
export interface QueryEditorOptions {
  readonly host: Readonly<Ref<HTMLElement | null>>
  readonly query: string
  readonly getProblems: () => readonly QueryProblem[]
  readonly getFunctions: () => readonly QueryFunction[]
  readonly onChange: (query: string) => void
}

/** The editor of the query: focus puts the cursor in it. */
export interface QueryEditor {
  focus(): void
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
    if (!parent) return
    view = new EditorView({
      parent,
      doc: options.query,
      extensions: [
        history(),
        queryLanguage,
        syntaxHighlighting(queryHighlighting),
        queryTheme,
        autocompletion({ override: [createQueryCompletion(options.getFunctions)] }),
        keymap.of([
          ...completionKeymap,
          ...historyKeymap,
          ...defaultKeymap,
          { key: 'Tab', run: insertTab, shift: indentLess },
        ]),
        EditorView.lineWrapping,
        EditorView.contentAttributes.of({ 'aria-label': words.queryField }),
        typingListener,
        problemAtCursor,
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

  return {
    focus: () => view?.focus(),
  }
}
