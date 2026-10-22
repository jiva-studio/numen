import type { Disposable } from './types'
import type { openNotes } from './notes'

// FileBuffers are the open files of one kind a plugin's tabs edit.
export type FileBuffers = ReturnType<typeof openNotes>

// BufferHost carries what a window does with the files its tabs hold open, when it quits.
export interface BufferHost {
  // What the buffers hold unwritten is written when the window quits, and a file that moved under
  // one is put to the person before the window goes.
  track(buffers: FileBuffers): Disposable
  // Work the window finishes before it quits.
  addFlushHandler(handler: () => Promise<unknown>): Disposable
}

export type FileConflict = 'gone' | 'stale' | null

// parseFileConflict says which conflict a state names, and nothing for every other state.
export function parseFileConflict(state: string): FileConflict {
  return state === 'gone' || state === 'stale' ? state : null
}
