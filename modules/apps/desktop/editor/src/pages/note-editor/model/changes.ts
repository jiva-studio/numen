/**
 * The changes the window is drawing, and the intervals they are let go of on.
 *
 * `hold.ts` decides; this carries out what it decides — the intervals, and
 * holding the answers where the template can draw them.
 */
import { ref } from 'vue'
import type { NoteEdit } from '@/entities/note'
import { holdChanges, HOLD_LIMITS, type Change, type HoldLimits, type TimerRequest } from './hold'

/** Every change in flight, filed by the note it stands on. */
export function noteChanges(limits: HoldLimits = HOLD_LIMITS) {
  const decided = holdChanges(limits)
  /** What each note is drawn with, which Vue reads to draw it. */
  const changes = ref(new Map<string, Change>())
  /** The interval each note is waiting on, so arming again replaces it. */
  const timers = new Map<string, ReturnType<typeof setTimeout>>()

  const carry = (path: string, arm: TimerRequest | null): void => {
    const change = decided.getChange(path)
    if (change) changes.value.set(path, change)
    else changes.value.delete(path)
    if (arm) hold(arm)
  }

  function hold(arm: TimerRequest): void {
    clearTimeout(timers.get(arm.path))
    timers.set(
      arm.path,
      setTimeout(() => {
        timers.delete(arm.path)
        decided.handleTimeout(arm.path)
        carry(arm.path, null)
      }, arm.after),
    )
  }

  /** A change was reported. */
  const reportChange = (edit: NoteEdit): void => carry(edit.path, decided.reportChange(edit))

  /** The note changed under whatever is drawn over it. */
  const handleNoteChange = (path: string): void => carry(path, decided.handleNoteChange(path))

  /** A note the window is no longer showing. */
  const closeNote = (path: string): void => {
    clearTimeout(timers.get(path))
    timers.delete(path)
    decided.closeNote(path)
    carry(path, null)
  }

  /** What one note is drawn with, or nothing. */
  const getChange = (path: string): Change | null => changes.value.get(path) ?? null

  /** Every interval is let go of, for a window that is going. */
  const close = (): void => {
    for (const timer of timers.values()) clearTimeout(timer)
    timers.clear()
  }

  return { reportChange, handleNoteChange, closeNote, getChange, close }
}
