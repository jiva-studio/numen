import { autocompletion } from '@codemirror/autocomplete'
import { EditorState, type Extension } from '@codemirror/state'
import type { TodosApi } from '../api/wire'
import { WORDS } from '../words'
import type { ListFieldsResponse } from '../types'
import { createTaskCompletion } from './taskCompletion'
import { createTaskExtension } from './taskExtension'

/** The id the manifest declares the editor extension under. */
export const TASKS_EXTENSION = 'tasks'

export interface TaskEditorDeps {
  readonly api: TodosApi
  readonly getToday: () => string
  readonly getFields: () => ListFieldsResponse
  readonly showError: (message: string) => void
}

export function createTaskEditorExtension({
  api,
  getToday,
  getFields,
  showError,
}: TaskEditorDeps): Extension {
  return [
    createTaskExtension({
      tick: async (request) => {
        const result = await api.setTaskDone({ ...request, today: getToday() })
        if (!result.ok) showError(WORDS.tickFailed)
        else if (result.value.error !== '') showError(WORDS.changedNote)
      },
    }),
    autocompletion(),
    EditorState.languageData.of(() => [
      { autocomplete: createTaskCompletion({ getFields, getToday }) },
    ]),
  ]
}
