/**
 * The code editor of the query in an element: the search language highlighted, the names of the catalog
 * completed, the problems underlined, and Mod-Enter to save. It is made when the component mounts and
 * destroyed when it unmounts.
 */
import { onBeforeUnmount, onMounted, watch, type Ref } from 'vue'
import { autocompletion, completionKeymap } from '@codemirror/autocomplete'
import { defaultKeymap, history, historyKeymap } from '@codemirror/commands'
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

/** Where the editor is drawn, what it starts with, and what it reads while it is open. */
export interface QueryEditorOptions {
  readonly host: Readonly<Ref<HTMLElement | null>>
  readonly query: string
  readonly getProblems: () => readonly QueryProblem[]
  readonly getFunctions: () => readonly QueryFunction[]
  readonly onSave: () => void
}

/** The editor of the query: its text now ('' before it is drawn), and focus to put the cursor in it. */
export interface QueryEditor {
  getQuery(): string
  focus(): void
}

/** The editor of the query, drawn into the host when the component mounts. */
export function useQueryEditor(options: QueryEditorOptions): QueryEditor {
  let view: EditorView | null = null

  const saveKey = {
    key: 'Mod-Enter',
    run: () => {
      options.onSave()
      return true
    },
  }

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
        autocompletion({ override: [createQueryCompletion(options.getFunctions)] }),
        keymap.of([saveKey, ...completionKeymap, ...historyKeymap, ...defaultKeymap]),
        EditorView.lineWrapping,
        EditorView.contentAttributes.of({ 'aria-label': words.queryField }),
      ],
    })
  })

  onBeforeUnmount(() => {
    view?.destroy()
    view = null
  })

  watch(options.getProblems, (problems) => {
    if (!view) return
    view.dispatch(setDiagnostics(view.state, getQueryDiagnostics(view.state.doc, problems)))
  })

  return {
    getQuery: () => view?.state.doc.toString() ?? '',
    focus: () => view?.focus(),
  }
}
