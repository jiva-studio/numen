import { reportFailure, runInBackground } from '@numen/ui'
import type { Core } from '@/entities/core'
import { useNote } from './useNote'

export interface NoteSheetDeps {
  readonly core: Core
  readonly path: string
  readonly focusEditor: () => void
  readonly emitError: (message: string) => void
  readonly emitClose: () => void
}

/**
 * The note of the sheet: reading it when the sheet opens, which focuses the editor or reports the
 * error and closes, and writing it, which closes the sheet or reports the error.
 */
export function useNoteSheet(deps: NoteSheetDeps) {
  const { prose, isReading, read, keep } = useNote(deps.core, deps.path)

  async function open(): Promise<void> {
    const answer = await read()
    if (!answer.ok) {
      deps.emitError(answer.error)
      deps.emitClose()
      return
    }
    deps.focusEditor()
  }

  async function save(): Promise<void> {
    const answer = await keep()
    if (!answer.ok) {
      deps.emitError(answer.error)
      return
    }
    deps.emitClose()
  }

  return {
    prose,
    isReading,
    open: () => runInBackground(open(), reportFailure),
    save: () => runInBackground(save(), reportFailure),
  }
}
