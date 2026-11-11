import { reportFailure, runInBackground } from '@numen/ui'
import type { ConflictPrompt } from './flush'

/** Keeps the note as it is in the window, in the background. */
export function keepConflict(conflict: ConflictPrompt): void {
  runInBackground(conflict.keep(), reportFailure)
}

/** Takes the note as it is on disk, in the background. */
export function takeConflict(conflict: ConflictPrompt): void {
  runInBackground(conflict.take(), reportFailure)
}

/** Postpones the decision on the note. */
export function postponeConflict(conflict: ConflictPrompt): void {
  conflict.later()
}
