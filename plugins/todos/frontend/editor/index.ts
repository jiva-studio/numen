import { autocompletion } from '@codemirror/autocomplete'
import { EditorState, type Extension } from '@codemirror/state'
import type { TodosApi } from '../api/wire'
import { WORDS } from '../words'
import type { ListFieldsResponse } from '../types'
import { createTaskCompletion } from './lib/taskCompletion'
import { completionTheme, renderCompletionIcon } from './ui/completionTheme'
import { createTaskExtension } from './ui/taskExtension'

/** The id the manifest declares the editor extension under. */
export const TASKS_EXTENSION = 'tasks'

export interface TaskEditorDeps {
  readonly api: TodosApi
  readonly getToday: () => string
  readonly getFields: () => ListFieldsResponse
  readonly loadFields: () => Promise<void>
  readonly hideFields: boolean
  readonly showError: (message: string) => void
}

export function createTaskEditorExtension({
  api,
  getToday,
  getFields,
  loadFields,
  hideFields,
  showError,
}: TaskEditorDeps): Extension {
  const source = createTaskCompletion({ getFields, getToday, load: loadFields })
  return [
    createTaskExtension({
      hideFields,
      showError,
      tick: async (request) => {
        const result = await api.setTaskDone({ ...request, today: getToday() })
        if (!result.ok) showError(WORDS.tickFailed)
        else if (result.value.error !== '') showError(WORDS.changedNote)
      },
    }),
    autocompletion({ addToOptions: [{ render: renderCompletionIcon, position: 20 }] }),
    completionTheme,
    EditorState.languageData.of(() => [{ autocomplete: source }]),
  ]
}
