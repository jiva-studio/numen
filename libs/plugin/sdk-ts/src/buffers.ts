import type { Caught } from './errors'
import type { Disposable } from './types'

// UnsavedBuffer is text a buffer could not write, and the two ways the person settles it.
export interface UnsavedBuffer {
  readonly name: string
  // Writes the text again over the file.
  keep(this: void): Promise<void>
  // Gives the text up and takes the file's version.
  take(this: void): Promise<void>
}

// UnsavedBuffersError is what FileBuffers.flush rejects with when text stays unwritten. The window
// stops the quit and shows the person each buffer in `unsaved` by its name, such as a file path, with
// the plugin it belongs to. The window fills in `pluginId`, so a plugin may leave it empty.
export class UnsavedBuffersError extends Error {
  readonly unsaved: readonly UnsavedBuffer[]
  readonly pluginId: string

  constructor(unsaved: readonly UnsavedBuffer[], pluginId = '') {
    super(`The text of ${unsaved.map((one) => one.name).join(', ')} was not written`)
    this.name = 'UnsavedBuffersError'
    this.unsaved = unsaved
    this.pluginId = pluginId
  }
}

// isUnsavedBuffersError tells the error of any copy of the SDK, since a plugin may bundle its own.
export function isUnsavedBuffersError(error: Caught): error is UnsavedBuffersError {
  return error instanceof Error && 'unsaved' in error && Array.isArray(error.unsaved)
}

// FileBuffers are the open files of one kind a plugin's tabs edit.
export interface FileBuffers {
  // Writes the unwritten text. It rejects with an UnsavedBuffersError when text stays unwritten, so the
  // window does not quit on it and shows the person what is unsaved.
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
