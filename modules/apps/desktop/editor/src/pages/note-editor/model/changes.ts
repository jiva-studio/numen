/**
 * The changes the window is drawing, and the intervals they are let go of on.
 *
 * `changeTracker.ts` decides which changes stand; this runs the timers and
 * keeps the changes where the template can draw them.
 */
import { ref } from 'vue'
import type { NoteEdit } from '@/entities/note'
import {
  createChangeTracker,
  CHANGE_LIMITS,
  type Change,
  type ChangeLimits,
  type TimerRequest,
} from './changeTracker'

/** Every change in flight, filed by the note it stands on. */
export function createNoteChanges(limits: ChangeLimits = CHANGE_LIMITS) {
  const tracker = createChangeTracker(limits)
  /** What each note is drawn with, which Vue reads to draw it. */
  const changes = ref(new Map<string, Change>())
  /** The interval each note is waiting on, so arming again replaces it. */
  const timers = new Map<string, ReturnType<typeof setTimeout>>()

  const syncChange = (path: string, request: TimerRequest | null): void => {
    const change = tracker.getChange(path)
    if (change) changes.value.set(path, change)
    else changes.value.delete(path)
    if (request) scheduleExpiry(request)
  }

  function scheduleExpiry(request: TimerRequest): void {
    clearTimeout(timers.get(request.path))
    timers.set(
      request.path,
      setTimeout(() => {
        timers.delete(request.path)
        tracker.expireChange(request.path)
        syncChange(request.path, null)
      }, request.after),
    )
  }

  /** A change was reported. */
  const reportChange = (edit: NoteEdit): void => syncChange(edit.path, tracker.reportChange(edit))

  /** The note changed under whatever is drawn over it. */
  const reportNoteChange = (path: string): void => syncChange(path, tracker.reportNoteChange(path))

  /** A note the window is no longer showing. */
  const closeNote = (path: string): void => {
    clearTimeout(timers.get(path))
    timers.delete(path)
    tracker.closeNote(path)
    syncChange(path, null)
  }

  /** What one note is drawn with, or nothing. */
  const getChange = (path: string): Change | null => changes.value.get(path) ?? null

  /** Every interval is let go of, for a window that is going. */
  const close = (): void => {
    for (const timer of timers.values()) clearTimeout(timer)
    timers.clear()
  }

  return { reportChange, reportNoteChange, closeNote, getChange, close }
}
