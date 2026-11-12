import type { Disposable } from './types'

// FileBuffers are the open files of one kind a plugin's tabs edit.
export interface FileBuffers {
  // Writes the unwritten text. It rejects when text stays unwritten, so the window does not quit on it.
  flush(this: void): Promise<void>
}

// BufferHost carries what a window does with the files its tabs hold open, when it quits.
export interface BufferHost {
  // The unwritten buffer contents are written when the window quits, and a file that changed on disk under
  // a buffer is reported to the user before the window closes.
  track(buffers: FileBuffers): Disposable
  // Work the window finishes before it quits.
  addFlushHandler(handler: () => Promise<void>): Disposable
}

export type FileConflict = 'gone' | 'stale' | null
