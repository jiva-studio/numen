/**
 * The change each open note is being shown, and when it stops being shown.
 *
 * A change is reported before the note holds it and ended once the write is
 * over, which is not the moment the text arrives. A change that has ended is
 * held until the note changes under it and dropped a moment later; one whose
 * text never arrives is dropped after `endedMs`.
 */
import type { NoteEdit } from '@/entities/note'

/** What is being drawn over one note. */
export interface Change {
  /** Whatever the change was reported by. Never read, only handed back. */
  readonly id: string
  /** The run being replaced, as offsets into the prose. */
  readonly from: number
  readonly to: number
  /** What is going in where that run stands. */
  readonly text: string
}

/** The two intervals a change that has ended waits on, in milliseconds. */
export interface ChangeLimits {
  /** How long a change stays after the note changed under it. */
  readonly noteChangedMs: number
  /** How long a change that has ended stays when its text never arrives. */
  readonly endedMs: number
  /** How long a change stays when it is never ended. */
  readonly unendedMs: number
}

export const CHANGE_LIMITS: ChangeLimits = { noteChangedMs: 900, endedMs: 4000, unendedMs: 15000 }

/** What should be done: the interval for one note, armed again over the last. */
export interface TimerRequest {
  readonly path: string
  readonly after: number
}

export function createChangeTracker(limits: ChangeLimits = CHANGE_LIMITS) {
  const changes = new Map<string, Change>()
  /** The notes whose change is over and is being let go of. */
  const endedPaths = new Set<string>()

  /** A change was reported. */
  const reportChange = (edit: NoteEdit): TimerRequest | null => {
    if (!edit.isComplete) {
      changes.set(edit.path, {
        id: edit.change,
        from: edit.span.from,
        to: edit.span.to,
        text: edit.text,
      })
      endedPaths.delete(edit.path)
      // A change is drawn on the word of whoever is making it, and that word
      // stops arriving when an agent is stopped mid-call. The `unendedMs` interval is what a
      // drawing nobody ends costs.
      return { path: edit.path, after: limits.unendedMs }
    }
    // A change nobody is drawing ends nothing.
    if (!changes.has(edit.path)) return null
    endedPaths.add(edit.path)
    return { path: edit.path, after: limits.endedMs }
  }

  /** The note changed under whatever is drawn over it. */
  const reportNoteChange = (path: string): TimerRequest | null => {
    if (!endedPaths.has(path)) return null
    return { path, after: limits.noteChangedMs }
  }

  /** The interval for one note fired. */
  const expireChange = (path: string): void => {
    changes.delete(path)
    endedPaths.delete(path)
  }

  /** A note the window is no longer showing. */
  const closeNote = (path: string): void => expireChange(path)

  /** What one note is drawn with, or nothing. */
  const getChange = (path: string): Change | null => changes.get(path) ?? null

  return { reportChange, reportNoteChange, expireChange, closeNote, getChange }
}
