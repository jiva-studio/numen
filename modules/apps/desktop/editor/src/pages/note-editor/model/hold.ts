/**
 * The change each open note is being shown, and when it stops being shown.
 *
 * A change is reported before the note holds it and ended once the write is
 * over, which is not the moment the text arrives. A change that has ended is
 * held until the note changes under it and dropped a moment later; one whose
 * text never arrives is dropped on the longer bound.
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
export interface HoldLimits {
  /** How long a change stays after the note changed under it. */
  readonly settle: number
  /** How long a change whose text never arrived stays at all. */
  readonly bound: number
  /** How long a change stays when nothing more is said about it. */
  readonly abandoned: number
}

export const holding: HoldLimits = { settle: 900, bound: 4000, abandoned: 15000 }

/** What should be done: the interval for one note, armed again over the last. */
export interface TimerRequest {
  readonly path: string
  readonly after: number
}

export function holdChanges(limits: HoldLimits = holding) {
  const changes = new Map<string, Change>()
  /** The notes whose change is over and is being let go of. */
  const ending = new Set<string>()

  /** A change was reported. */
  const reportChange = (said: NoteEdit): TimerRequest | null => {
    if (!said.isComplete) {
      changes.set(said.path, {
        id: said.change,
        from: said.span.from,
        to: said.span.to,
        text: said.text,
      })
      ending.delete(said.path)
      // A change is drawn on the word of whoever is making it, and that word
      // stops arriving when an agent is stopped mid-call. The bound is what a
      // drawing nobody ends costs.
      return { path: said.path, after: limits.abandoned }
    }
    // A change nobody is drawing ends nothing.
    if (!changes.has(said.path)) return null
    ending.add(said.path)
    return { path: said.path, after: limits.bound }
  }

  /** The note changed under whatever is drawn over it. */
  const handleNoteChange = (path: string): TimerRequest | null => {
    if (!ending.has(path)) return null
    return { path, after: limits.settle }
  }

  /** The interval for one note fired. */
  const handleTimeout = (path: string): void => {
    changes.delete(path)
    ending.delete(path)
  }

  /** A note the window is no longer showing. */
  const shut = (path: string): void => handleTimeout(path)

  /** What one note is drawn with, or nothing. */
  const getChange = (path: string): Change | null => changes.get(path) ?? null

  return { reportChange, handleNoteChange, handleTimeout, shut, getChange }
}
