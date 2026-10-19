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
export function createNoteWriter(
  at: string,
  writeNote: PluginHostDeps['writeNote'],
  errorMessage: Ref<string>,
): (body: string) => void {
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

/** The body the first read gives, unless the tab was edited while it was in flight. */
export function startNoteRead(
  at: string,
  readNote: NonNullable<PluginHostDeps['readNote']>,
  state: NoteReadState,
): void {
  readNote(at)
    .then((result) => {
      state.isLoading.value = false
      if (state.hasEdited()) return
      if (typeof result === 'string') {
        state.note.value = { path: at, body: result }
      } else if (result && typeof result === 'object') {
        if (result.body !== undefined) state.note.value = { path: at, body: result.body }
        if (result.error) state.errorMessage.value = result.error
      }
    })
    .catch((err) => {
      state.isLoading.value = false
      state.errorMessage.value = getErrorMessage(err)
    })
}
