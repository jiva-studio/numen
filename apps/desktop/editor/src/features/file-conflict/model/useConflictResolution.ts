/**
 * The resolutions the prompt for unsaved notes offers: keep the note, take the one on disk, or
 * decide later. Keeping and taking run in the background.
 */
import { reportFailure, runInBackground } from '@numen/ui'
import type { ConflictPrompt } from './flush'

export function useConflictResolution() {
  function keep(conflict: ConflictPrompt): void {
    runInBackground(conflict.keep(), reportFailure)
  }

  function take(conflict: ConflictPrompt): void {
    runInBackground(conflict.take(), reportFailure)
  }

  function later(conflict: ConflictPrompt): void {
    conflict.later()
  }

  return { keep, take, later }
}
