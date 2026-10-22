/**
 * How a plugin's note tab reads its note and shows the failures of its writes.
 */
import type { Ref } from 'vue'
import type { PluginHostDeps } from './types'

const getErrorMessage = (err: unknown): string => (err instanceof Error ? err.message : String(err))

/**
 * A write error is shown until a newer write succeeds, and the failure of a write
 * that a newer write replaced is not shown.
 */
export const createNoteWriter = (
  at: string,
  writeNote: PluginHostDeps['writeNote'],
  errorMessage: Ref<string>,
): ((body: string) => void) => {
  let writes = 0
  let hasWriteError = false
  return (body) => {
    if (!writeNote) return
    const turn = ++writes
    writeNote(at, body).then(
      () => {
        if (turn !== writes || !hasWriteError) return
        hasWriteError = false
        errorMessage.value = ''
      },
      (err) => {
        if (turn !== writes) return
        hasWriteError = true
        errorMessage.value = getErrorMessage(err)
      },
    )
  }
}

export interface NoteReadState {
  readonly note: Ref<{ path: string; body: string }>
  readonly errorMessage: Ref<string>
  readonly isLoading: Ref<boolean>
  readonly hasEdited: () => boolean
}

type NoteRead = Awaited<ReturnType<NonNullable<PluginHostDeps['readNote']>>>

/** What a read answers, as the body and the error it holds. */
const splitRead = (result: NoteRead): { body?: string; error?: string } => {
  if (typeof result === 'string') return { body: result }
  return result ?? {}
}

/** The body and the error a read answered, put where the tab shows them. */
const applyRead = (targetPath: string, result: NoteRead, state: NoteReadState): void => {
  const { body, error } = splitRead(result)
  if (body !== undefined) state.note.value = { path: at, body }
  if (error) state.errorMessage.value = error
}

const finishRead = (targetPath: string, result: NoteRead, state: NoteReadState): void => {
  state.isLoading.value = false
  if (!state.hasEdited()) applyRead(at, result, state)
}

const failRead = (err: unknown, state: NoteReadState): void => {
  state.isLoading.value = false
  state.errorMessage.value = getErrorMessage(err)
}

/** The body the first read gives, unless the tab was edited while it was in flight. */
export const startNoteRead = (
  at: string,
  readNote: NonNullable<PluginHostDeps['readNote']>,
  state: NoteReadState,
): void => {
  readNote(at)
    .then((result) => finishRead(at, result, state))
    .catch((err) => failRead(err, state))
}
